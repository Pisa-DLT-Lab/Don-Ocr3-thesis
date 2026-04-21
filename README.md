# Decentralized Oracle Network (DON) for LLM Royalties

This repository contains my Bachelor's thesis project, focused on the development of a Decentralized Oracle Network (DON) utilizing Chainlink's architecture and libraries.

The project proposes a novel decentralized application designed to address fair compensation in the AI industry. Specifically, it attributes and distributes royalties to the original owners of the data used to train Large Language Models (LLMs). By leveraging the off-chain computation capabilities of the DON, custom oracles securely execute an attribution algorithm to calculate the correct royalty distribution, ensuring a transparent, trustless, and decentralized reward system.

## Project Structure

* **chain**: Contains the Solidity smart contracts (`Aggregator.sol`, `OracleQueue.sol`, `OracleVerifier.sol`), Hardhat configuration, and deployment scripts.
* **oracle**: Contains the off-chain Oracle node backend written in Go, including the custom OCR3 (Off-Chain Reporting) plugin and listener.
* **IpfsAgent**: Scripts and configurations for handling data storage and retrieval via IPFS.
* **docker-compose.yml**: Static Docker Compose setup for the oracle network and testing environment.
* **scripts**: Automation helpers for generating parametrized Docker Compose stacks and starting experiments.

## Tech Stack
* **Blockchain/Smart Contracts:** Solidity, Hardhat, Ethers.js
* **Oracle Infrastructure:** Chainlink DON, Go (Golang), Docker
* **Decentralized Storage:** IPFS (kubo)

---

## Prerequisites

To run this project locally, ensure you have the following installed on your machine:
* **Node.js** (v24.13.0 or higher)
* **Go** (v1.23.4 linux/amd64 or higher)
* **Docker** and **Docker Compose** (v29.2.0 and Docker Compose version v5.0.2)
* **Git**

---

## Installation & Setup

1. **Clone the repository**
```bash
    git clone https://github.com/FedeTome0/Don-Ocr3-thesis.git
    cd Don-Ocr3-thesis
```
2. **Install Smart Contract dependencies (Hardhat)**
```bash
    cd chain
    npm install
    cd ..
```
---

## How to Run the Project locally 
Thanks to the Docker orchestration, booting the entire ecosystem is highly automated.
The recommended path is the generated stack script, which creates a parametrized Compose file for a selected number of oracles and network seed.

The generated setup will:

- Start the Hardhat local node
- Wait for it to be ready
- Fund the seed-derived oracle accounts plus one extra customer account in Hardhat
- Deploy the smart contracts with the matching OCR config digest
- Boot the off-chain oracle nodes
- Apply deterministic simulated latency among oracle containers by default

To test the full system, follow this chronological sequence.

> **Note - Generated Files:** The automation writes `docker-compose.generated.yml`, `generated/hardhat.config.generated.js`, `generated/deployContracts.generated.js`, and `generated/config-digests.json`. These are derived artifacts for a specific experiment configuration.

> **Note - Digest Cache:** The first run for a new configuration computes the OCR config digest. The result is saved in `generated/config-digests.json`, so rerunning the same parameters reuses the cached digest.

> **Note - Byzantine Testing:** You can test the BFT consensus by simulating malicious node behavior. In the generated Compose file, change a node's `MALICIOUS_MODE` environment variable to `"transmit_fail"`, `"timeout"`, or `"alter"`.

---
## Step 1: Start the Local AI Backend

The oracle containers expect the attribution service to be reachable at `host.docker.internal:9090`. From the repository root:

```bash
cd model
python3 -m venv .venv
source .venv/bin/activate
pip3 install dattri torch numpy transformers datasets tiktoken wandb tqdm web3 Flask
```

The local service expects the Shakespeare checkpoint at:

```bash
model/nanoGPT/out-shakespeare-char/ckpt.pt
```

If the checkpoint is stored elsewhere, create the expected directory and link it:

```bash
mkdir -p nanoGPT/out-shakespeare-char
ln -s /absolute/path/to/ckpt.pt nanoGPT/out-shakespeare-char/ckpt.pt
```

Then start the service:

```bash
python3 model_server_service.py
```

By default, `model_server_service.py` listens on `0.0.0.0:9090`, which is the address expected by the oracle containers through `host.docker.internal:9090`.

You can override the host or port if needed:

```bash
MODEL_SERVER_HOST=0.0.0.0 MODEL_SERVER_PORT=9090 python3 model_server_service.py
```


---

## Step 2: Boot the Core Infrastructure

Open a new terminal in the root directory of the project. If `.env` does not exist, the runner creates it from `.env.example`.

Run a generated experiment by passing:

- `NUM_ORACLES`
- `NETWORK_SEED`

The generated oracle placement first selects one macro region with these probabilities:

| Macro region | Patent applications | Placement probability |
| --- | ---: | ---: |
| Africa | 19,100 | 0.5127516779% |
| Asia | 2,612,500 | 70.1342281879% |
| Europe | 362,700 | 9.7369127517% |
| LAC | 55,600 | 1.4926174497% |
| Northern America | 638,600 | 17.1436241611% |
| Oceania | 36,500 | 0.9798657718% |

