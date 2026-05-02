# A decentralized framework for automated and trustworthy royalty computation for AI model training data contributors

This repository contains the implementation of a framework for computing and distributing royalties to the original owners of the data used to train Large Language Models (LLMs). 

The system leverages the Ethereum blockchain and is built on top of a Decentralized Oracle Network (DON) using Chainlink's Off-Chain Reporting (OCR) protocol.

## Contributors

This codebase was developed by the following contributors:

- **Federico Tomellini**, University of Pisa, Italy
- **Calogero Turco**, University of Pisa, Italy
- **Matteo Loporchio**, University of Pisa, Italy

## Project structure

The project is organized into the following main directories:

* **chain**: Contains the Solidity smart contracts, Hardhat configuration, and deployment scripts.
* **model**: Contains the Python code for the AI model and the corresponding attribution method.
* **oracle**: Contains the off-chain oracle node backend written in Go, including the custom OCR3 (Off-Chain Reporting v3) plugin and listener.
* **scripts**: Automation helpers for generating parametrized Docker Compose stacks and starting experiments.

## Tech stack

The project utilizes a variety of technologies across different components:

* **Blockchain/Smart Contracts:** Solidity, Hardhat, Ethers.js
* **Oracle Infrastructure:** Chainlink DON, Go (Golang), Docker
* **Decentralized Storage:** IPFS (kubo)
* **AI Model & Attribution:** Python, PyTorch, dattri

## Installation and setup

To run this project locally, ensure you have the following installed on your machine:

* **Git**
* **Node.js** (v24.13.0 or higher)
* **Go** (v1.23.4 linux/amd64 or higher)
* **Docker** and **Docker Compose** (v29.2.0 and Docker Compose version v5.0.2)
* **Python** (for the AI backend service)

Follow these steps to set up the project locally:

1. **Clone the repository**
```bash
    git clone https://github.com/FedeTome0/Don-Ocr3-thesis.git
    cd Don-Ocr3-thesis
```
2. **Install smart contract dependencies (Hardhat)**
```bash
    cd chain
    npm install
    cd ..
```
3. **Install Python dependencies for the AI backend**
```bash
    cd model
    python3 -m venv .venv
    source .venv/bin/activate
    pip3 install dattri torch numpy transformers datasets tiktoken wandb tqdm Flask
```

## How to run

To test the full system, follow this chronological sequence.

<!--
> **Note - Generated Files:** The automation writes `docker-compose.generated.yml`, `generated/hardhat.config.generated.js`, `generated/deployContracts.generated.js`, and `generated/config-digests.json`. These are derived artifacts for a specific experiment configuration.

> **Note - Digest Cache:** The first run for a new configuration computes the OCR config digest. The result is saved in `generated/config-digests.json`, so rerunning the same parameters reuses the cached digest.

> **Note - Byzantine Testing:** Malicious oracle behavior can be injected directly at generation time. By default, both `--malicious-alter-count` and `--malicious-timeout-count` are `0`, so all generated nodes are honest unless you opt in.

-->

### Step 1: Start the AI module

Before starting the oracle containers, ensure the Python AI module (i.e., the generation and attribution service) is running and accessible at `host.docker.internal:9090`.

To start the service:

```bash
cd model
source .venv/bin/activate
python3 model_server_service.py
```

By default, `model_server_service.py` listens on `0.0.0.0:9090`, which is the address expected by the oracle containers through `host.docker.internal:9090`.

### Step 2: Boot the core infrastructure

Thanks to the Docker orchestration, booting the entire ecosystem is highly automated. 

In particular, the **generated stack script** creates a parametrized compose file for a selected number of oracles and network seed.

The generated setup will:

- Start the Hardhat local node.
- Wait for it to be ready.
- Fund the seed-derived oracle accounts plus one extra customer account in Hardhat.
- Deploy the smart contracts with the matching OCR config digest.
- Boot the off-chain oracle nodes.
- Apply deterministic simulated latency among oracle containers by default.

To run the script, open a new terminal in the root directory of the project. Note that, if no `.env` file exists, the runner creates it from `.env.example`. Then execute:

```bash
scripts/run_generated_stack_toxiproxy.sh <NUMBER_OF_ORACLES> <NETWORK_SEED> [--malicious-alter-count N] [--malicious-timeout-count M] [--disable-latency]
```

The script requires the following mandatory parameters:

- `NUMBER_OF_ORACLES`: The number of oracle nodes to generate in the stack (e.g., `7`).
- `NETWORK_SEED`: A seed for deterministic generation of oracle keys, latency, and malicious node selection (e.g., `123`).

The script also accepts optional flags for malicious node configuration and latency control:

- `--malicious-alter-count N`: Number of oracles to assign the "alter" malicious behavior (default: `0`). These nodes will alter their response to the Aggregator, simulating data tampering or misreporting.

- `--malicious-timeout-count M`: Number of oracles to assign the "timeout" malicious behavior (default: `0`). These nodes will exhibit timeout behavior, simulating unresponsiveness or denial of service.

