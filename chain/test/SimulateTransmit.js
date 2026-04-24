// This test measures the gas used for calling the functions of the Aggregator contract.

const { ethers } = require("hardhat");
const yargs = require("yargs/yargs");
const { hideBin } = require("yargs/helpers");

const args = yargs(hideBin(process.argv))
    .option("numOracles", {
        alias: "n",
        description: "Number of oracles",
        type: "number",
        default: 31
    })
    .option("numFaulty", {
        alias: "f",
        description: "Number of faulty oracles",
        type: "number",
        default: 10
    })
    .option("numHolders", {
        alias: "h",
        description: "Number of holders",
        type: "number",
        default: 1000
    })
    .option("reportSize", {
        alias: "r",
        description: "Size of each report",
        type: "number",
        default: 1000
    })
    .argv;

const NUM_ORACLES = args.numOracles;
const NUM_FAULTY = args.numFaulty;
const NUM_HOLDERS = args.numHolders;
const REPORT_SIZE = args.reportSize;
const REQUEST_FEE = ethers.parseEther("0.01"); // Fee paid by the customer
const ORACLE_REWARD = ethers.parseEther("0.003"); // Reward to refund the oracle
const MODEL_CREATOR_REWARD = ethers.parseEther("0.002"); // Reward for the model creator
const CONFIG_DIGEST = "0x0001eb5bd089a1e47f2e7bf9d2332c5b1d9717a4268f5441b3a61d2db8c05475";
const IPFS_EXAMPLE_CID = "bafybeigrf2dwtpjkiovnigysyto3d55opf6qkdikx6d65onrqnfzwgdkfa"

// Generates an array of random Ethereum addresses of the specified size.
function generateAddressArray(size) {
    const array = [];
    for (let i = 0; i < size; i++) {
        array.push(ethers.Wallet.createRandom().address);
    }
    return array;
}

// Packs an array of byte values into a Solidity bytes32 value.
function packBytesToBytes32(values) {
    if (values.length > 32) {
        throw new Error("Too many values (max 32)");
    }
    let packed = 0n;
    for (let i = 0; i < values.length; i++) {
        if (values[i] < 0 || values[i] > 255) {
            throw new Error("Each value must be a byte (0-255)");
        }
        const shift = 8n * BigInt(31 - i);
        packed |= BigInt(values[i]) << shift;
    }
    return ethers.toBeHex(packed, 32); // returns bytes32
}

// Derive a full Wallet (with private key access) from a Hardhat signer index.
function getDerivedWallet(index, mnemonic) {
    const phrase = mnemonic || "test test test test test test test test test test test junk";
    return ethers.HDNodeWallet.fromPhrase(
        phrase,
        undefined,
        `m/44'/60'/0'/0/${index}`
    );
}

