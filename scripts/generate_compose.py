#!/usr/bin/env python3
"""Generate a Docker Compose stack with parametric number of oracles and funded seed-derived oracle keys.

* docker-compose.generated.yml is written as a new file.
* generated/hardhat.config.generated.js is written as a new file.
* generated/deployContracts.generated.js is written as a trace/debug artifact.
* the chain service mounts the generated Hardhat config.
* the chain service passes CONFIG_DIGEST to chain/scripts/deployContracts.js.
* oracle services mount oracle/setup_network_parametric.sh at runtime.

Usage:
    python3 scripts/generate_compose.py --num-oracles 7 --network-seed 42
    docker compose -f docker-compose.generated.yml --env-file .env up --build
"""

from __future__ import annotations

import argparse
import hashlib
import ipaddress
import json
import os
import re
import select
import subprocess
import time
import uuid
from pathlib import Path
from typing import Iterable


SECP256K1_ORDER = int(
    "fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141",
    16,
)

# Hardhat default account #0. Keeping the deployer stable preserves the usual
# local deployment addresses in .env for OracleQueue and OracleVerifier.
DEFAULT_DEPLOYER_PRIVATE_KEY = (
    "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
)

DEFAULT_BALANCE_WEI = "10000000000000000000000"  # 10000 ETH
DEFAULT_LOCATIONS = "Milan,Toronto,Moscow,Lisbon,Mumbai,Johannesburg,NewYork"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Generate docker-compose.generated.yml and a mounted Hardhat config "
            "for N seed-keyed oracle containers."
        )
    )
    parser.add_argument(
        "positionals",
        nargs="*",
        help="Optional positional form: NUM_ORACLES NETWORK_SEED",
    )
    parser.add_argument("-n", "--num-oracles", type=int)
    parser.add_argument("-s", "--network-seed", "--seed", type=int)
    parser.add_argument(
        "--key-seed",
        help="Seed used for oracle Ethereum keys. Defaults to --network-seed.",
    )
    parser.add_argument(
        "--ocr-seed",
        type=int,
        default=1,
        help="Seed passed to the Go OCR process for off-chain identity derivation.",
    )
    parser.add_argument(
        "-f",
        "--fault-tolerance",
        type=int,
        help="OCR fault tolerance. Defaults to floor((NUM_ORACLES - 1) / 3).",
    )
    parser.add_argument(
        "--locations",
        default=DEFAULT_LOCATIONS,
        help=(
            "Comma-separated location pool used by setup_network_parametric.sh. "
            f"Default: {DEFAULT_LOCATIONS}"
        ),
    )
    parser.add_argument(
        "--out",
        default="docker-compose.generated.yml",
        help="Compose output path, relative to the repository root by default.",
    )
    parser.add_argument(
        "--hardhat-config",
        default="generated/hardhat.config.generated.js",
        help="Generated Hardhat config path, relative to the repository root by default.",
    )
    parser.add_argument(
        "--deploy-script",
        default="generated/deployContracts.generated.js",
        help="Generated deploy script path, relative to the repository root by default.",
    )
    parser.add_argument(
        "--config-digest",
        help=(
            "OCR config digest to inject into the generated deploy script. "
            "When omitted, it is computed with the oracle Go code."
        ),
    )
    parser.add_argument(
        "--digest-cache",
        default="generated/config-digests.json",
        help=(
            "JSON cache for computed config digests, relative to the repository "
            "root by default."
        ),
    )
    parser.add_argument(
        "--digest-runner",
        choices=("auto", "local", "docker"),
        default="auto",
        help=(
            "How to run the Go digest helper. auto uses local Go >= 1.23 when "
            "available, otherwise Docker with golang:1.24-alpine."
        ),
    )
    parser.add_argument(
        "--digest-timeout-seconds",
        type=int,
        default=900,
        help="Timeout for computing the OCR config digest.",
    )
    parser.add_argument(
        "--env-file",
        default=".env",
        help="Env file referenced by generated Compose, relative to the repository root by default.",
    )
    parser.add_argument("--subnet", default="10.5.0.0/16")
    parser.add_argument("--oracle-base-ip", default="10.5.0.20")
    parser.add_argument("--chain-ip", default="10.5.0.10")
    parser.add_argument("--ipfs-ip", default="10.5.0.11")
    parser.add_argument("--bootstrap-ip", default="10.5.0.12")
    parser.add_argument(
        "--deployer-private-key",
        default=DEFAULT_DEPLOYER_PRIVATE_KEY,
        help="Funded deployer key for Hardhat account #0.",
    )
    parser.add_argument(
        "--account-balance-wei",
        default=DEFAULT_BALANCE_WEI,
        help="Wei balance assigned to the deployer and each generated oracle key.",
    )
    latency_group = parser.add_mutually_exclusive_group()
    latency_group.add_argument(
        "--enable-latency",
        action="store_true",
        dest="enable_latency",
        default=True,
    )
    latency_group.add_argument(
        "--disable-latency",
        action="store_false",
        dest="enable_latency",
    )

    args = parser.parse_args()

    if len(args.positionals) > 2:
        parser.error("expected at most two positional arguments: NUM_ORACLES NETWORK_SEED")

    if args.num_oracles is None and args.positionals:
        args.num_oracles = int(args.positionals[0])

    if args.network_seed is None and len(args.positionals) >= 2:
        args.network_seed = int(args.positionals[1])

    if args.num_oracles is None or args.network_seed is None:
        parser.error("NUM_ORACLES and NETWORK_SEED are required")

    if args.num_oracles < 1:
        parser.error("NUM_ORACLES must be >= 1")

    if args.fault_tolerance is None:
        args.fault_tolerance = (args.num_oracles - 1) // 3

    if args.fault_tolerance < 0:
        parser.error("--fault-tolerance must be >= 0")

    if args.key_seed is None:
        args.key_seed = str(args.network_seed)

    return args


