#!/bin/sh
set -eu

usage() {
    cat <<'EOF'
Usage:
  scripts/run_generated_stack.sh NUM_ORACLES NETWORK_SEED [generator args...]

Examples:
  scripts/run_generated_stack.sh 7 123
  scripts/run_generated_stack.sh 10 999 --disable-latency

Environment overrides:
  ENV_FILE=.env
  COMPOSE_FILE=docker-compose.generated.yml

The script:
  1. creates .env from .env.example if .env is missing
  2. runs scripts/generate_compose.py
  3. runs docker compose up --build with the generated Compose file
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
fi

if [ "$#" -lt 2 ]; then
    usage >&2
    exit 2
fi

NUM_ORACLES="$1"
NETWORK_SEED="$2"
shift 2

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "${SCRIPT_DIR}/.." && pwd)"

ENV_FILE="${ENV_FILE:-.env}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.generated.yml}"

cd "$REPO_ROOT"

cat <<'EOF'
[RUN] First-time runs can be slow while Docker pulls images and Go modules.
[RUN] To warm the main images beforehand, run:
[RUN]   docker pull golang:1.24-alpine
[RUN]   docker pull node:18-alpine
[RUN]   docker pull alpine:latest
[RUN]   docker pull ipfs/kubo:latest
[RUN] If digest computation times out, rerun with: --digest-timeout-seconds 3600
EOF

if [ ! -f "$ENV_FILE" ]; then
    if [ "$ENV_FILE" = ".env" ] && [ -f ".env.example" ]; then
        echo "[RUN] .env missing; creating it from .env.example"
        cp .env.example .env
    else
        echo "[RUN] Env file not found: $ENV_FILE" >&2
        exit 1
    fi
fi

echo "[RUN] Generating Compose stack for NUM_ORACLES=${NUM_ORACLES}, NETWORK_SEED=${NETWORK_SEED}"
if ! python3 scripts/generate_compose.py \
    --num-oracles "$NUM_ORACLES" \
    --network-seed "$NETWORK_SEED" \
    --out "$COMPOSE_FILE" \
    --env-file "$ENV_FILE" \
    "$@"
then
    cat >&2 <<'EOF'
[RUN] Generation failed, so Docker Compose was not started.
[RUN] The config digest cache is only updated after a digest is computed successfully.
[RUN] For a slow first Docker-based digest computation, try:
[RUN]   scripts/run_generated_stack.sh NUM_ORACLES NETWORK_SEED --digest-timeout-seconds 3600
[RUN] Or pre-warm the helper image:
[RUN]   docker pull golang:1.24-alpine
EOF
    exit 1
fi

echo "[RUN] Starting generated Docker Compose stack"
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up --build