async function main() {
    console.log(`Testing with ${NUM_ORACLES} oracles, ${NUM_FAULTY} faulty nodes, ${NUM_HOLDERS} holders, report size ${REPORT_SIZE}`);

    const signers = await ethers.getSigners();
    const modelCreatorAddress = signers[0].address;
    const holderAddresses = generateAddressArray(NUM_HOLDERS);
    const oracleAddresses = [];
    for (let i = 1; i <= NUM_ORACLES; i++) {
        oracleAddresses.push(signers[i].address);
    }

    // Deploy the OracleQueue contract.
    const OracleQueue = await ethers.getContractFactory("OracleQueue");
    const queueContract = await OracleQueue.deploy();
    const queueReceipt = await queueContract.deploymentTransaction().wait();
    const queueAddress = await queueContract.getAddress();
    console.log(`OracleQueue deployed at ${queueAddress} with gas used: ${queueReceipt.gasUsed.toString()}`);
    
    // Deploy the OracleVerifier contract.
    const OracleVerifier = await ethers.getContractFactory("OracleVerifier");
    const verifierContract = await OracleVerifier.deploy(oracleAddresses, NUM_FAULTY, CONFIG_DIGEST);
    const verifierReceipt = await verifierContract.deploymentTransaction().wait();
    const verifierAddress = await verifierContract.getAddress();
    console.log(`OracleVerifier deployed at ${verifierAddress} with gas used: ${verifierReceipt.gasUsed.toString()}`);

    // Deploy the RoyaltyManager contract.
    const RoyaltyManager = await ethers.getContractFactory("RoyaltyManager");
    const royaltyManagerContract = await RoyaltyManager.deploy(holderAddresses, verifierAddress);
    const royaltyManagerReceipt = await royaltyManagerContract.deploymentTransaction().wait();
    const royaltyManagerAddress = await royaltyManagerContract.getAddress();
    console.log(`RoyaltyManager deployed at ${royaltyManagerAddress} with gas used: ${royaltyManagerReceipt.gasUsed.toString()}`);

    // Deploy the Aggregator contract.
    const Aggregator = await ethers.getContractFactory("Aggregator");
    const aggregatorContract = await Aggregator.deploy(
        REQUEST_FEE, 
        ORACLE_REWARD, 
        MODEL_CREATOR_REWARD,
        verifierAddress,
        queueAddress,
        royaltyManagerAddress
    );
    const aggregatorReceipt = await aggregatorContract.deploymentTransaction().wait();
    const aggregatorAddress = await aggregatorContract.getAddress();
    console.log(`Aggregator deployed at ${aggregatorAddress} with gas used: ${aggregatorReceipt.gasUsed.toString()}`);

    // Set the Aggregator address in the OracleQueue contract
    const setAggregatorTx = await queueContract.setAggregator(aggregatorAddress);
    const setAggregatorReceipt = await setAggregatorTx.wait();
    console.log(`Set Aggregator in OracleQueue with gas used: ${setAggregatorReceipt.gasUsed.toString()}`);

    // Set the Aggregator address in the OracleVerifier contract
    const setVerifierTx = await verifierContract.setAggregator(aggregatorAddress);
    const setVerifierReceipt = await setVerifierTx.wait();
    console.log(`Set Aggregator in OracleVerifier with gas used: ${setVerifierReceipt.gasUsed.toString()}`);

    // Set the Aggregator address in the RoyaltyManager contract
    const setRoyaltyManagerTx = await royaltyManagerContract.setAggregator(aggregatorAddress);
    const setRoyaltyManagerReceipt = await setRoyaltyManagerTx.wait();
    console.log(`Set Aggregator in RoyaltyManager with gas used: ${setRoyaltyManagerReceipt.gasUsed.toString()}`);

    // Call requestAttribution.
    const requestAttributionTx = await aggregatorContract.requestAttribution(
        IPFS_EXAMPLE_CID, 
        {value: REQUEST_FEE}
    );
    const requestAttributionReceipt = await requestAttributionTx.wait();
    const event = requestAttributionReceipt.logs.map(log => {
        try {
            return queueContract.interface.parseLog(log);
        } catch {
            return null;
        }
    }).find(e => e && e.name === "LogNewCustomerRequest");
    const requestId = event.args.requestId;
    console.log(`Request with ID: ${requestId.toString()} created with gas used: ${requestAttributionReceipt.gasUsed.toString()}`);

    // Call approveJob.
    const approveJobTx = await aggregatorContract.approveJob(requestId);
    const approveJobReceipt = await approveJobTx.wait();
    console.log(`Approved job with ID: ${requestId.toString()} with gas used: ${approveJobReceipt.gasUsed.toString()}`);

    // Call transmit.
    const signer = signers[1]; 
    const contractAsSigner = aggregatorContract.connect(signer);
    const seqNr = 1; // example sequence number
    const jobId = 0; // example job ID
    const vector = new Array(REPORT_SIZE).fill(0); vector[0] = 1;// example result vector
    const reportData = ethers.AbiCoder.defaultAbiCoder().encode(["uint256", "int128[]"], [jobId, vector]);
    const hash = ethers.solidityPackedKeccak256(["bytes32", "uint64", "bytes"], [CONFIG_DIGEST, seqNr, reportData]);
    let rs = [];
    let ss = [];
    let vs = [];
    for (let i = 1; i <= NUM_ORACLES; i++) {
        const wallet = getDerivedWallet(i);
        const signature = wallet.signingKey.sign(ethers.getBytes(hash));
        const sig = ethers.Signature.from(signature);
        rs.push(ethers.zeroPadValue(sig.r, 32));
        ss.push(ethers.zeroPadValue(sig.s, 32));
        vs.push(sig.v);
        //const recovered = ethers.recoverAddress(hash, {r: sig.r, s: sig.s, v: sig.v});
        //console.log(sig.v, signers[i].address, recovered);
    }
    const rawVs = packBytesToBytes32(vs);
    const transmitTx = await contractAsSigner.transmit(CONFIG_DIGEST, seqNr, reportData, rs, ss, rawVs);
    const transmitReceipt = await transmitTx.wait();
    console.log(`Transmitted report for job ID: ${jobId} with gas used: ${transmitReceipt.gasUsed.toString()}`);
}

main().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});