def resolve_repo_path(repo_root: Path, value: str) -> Path:
    path = Path(value)
    if path.is_absolute():
        return path
    return repo_root / path


def compose_path(compose_dir: Path, path: Path) -> str:
    rel = os.path.relpath(path, compose_dir).replace(os.sep, "/")
    if not rel.startswith("."):
        rel = f"./{rel}"
    return rel


def display_path(repo_root: Path, path: Path) -> str:
    try:
        return str(path.relative_to(repo_root))
    except ValueError:
        return str(path)


def yaml_quote(value: object) -> str:
    text = str(value)
    return '"' + text.replace("\\", "\\\\").replace('"', '\\"') + '"'


def normalize_config_digest(digest: str) -> str:
    value = digest.strip()
    if value.startswith(("0x", "0X")):
        value = value[2:]

    if not re.fullmatch(r"[0-9a-fA-F]{64}", value):
        raise SystemExit(
            "--config-digest must be a 32-byte hex value, with or without 0x prefix"
        )

    return "0x" + value.lower()


def subprocess_output_text(output: object) -> str:
    if output is None:
        return ""
    if isinstance(output, bytes):
        return output.decode("utf-8", errors="replace")
    return str(output)


def derive_oracle_private_key(key_seed: str, oracle_index: int, used: set[str]) -> str:
    counter = 0
    while True:
        material = f"oracle-private-key|{key_seed}|{oracle_index}|{counter}".encode()
        digest = hashlib.sha256(material).digest()
        value = int.from_bytes(digest, byteorder="big") % (SECP256K1_ORDER - 1) + 1
        private_key = f"{value:064x}"
        if private_key not in used:
            used.add(private_key)
            return private_key
        counter += 1


def generate_oracle_ips(base_ip: str, count: int) -> list[str]:
    base = ipaddress.ip_address(base_ip)
    return [str(base + i) for i in range(count)]