You can combine both flags to create mixed malicious node configurations. For example, `--malicious-alter-count 2 --malicious-timeout-count 1` will assign 2 oracles to alter and 1 oracle to timeout behavior. 

Finally, simulated network latency is enabled by default. The optional `--disable-latency` flag allows you to disable the latency among oracle nodes while keeping the generated configuration and Compose stack intact. This can be useful for testing scenarios where latency is not a factor or when you want to isolate other variables in the system.

This script uses the Toxiproxy framework (https://github.com/shopify/toxiproxy) to simulate latency. Note that Toxiproxy-based latency is used by default, which is compatible with all Docker hosts. If you are running on a Linux host with `tc netem` support, you can use the `scripts/run_generated_stack.sh` script instead to apply latency directly at the kernel level (do note that this tool supports network configuration with at most 15 oracles).


<!--
#### Option A: Toxiproxy latency

Use this option on WSL or any Docker host where Linux `tc netem` qdisc support is unavailable.

Example with 7 oracles and seed `123`:

```bash
scripts/run_generated_stack_toxiproxy.sh 7 123
```

Example with 2 malicious `alter` nodes and 1 malicious `timeout` node:

```bash
scripts/run_generated_stack_toxiproxy.sh 7 123 --malicious-alter-count 2 --malicious-timeout-count 1
```

For 5 oracles:

```bash
scripts/run_generated_stack_toxiproxy.sh 5 42
```

#### Option B: Kernel `tc netem` latency

Use this option on Linux hosts with `sch_prio`, `sch_netem`, and `cls_u32` support.

Example with 7 oracles and seed `123`:

```bash
scripts/run_generated_stack.sh 7 123
```

Example with 2 malicious `alter` nodes and 1 malicious `timeout` node:

```bash
scripts/run_generated_stack.sh 7 123 --malicious-alter-count 2 --malicious-timeout-count 1
```

For 5 oracles:

```bash
scripts/run_generated_stack.sh 5 42
```

Simulated network latency is enabled by default. To disable it while keeping generated keys and Compose:

```bash
scripts/run_generated_stack.sh 7 123 --disable-latency
```

#### Note on OCR config digest computation

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

#### What happens under the hood

- `scripts/generate_compose.py` generates a Compose stack for the requested number of oracles
- Oracle private keys are deterministically derived from the seed
- Malicious oracle roles are also deterministically selected from `NETWORK_SEED`; by default `--malicious-alter-count=0` and `--malicious-timeout-count=0`
- A generated Hardhat config funds those oracle accounts
- The OCR config digest is computed and cached
- `generated/deployContracts.generated.js` deploys contracts using the selected digest from `CONFIG_DIGEST`
- `generated/deployContracts.generated.js` sets the Aggregator filter policy from `.env`
- The oracle containers read `AGGREGATOR_ADDRESS` from `.env`; Queue, Verifier, and filter policy are discovered from Aggregator on-chain
- `scripts/generate_compose.py` assigns deterministic WIPO-weighted Azure regions from `NETWORK_SEED` and `ORACLE_ID`
- `oracle/setup_network_parametric.sh` reads those generated Azure regions and applies `tc netem` latency by default
- Docker Compose builds and starts the chain, IPFS, bootstrap, and oracle containers

You can generate malicious oracle mixes with:

```bash
--malicious-alter-count N
--malicious-timeout-count M
```

Selected nodes receive `MALICIOUS_MODE=alter` or `MALICIOUS_MODE=timeout` in the generated Compose file. The same `NETWORK_SEED` always produces the same malicious-node assignment for a given `(NUM_ORACLES, N, M)` configuration.

#### Latency data sources

In the absence of precise data about where oracle services are geographically deployed, we use regional patent application counts as a proxy for regional technological activity and derive placement probabilities from the latest available WIPO data.

- `oracle/latency/wipo_patent_region_probabilities.csv` stores the WIPO-derived region placement probabilities. Source: WIPO IP Statistics Data Center: https://www3.wipo.int/ipstats/key-search/search-result?type=KEY&key=203
- `oracle/latency/azure_region_latencies.csv` stores the Azure inter-region latency matrix used for toxiproxi and `tc netem` latency assignment. Source: https://learn.microsoft.com/en-us/azure/networking/azure-network-latency?tabs=Americas%2CWestUS
- Generated oracle locations are deterministic from `NETWORK_SEED` and `ORACLE_ID`.

-->

**Important**: Once you have launched the script, wait until you see the message:

```
--- CHAIN READY ---
```

in the Docker logs before proceeding.


### Step 3: Simulate the Workflow

Once the Docker infrastructure is up and running, you can simulate the main workflow of the system, which involves interactions between the **model creator** and the **customer** through the deployed smart contracts and oracle network.

#### 1. Start the Model Creator Listener

Open a new terminal (1):

```bash
cd chain
npx hardhat run scripts/modelCreatorApprove.js --network localhost
```

This script simulates the model creator listening for ```Aggregator``` request events. Wait for it to be deployed and then start listening to the events.

#### 2. Trigger the Customer Request

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


## How to stop the environment

To stop the execution of containers, press `CTRL+C` in the Docker terminal.

<!--

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

-->