Within the selected macro region, the generator distributes probability evenly across eligible Azure subgroups, then evenly across eligible Azure regions in each subgroup.

### Option A: Toxiproxy latency

Use this option on WSL or any Docker host where Linux `tc netem` qdisc support is unavailable.

Example with 7 oracles and seed `123`:

```bash
scripts/run_generated_stack_toxiproxy.sh 7 123
```

For 5 oracles:

```bash
scripts/run_generated_stack_toxiproxy.sh 5 42
```

### Option B: Kernel `tc netem` latency

Use this option on Linux hosts with `sch_prio`, `sch_netem`, and `cls_u32` support.

Example with 7 oracles and seed `123`:

```bash
scripts/run_generated_stack.sh 7 123
```

For 5 oracles:

```bash
scripts/run_generated_stack.sh 5 42
```

Simulated network latency is enabled by default. To disable it while keeping generated keys and Compose:

```bash
scripts/run_generated_stack.sh 7 123 --disable-latency
```

If the first OCR digest computation is slow, pre-pull the main images:

```bash
docker pull golang:1.24-alpine
docker pull ghcr.io/shopify/toxiproxy:2.12.0
docker pull curlimages/curl:8.10.1
docker pull node:18-alpine
docker pull alpine:latest
docker pull ipfs/kubo:latest
```

If digest computation still needs more time:

```bash
scripts/run_generated_stack.sh 7 123 --digest-timeout-seconds 3600
```

The on-chain Aggregator filter policy is configured from `.env` during deployment:

```bash
FILTER_POLICY=TOP_HOLDERS
FILTER_THRESHOLD=100
```

`FILTER_POLICY` accepts `TOP_HOLDERS` or `TOP_VALUES`. If it is not set, deployment defaults to `TOP_HOLDERS`.

### What happens under the hood

- `scripts/generate_compose.py` generates a Compose stack for the requested number of oracles
- Oracle private keys are deterministically derived from the seed
- A generated Hardhat config funds those oracle accounts
- The OCR config digest is computed and cached
- `generated/deployContracts.generated.js` deploys contracts using the selected digest from `CONFIG_DIGEST`
- `generated/deployContracts.generated.js` sets the Aggregator filter policy from `.env`
- The oracle containers read `AGGREGATOR_ADDRESS` from `.env`; Queue, Verifier, and filter policy are discovered from Aggregator on-chain
- `scripts/generate_compose.py` assigns deterministic WIPO-weighted Azure regions from `NETWORK_SEED` and `ORACLE_ID`
- `oracle/setup_network_parametric.sh` reads those generated Azure regions and applies `tc netem` latency by default
- Docker Compose builds and starts the chain, IPFS, bootstrap, and oracle containers

### Latency data sources

In the absence of precise data about where oracle services are geographically deployed, we use regional patent application counts as a proxy for regional technological activity and derive placement probabilities from the latest available WIPO data.

- `oracle/latency/wipo_patent_region_probabilities.csv` stores the WIPO-derived region placement probabilities. Source: WIPO IP Statistics Data Center: https://www3.wipo.int/ipstats/key-search/search-result?type=KEY&key=203
- `oracle/latency/azure_region_latencies.csv` stores the Azure inter-region latency matrix used for toxiproxi and `tc netem` latency assignment. Source: https://learn.microsoft.com/en-us/azure/networking/azure-network-latency?tabs=Americas%2CWestUS
- Generated oracle locations are deterministic from `NETWORK_SEED` and `ORACLE_ID`.

Wait until you see the message:

```
--- CHAIN READY ---
```

in the Docker logs before proceeding.

## Step 3: Simulate the Workflow

Now you simulate the two actors of the system:

- Model Creator
- Customer

---

### 1. Start the Model Creator Listener

Open a new terminal (1):

```bash
cd chain
npx hardhat run scripts/modelCreatorApprove.js --network localhost
```

This script simulates the model creator listening for Aggregator request events. Wait for it to be deployed and then start listening to the events.

---

### 2. Trigger the Customer Request

Open another terminal (2):

```bash
cd chain
npx hardhat run scripts/customerRequest.js --network localhost
```
Check for latest result in terminal (2):

```bash
cd chain
npx hardhat run scripts/verify.js --network localhost
```

Instead, for testing:
```bash
cd chain
npx hardhat run scripts/benchmark.js --network localhost
```

At this point:

- The computation starts
- The oracle consensus begins
- Royalties are calculated

The customer scripts use signer `NUM_ORACLES + 1`, after account `0` for the model creator and accounts `1..NUM_ORACLES` for oracle nodes.

You can monitor everything in the main Docker terminal.
To check the current result through the Aggregator facade, use `verify.js` to inspect the first ten numbers of the vector.

---

## How to Stop the Environment

To stop execution:

Press `CTRL+C` in the Docker terminal.

To rebuild the containers:

```bash
docker compose -f docker-compose.generated.yml --env-file .env down -v
```

For the static Compose mode:

```bash
docker compose down -v
```

To reset everything (deleting docker's data)
```bash
docker system prune -a -f
```