def sha256_text(value: str) -> str:
    return hashlib.sha256(value.encode("utf-8")).hexdigest()


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as file:
        for chunk in iter(lambda: file.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def digest_cache_source_hash(repo_root: Path) -> str:
    paths = [
        repo_root / "oracle" / "cmd" / "configdigest" / "main.go",
        repo_root / "oracle" / "go.mod",
        repo_root / "oracle" / "go.sum",
    ]

    digest = hashlib.sha256()
    for path in paths:
        if path.exists():
            digest.update(path.relative_to(repo_root).as_posix().encode("utf-8"))
            digest.update(b"\0")
            digest.update(sha256_file(path).encode("utf-8"))
            digest.update(b"\0")
    return digest.hexdigest()


def digest_cache_key(
    repo_root: Path,
    args: argparse.Namespace,
    oracle_private_keys: list[str],
) -> tuple[str, dict[str, object]]:
    material = {
        "version": 1,
        "num_oracles": args.num_oracles,
        "fault_tolerance": args.fault_tolerance,
        "ocr_seed": args.ocr_seed,
        "network_seed": args.network_seed,
        "key_seed": str(args.key_seed),
        "oracle_private_keys_sha256": sha256_text("\n".join(oracle_private_keys)),
        "digest_source_sha256": digest_cache_source_hash(repo_root),
    }
    encoded = json.dumps(material, sort_keys=True, separators=(",", ":"))
    return sha256_text(encoded), material


def load_digest_cache(cache_path: Path) -> dict[str, object]:
    if not cache_path.exists():
        return {"version": 1, "entries": {}}

    try:
        data = json.loads(cache_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise SystemExit(f"invalid digest cache JSON at {cache_path}: {exc}") from exc

    if not isinstance(data, dict):
        raise SystemExit(f"invalid digest cache at {cache_path}: expected object")

    if data.get("version") != 1 or not isinstance(data.get("entries"), dict):
        return {"version": 1, "entries": {}}

    return data


def store_digest_cache(cache_path: Path, data: dict[str, object]) -> None:
    cache_path.parent.mkdir(parents=True, exist_ok=True)
    cache_path.write_text(
        json.dumps(data, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )


def command_block(items: Iterable[str], indent: int = 4) -> list[str]:
    pad = " " * indent
    lines = [f"{pad}command:"]
    for item in items:
        lines.append(f"{pad}  - {yaml_quote(item)}")
    return lines


def environment_block(mapping: dict[str, str], indent: int = 4) -> list[str]:
    pad = " " * indent
    return [f"{pad}{key}: {yaml_quote(value)}" for key, value in mapping.items()]


def render_compose(
    args: argparse.Namespace,
    repo_root: Path,
    output_path: Path,
    hardhat_config_path: Path,
    deploy_script_path: Path,
    env_file_path: Path,
    oracle_private_keys: list[str],
    oracle_ips: list[str],
    config_digest: str,
) -> str:
    compose_dir = output_path.parent
    chain_context = compose_path(compose_dir, repo_root / "chain")
    oracle_context = compose_path(compose_dir, repo_root / "oracle")
    latency_script = compose_path(
        compose_dir, repo_root / "oracle" / "setup_network_parametric.sh"
    )
    hardhat_config = compose_path(compose_dir, hardhat_config_path)
    env_file = compose_path(compose_dir, env_file_path)
    ipfs_data = compose_path(compose_dir, repo_root / "IpfsAgent" / "ipfs_data")
    ipfs_staging = compose_path(compose_dir, repo_root / "IpfsAgent" / "ipfs_staging")

    oracle_ips_csv = ",".join(oracle_ips)
    enable_latency = "true" if args.enable_latency else "false"

    lines: list[str] = [
        "# Generated by scripts/generate_compose.py. Do not edit by hand.",
        f"# NUM_ORACLES={args.num_oracles}",
        f"# NETWORK_SEED={args.network_seed}",
        f"# KEY_SEED={args.key_seed}",
        "# The chain service mounts a generated Hardhat config so the seed-derived",
        "# oracle accounts are funded when npx hardhat node starts.",
        "# The chain service passes CONFIG_DIGEST to chain/scripts/deployContracts.js.",
    ]

    lines.extend(
        [
            "",
            "x-generated-oracle-keys: &generated-oracle-keys",
        ]
    )

    for idx, private_key in enumerate(oracle_private_keys):
        lines.append(f"  ORACLE{idx}_PRIVATE_KEY: {yaml_quote(private_key)}")

    common_env = {
        "CHAIN_RPC": "${CHAIN_RPC_URL}",
        "VERIFIER_ADDRESS": "${VERIFIER_ADDRESS}",
        "QUEUE_ADDRESS": "${QUEUE_ADDRESS}",
        "QUEUE_CONTRACT_ADDRESS": "${QUEUE_ADDRESS}",
        "IPFS_API_URL": "${DOCKER_IPFS_URL}",
        "MALICIOUS_MODE": "false",
        "NUM_ORACLES": str(args.num_oracles),
        "FAULT_TOLERANCE": str(args.fault_tolerance),
        "OCR_SEED": str(args.ocr_seed),
        "NETWORK_SEED": str(args.network_seed),
        "NETWORK_LOCATIONS": args.locations,
        "ORACLE_IPS": oracle_ips_csv,
        "ENABLE_LATENCY": enable_latency,
    }

    lines.extend(
        [
            "",
            "x-oracle-common-env: &oracle-common-env",
            "  <<: *generated-oracle-keys",
        ]
    )
    lines.extend(environment_block(common_env, indent=2))

    lines.extend(
        [
            "",
            "services:",
            "  chain:",
            "    container_name: chain",
            f"    build: {yaml_quote(chain_context)}",
            "    ports:",
            '      - "8545:8545"',
            "    networks:",
            "      ocr-net:",
            f"        ipv4_address: {args.chain_ip}",
            "    env_file:",
            f"      - {yaml_quote(env_file)}",
            "    environment:",
            f"      NUM_ORACLES: {yaml_quote(args.num_oracles)}",
            f"      FAULT_TOLERANCE: {yaml_quote(args.fault_tolerance)}",
            f"      CONFIG_DIGEST: {yaml_quote(config_digest)}",
            "    volumes:",
            f"      - {yaml_quote(f'{hardhat_config}:/app/hardhat.config.js:ro')}",
            "",
            "  ipfs:",
            "    image: ipfs/kubo:latest",
            "    container_name: ipfs_node",
            "    restart: always",
            "    ports:",
            '      - "4001:4001"',
            '      - "5001:5001"',
            '      - "8080:8080"',
            "    environment:",
            "      IPFS_PROFILE: server",
            "    volumes:",
            f"      - {yaml_quote(f'{ipfs_data}:/data/ipfs')}",
            f"      - {yaml_quote(f'{ipfs_staging}:/export')}",
            "    networks:",
            "      ocr-net:",
            f"        ipv4_address: {args.ipfs_ip}",
            "",
            "  bootstrap:",
            f"    build: {yaml_quote(oracle_context)}",
            "    env_file:",
            f"      - {yaml_quote(env_file)}",
            "    environment:",
            "      <<: *oracle-common-env",
        ]
    )

    lines.extend(
        command_block(
            [
                "oracle",
                "-mode",
                "bootstrap",
                "-n",
                str(args.num_oracles),
                "-f",
                str(args.fault_tolerance),
                "-seed",
                str(args.ocr_seed),
                "-bootstrap_listen",
                "0.0.0.0:${BOOTSTRAP_PORT}",
                "-bootstrap_announce",
                "${BOOTSTRAP_ADDR}",
            ],
            indent=4,
        )
    )

    lines.extend(
        [
            "    ports:",
            '      - "19900:19900"',
            "    networks:",
            "      ocr-net:",
            f"        ipv4_address: {args.bootstrap_ip}",
            "    depends_on:",
            "      chain:",
            "        condition: service_started",
        ]
    )

    for idx, (private_key, ip_address) in enumerate(zip(oracle_private_keys, oracle_ips)):
        service_name = f"oracle{idx}"
        lines.extend(
            [
                "",
                f"  {service_name}:",
                f"    container_name: node_oracle{idx}",
                f"    hostname: oracle{idx}",
                f"    build: {yaml_quote(oracle_context)}",
                "    env_file:",
                f"      - {yaml_quote(env_file)}",
                "    extra_hosts:",
                '      - "host.docker.internal:host-gateway"',
                "    cap_add:",
                "      - NET_ADMIN",
                "    entrypoint:",
                '      - "/bin/sh"',
                '      - "/usr/local/bin/setup_network_parametric.sh"',
                "    environment:",
                "      <<: *oracle-common-env",
                f"      PRIVATE_KEY: {yaml_quote(private_key)}",
                f"      ORACLE_ID: {yaml_quote(idx)}",
                "    volumes:",
                f"      - {yaml_quote(f'{latency_script}:/usr/local/bin/setup_network_parametric.sh:ro')}",
            ]
        )

        lines.extend(
            command_block(
                [
                    "oracle",
                    "-mode",
                    "oracle",
                    "-n",
                    str(args.num_oracles),
                    "-f",
                    str(args.fault_tolerance),
                    "-oracle_id",
                    str(idx),
                    "-seed",
                    str(args.ocr_seed),
                    "-bootstrap_addr",
                    "${BOOTSTRAP_ADDR}",
                    "-p2p_listen",
                    "0.0.0.0:${ORACLE_P2P_PORT}",
                    "-p2p_announce",
                    f"oracle{idx}:${{ORACLE_P2P_PORT}}",
                ],
                indent=4,
            )
        )

        lines.extend(
            [
                "    networks:",
                "      ocr-net:",
                f"        ipv4_address: {ip_address}",
                "    depends_on:",
                "      - chain",
                "      - bootstrap",
            ]
        )

    lines.extend(
        [
            "",
            "networks:",
            "  ocr-net:",
            "    driver: bridge",
            "    ipam:",
            "      config:",
            f"        - subnet: {args.subnet}",
            "",
        ]
    )

    return "\n".join(lines)


def render_hardhat_config(
    args: argparse.Namespace,
    deployer_private_key: str,
    oracle_private_keys: list[str],
) -> str:
    account_lines = [
        "const accounts = [",
        "  // Account #0: stable Hardhat deployer/model creator.",
        (
            '  { privateKey: "0x'
            + deployer_private_key
            + '", balance: "'
            + args.account_balance_wei
            + '" },'
        ),
        "  // Accounts #1..N: seed-derived oracle accounts funded by Hardhat.",
    ]

    for private_key in oracle_private_keys:
        account_lines.append(
            '  { privateKey: "0x'
            + private_key
            + '", balance: "'
            + args.account_balance_wei
            + '" },'
        )

    account_lines.append("];")

    return "\n".join(
        [
            "// Generated by scripts/generate_compose.py. Do not edit by hand.",
            f"// NUM_ORACLES={args.num_oracles}",
            f"// NETWORK_SEED={args.network_seed}",
            f"// KEY_SEED={args.key_seed}",
            'require("@nomicfoundation/hardhat-ethers");',
            'require("dotenv").config({ path: "../.env" });',
            "",
            *account_lines,
            "",
            "/** @type import('hardhat/config').HardhatUserConfig */",
            "module.exports = {",
            '  solidity: "0.8.24",',
            "  networks: {",
            "    hardhat: {",
            "      chainId: Number(process.env.CHAIN_ID || 31337),",
            "      accounts,",
            "    },",
            "    localhost: {",
            '      url: "http://127.0.0.1:8545",',
            "    },",
            "    docker: {",
            '      url: process.env.CHAIN_RPC_URL || "http://chain:8545",',
            "    },",
            "  },",
            "};",
            "",
        ]
    )


def render_deploy_script(args: argparse.Namespace, config_digest: str) -> str:
    return "\n".join(
        [
            "// Generated by scripts/generate_compose.py. Do not edit by hand.",
            f"// NUM_ORACLES={args.num_oracles}",
            f"// CONFIG_DIGEST={config_digest}",
            'const hre = require("hardhat");',
            "",
            "async function main() {",
            '  console.log("\\n=======================================================");',
            '  console.log(" STARTING GENERATED DEPLOYMENT: OracleQueue & OracleVerifier");',
            '  console.log("=======================================================\\n");',
            "",
            "  const [deployer] = await hre.ethers.getSigners();",
            "  console.log(`Deploying contracts with the account: ${deployer.address}`);",
            "",
            '  console.log("\\n[1/3] Deploying OracleQueue...");',
            '  const OracleQueue = await hre.ethers.getContractFactory("OracleQueue");',
            '  const feeInWei = hre.ethers.parseEther("0.02");',
            '  const rewardInWei = hre.ethers.parseEther("0.018");',
            "  const queue = await OracleQueue.deploy(feeInWei, rewardInWei);",
            "  await queue.waitForDeployment();",
            "  const queueAddress = await queue.getAddress();",
            "  console.log(`OracleQueue deployed at: ${queueAddress}`);",
            "",
            '  console.log("\\n[2/3] Deploying OracleVerifier...");',
            f'  const NUM_ORACLES = parseInt(process.env.NUM_ORACLES || "{args.num_oracles}", 10);',
            f'  const REAL_DIGEST = process.env.CONFIG_DIGEST || "{config_digest}";',
            "  const signers = await hre.ethers.getSigners();",
            "  const modelCreator = signers[0];",
            "  const oraclesArray = [];",
            "  for (let i = 1; i <= NUM_ORACLES; i++) {",
            "    if (!signers[i]) {",
            '      throw new Error(`Missing funded Hardhat signer for oracle index ${i - 1}`);',
            "    }",
            "    oraclesArray.push(signers[i].address);",
            "  }",
            f'  const fValue = parseInt(process.env.FAULT_TOLERANCE || "{args.fault_tolerance}", 10);',
            "  console.log(`Deploying for a ${NUM_ORACLES}-node network (f=${fValue})...`);",
            "  console.log(`CONFIG_DIGEST=${REAL_DIGEST}`);",
            '  console.log("- ModelCreator (Deployer):", modelCreator.address);',
            '  console.log("- Oracles Array:", oraclesArray);',
            "",
            '  const OracleVerifier = await hre.ethers.getContractFactory("OracleVerifier", modelCreator);',
            "  const verifier = await OracleVerifier.deploy(oraclesArray, fValue, REAL_DIGEST, queueAddress);",
            "  await verifier.waitForDeployment();",
            "  const verifierAddress = await verifier.getAddress();",
            "  console.log(`OracleVerifier deployed at: ${verifierAddress}`);",
            "",
            '  console.log("\\n[3/3] Authorizing Verifier in the Queue...");',
            "  const authTx = await queue.setVerifierAddress(verifierAddress);",
            "  await authTx.wait();",
            '  console.log("Authorization complete! Queue now trusts Verifier.");',
            "",
            '  console.log("\\n=======================================================");',
            '  console.log(" DEPLOYMENT FINISHED SUCCESSFULLY!");',
            "  console.log(` QUEUE_ADDRESS=${queueAddress}`);",
            "  console.log(` VERIFIER_ADDRESS=${verifierAddress}`);",
            "  console.log(` CONFIG_DIGEST=${REAL_DIGEST}`);",
            '  console.log("=======================================================\\n");',
            "}",
            "",
            "main().catch((error) => {",
            "  console.error(error);",
            "  process.exitCode = 1;",
            "});",
            "",
        ]
    )


def compute_config_digest(
    repo_root: Path,
    args: argparse.Namespace,
    oracle_private_keys: list[str],
    digest_cache_path: Path,
) -> str:
    cache_key, cache_material = digest_cache_key(repo_root, args, oracle_private_keys)
    cache = load_digest_cache(digest_cache_path)
    entries = cache["entries"]

    if args.config_digest:
        digest = normalize_config_digest(args.config_digest)
        entries[cache_key] = {
            **cache_material,
            "config_digest": digest,
            "source": "explicit",
        }
        store_digest_cache(digest_cache_path, cache)
        return digest

    cached = entries.get(cache_key)
    if isinstance(cached, dict):
        cached_digest = cached.get("config_digest")
        if isinstance(cached_digest, str):
            cached_digest = normalize_config_digest(cached_digest)
            print(
                f"Using cached CONFIG_DIGEST={cached_digest} "
                f"from {display_path(repo_root, digest_cache_path)}",
                flush=True,
            )
            return cached_digest

    env = os.environ.copy()
    env["DIGEST_NUM_ORACLES"] = str(args.num_oracles)
    env["DIGEST_FAULT_TOLERANCE"] = str(args.fault_tolerance)
    env["DIGEST_OCR_SEED"] = str(args.ocr_seed)
    for idx, private_key in enumerate(oracle_private_keys):
        env[f"ORACLE{idx}_PRIVATE_KEY"] = private_key

    completed = run_digest_helper(repo_root, args, env)

    match = re.search(r"CONFIG_DIGEST=(0x)?([0-9a-fA-F]{64})", completed.stdout)
    if not match:
        raise SystemExit(
            "digest helper completed but did not print CONFIG_DIGEST. Output:\n"
            + completed.stdout
        )

    digest = normalize_config_digest(match.group(0).split("=", 1)[1])
    entries[cache_key] = {
        **cache_material,
        "config_digest": digest,
        "source": args.digest_runner,
    }
    store_digest_cache(digest_cache_path, cache)
    return digest


def go_version_is_new_enough() -> bool:
    try:
        completed = subprocess.run(
            ["go", "version"],
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            check=True,
        )
    except (FileNotFoundError, subprocess.CalledProcessError):
        return False

    match = re.search(r"go(\d+)\.(\d+)", completed.stdout)
    if not match:
        return False

    major = int(match.group(1))
    minor = int(match.group(2))
    return major > 1 or (major == 1 and minor >= 23)


def run_local_digest_helper(
    repo_root: Path,
    env: dict[str, str],
    timeout_seconds: int,
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [
            "go",
            "run",
            "./cmd/configdigest",
        ],
        cwd=repo_root / "oracle",
        env=env,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        check=True,
        timeout=timeout_seconds,
    )


def run_docker_digest_helper(
    repo_root: Path,
    env: dict[str, str],
    timeout_seconds: int,
) -> subprocess.CompletedProcess[str]:
    docker_env_args: list[str] = []
    for key, value in sorted(env.items()):
        if key.startswith("DIGEST_") or re.fullmatch(r"ORACLE\d+_PRIVATE_KEY", key):
            docker_env_args.extend(["-e", f"{key}={value}"])

    container_name = f"tomellini-configdigest-{os.getpid()}-{uuid.uuid4().hex[:8]}"
    command = [
        "docker",
        "run",
        "--rm",
        "--name",
        container_name,
        "-v",
        f"{repo_root / 'oracle'}:/app",
        "-v",
        "tomellini-go-mod-cache:/go/pkg/mod",
        "-v",
        "tomellini-go-build-cache:/root/.cache/go-build",
        "-w",
        "/app",
        *docker_env_args,
        "golang:1.24-alpine",
        "sh",
        "-c",
        "apk add --no-cache git && go run ./cmd/configdigest",
    ]

    try:
        process = subprocess.Popen(
            command,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
        )
        output_parts: list[str] = []
        deadline = time.monotonic() + timeout_seconds

        assert process.stdout is not None
        while process.poll() is None:
            if time.monotonic() > deadline:
                subprocess.run(
                    ["docker", "rm", "-f", container_name],
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                    check=False,
                )
                raise subprocess.TimeoutExpired(
                    command,
                    timeout_seconds,
                    output="".join(output_parts),
                )

            readable, _, _ = select.select([process.stdout], [], [], 0.2)
            if not readable:
                continue

            line = process.stdout.readline()
            if line:
                output_parts.append(line)
                print(line, end="", flush=True)

        remainder = process.stdout.read()
        if remainder:
            output_parts.append(remainder)
            print(remainder, end="", flush=True)

        output = "".join(output_parts)
        if process.returncode != 0:
            raise subprocess.CalledProcessError(
                process.returncode,
                command,
                output=output,
            )

        return subprocess.CompletedProcess(command, process.returncode, stdout=output)
    except subprocess.TimeoutExpired:
        subprocess.run(
            ["docker", "rm", "-f", container_name],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            check=False,
        )
        raise


def run_digest_helper(
    repo_root: Path,
    args: argparse.Namespace,
    env: dict[str, str],
) -> subprocess.CompletedProcess[str]:
    if args.digest_runner in ("auto", "local") and go_version_is_new_enough():
        print("Computing OCR config digest with local Go...", flush=True)
        try:
            return run_local_digest_helper(repo_root, env, args.digest_timeout_seconds)
        except subprocess.CalledProcessError as exc:
            if args.digest_runner == "local":
                raise SystemExit(
                    "failed to compute OCR config digest with local Go:\n"
                    + subprocess_output_text(exc.stdout or exc.output)
                ) from exc
        except subprocess.TimeoutExpired as exc:
            raise SystemExit(
                "timed out computing OCR config digest with local Go:\n"
                + subprocess_output_text(exc.stdout or exc.output)
            ) from exc

    if args.digest_runner == "local":
        raise SystemExit(
            "local Go >= 1.23 is required for --digest-runner local. "
            "Use --digest-runner docker or pass --config-digest explicitly."
        )

    try:
        print("Computing OCR config digest with Docker golang:1.24-alpine...", flush=True)
        return run_docker_digest_helper(repo_root, env, args.digest_timeout_seconds)
    except FileNotFoundError as exc:
        raise SystemExit(
            "docker is required for digest computation in this environment. "
            "Install Docker, use local Go >= 1.23, or pass --config-digest explicitly."
        ) from exc
    except subprocess.CalledProcessError as exc:
        raise SystemExit(
            "failed to compute OCR config digest with Docker:\n"
            + subprocess_output_text(exc.stdout or exc.output)
        ) from exc
    except subprocess.TimeoutExpired as exc:
        raise SystemExit(
            "timed out computing OCR config digest with Docker:\n"
            + subprocess_output_text(exc.stdout or exc.output)
        ) from exc


def main() -> int:
    args = parse_args()
    repo_root = Path(__file__).resolve().parents[1]

    output_path = resolve_repo_path(repo_root, args.out)
    hardhat_config_path = resolve_repo_path(repo_root, args.hardhat_config)
    deploy_script_path = resolve_repo_path(repo_root, args.deploy_script)
    digest_cache_path = resolve_repo_path(repo_root, args.digest_cache)
    env_file_path = resolve_repo_path(repo_root, args.env_file)

    deployer_key = args.deployer_private_key.removeprefix("0x")
    if len(deployer_key) != 64:
        raise SystemExit("--deployer-private-key must be a 32-byte hex key")

    used_keys = {deployer_key.lower()}
    oracle_private_keys = [
        derive_oracle_private_key(str(args.key_seed), idx, used_keys)
        for idx in range(args.num_oracles)
    ]
    oracle_ips = generate_oracle_ips(args.oracle_base_ip, args.num_oracles)
    config_digest = compute_config_digest(
        repo_root,
        args,
        oracle_private_keys,
        digest_cache_path,
    )

    output_path.parent.mkdir(parents=True, exist_ok=True)
    hardhat_config_path.parent.mkdir(parents=True, exist_ok=True)
    deploy_script_path.parent.mkdir(parents=True, exist_ok=True)

    hardhat_config_path.write_text(
        render_hardhat_config(args, deployer_key, oracle_private_keys),
        encoding="utf-8",
    )
    deploy_script_path.write_text(
        render_deploy_script(args, config_digest),
        encoding="utf-8",
    )
    output_path.write_text(
        render_compose(
            args=args,
            repo_root=repo_root,
            output_path=output_path,
            hardhat_config_path=hardhat_config_path,
            deploy_script_path=deploy_script_path,
            env_file_path=env_file_path,
            oracle_private_keys=oracle_private_keys,
            oracle_ips=oracle_ips,
            config_digest=config_digest,
        ),
        encoding="utf-8",
    )

    print(f"Wrote {display_path(repo_root, output_path)}")
    print(f"Wrote {display_path(repo_root, hardhat_config_path)}")
    print(f"Wrote {display_path(repo_root, deploy_script_path)}")
    print(f"Digest cache: {display_path(repo_root, digest_cache_path)}")
    print(f"Computed CONFIG_DIGEST={config_digest}")
    print()
    print("Run:")
    print(
        f"  docker compose -f {display_path(repo_root, output_path)} "
        f"--env-file {display_path(repo_root, env_file_path)} up --build"
    )

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
