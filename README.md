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
- Fund the seed-derived oracle accounts in Hardhat
- Deploy the smart contracts with the matching OCR config digest
- Boot the off-chain oracle nodes
- Apply deterministic simulated latency among oracle containers by default

To test the full system, follow this chronological sequence.

> **Note - Generated Files:** The automation writes `docker-compose.generated.yml`, `generated/hardhat.config.generated.js`, `generated/deployContracts.generated.js`, and `generated/config-digests.json`. These are derived artifacts for a specific experiment configuration.

> **Note - Digest Cache:** The first run for a new configuration computes the OCR config digest. The result is saved in `generated/config-digests.json`, so rerunning the same parameters reuses the cached digest.

> **Note - Byzantine Testing:** You can test the BFT consensus by simulating malicious node behavior. In the generated Compose file, change a node's `MALICIOUS_MODE` environment variable to `"transmit_fail"`, `"timeout"`, or `"alter"`.

---
## Step 1: Start the AI Backend and SSH Tunnel

The oracles communicate with the Python AI server hosted on the remote machine (`satoshi`).

### Option A: Use the remote server

### 1. Access the remote server via SSH tunnel

```bash
ssh -L 9090:127.0.0.1:50100 tomelliniT@131.114.50.205
```
### 2. Follow this path
 Go in `data/tomelliniT/oc3-thesis/alps-ai-master/model`. Run `source env/bin/activate` to activate the python environment and then you can proceed by starting the server.

### 3. Start the Python model service

```bash
python model_service_v2.py
```

### Option B: Run the local model service

You can also run the attribution service locally. From the repository root:

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
- `chain/scripts/deployContracts.js` deploys contracts using the selected digest from `CONFIG_DIGEST`
- `chain/scripts/deployContracts.js` sets the Aggregator filter policy from `.env`
- The oracle containers read `AGGREGATOR_ADDRESS` from `.env`; Queue, Verifier, and filter policy are discovered from Aggregator on-chain
- `oracle/setup_network_parametric.sh` assigns deterministic pseudo-random locations and applies `tc netem` latency by default
- Docker Compose builds and starts the chain, IPFS, bootstrap, and oracle containers

Wait until you see the message:

```
--- CHAIN READY ---
```

in the Docker logs before proceeding.

### Manual Static Compose Mode

The original static Compose file is still available:

```bash
docker compose up --build
```

This mode uses the fixed services declared in `docker-compose.yml`. To change the network size manually, edit `docker-compose.yml` and `.env` consistently.

---

## Step 3: Simulate the Workflow

Now you simulate the two actors of the system:

- Model Creator
- Customer

---

### 1. Start the Model Creator Listener

Open a new terminal:

```bash
cd chain
npx hardhat run scripts/modelCreatorApprove.js --network localhost
```

This script simulates the model creator listening for Aggregator request events. Wait for it to be deployed and then start listening to the events.

---

### 2. Trigger the Customer Request

Open another terminal:

```bash
cd chain
npx hardhat run scripts/customerRequest.js --network localhost
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
