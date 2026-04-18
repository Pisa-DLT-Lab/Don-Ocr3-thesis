#!/bin/sh
set -eu

RPC_URL="${CHAIN_RPC:-${CHAIN_RPC_URL:-http://chain:8545}}"
AGGREGATOR="${AGGREGATOR_ADDRESS:-}"
MAX_WAIT_SECONDS="${WAIT_FOR_DEPLOY_TIMEOUT_SECONDS:-1800}"

echo " [WAIT] Waiting for Hardhat RPC at ${RPC_URL}..."

start_time="$(date +%s)"
while true; do
    if curl -sf \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
        "$RPC_URL" >/dev/null; then
        break
    fi

    now="$(date +%s)"
    if [ $((now - start_time)) -ge "$MAX_WAIT_SECONDS" ]; then
        echo " [WAIT] Timed out waiting for Hardhat RPC" >&2
        exit 1
    fi
    sleep 1
done

if [ -z "$AGGREGATOR" ]; then
    echo " [WAIT] AGGREGATOR_ADDRESS is not set" >&2
    exit 1
fi

echo " [WAIT] Waiting for deployment transactions to reach Aggregator block..."
while true; do
	block_response="$(curl -sf \
		-H "Content-Type: application/json" \
		-d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
		"$RPC_URL" || true)"

	if printf '%s' "$block_response" | grep -Eq '"result":"0x([4-9a-fA-F]|[1-9a-fA-F][0-9a-fA-F]+)"'; then
		break
	fi

	now="$(date +%s)"
	if [ $((now - start_time)) -ge "$MAX_WAIT_SECONDS" ]; then
		echo " [WAIT] Timed out waiting for deployment blocks; last response: ${block_response}" >&2
		exit 1
	fi
	sleep 1
done

echo " [WAIT] Waiting for Aggregator queue()/verifier() to be readable..."
while true; do
	queue_response="$(curl -sf \
		-H "Content-Type: application/json" \
		-d "{\"jsonrpc\":\"2.0\",\"method\":\"eth_call\",\"params\":[{\"to\":\"${AGGREGATOR}\",\"data\":\"0xe10d29ee\"},\"latest\"],\"id\":1}" \
		"$RPC_URL" || true)"
	verifier_response="$(curl -sf \
		-H "Content-Type: application/json" \
		-d "{\"jsonrpc\":\"2.0\",\"method\":\"eth_call\",\"params\":[{\"to\":\"${AGGREGATOR}\",\"data\":\"0x2b7ac3f3\"},\"latest\"],\"id\":1}" \
		"$RPC_URL" || true)"

	if printf '%s' "$queue_response" | grep -Eq '"result":"0x0{24}[0-9a-fA-F]{40}"' &&
		printf '%s' "$verifier_response" | grep -Eq '"result":"0x0{24}[0-9a-fA-F]{40}"'; then
		echo " [WAIT] Aggregator child contract getters are readable."
		break
	fi

	now="$(date +%s)"
	if [ $((now - start_time)) -ge "$MAX_WAIT_SECONDS" ]; then
		echo " [WAIT] Timed out waiting for Aggregator queue()/verifier(); last queue response: ${queue_response}; last verifier response: ${verifier_response}" >&2
		exit 1
	fi
	sleep 1
done

echo " [START] Deployment is ready. Start Oracle Node..."

exec "$@"
