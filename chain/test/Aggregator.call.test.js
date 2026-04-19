// This test measures the gas used for calling the functions of the Aggregator contract.

const { ethers } = require("hardhat");

const NUM_ORACLES = 31;
const NUM_FAULTY = 10;
const NUM_HOLDERS = 1000;
const REPORT_SIZE = 1000;
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

describe("Aggregator Test", function () {
    let queueContract, queueAddress;
    let verifierContract, verifierAddress;
    let royaltyManagerContract, royaltyManagerAddress;
    let aggregatorContract, aggregatorAddress;
    let signers;
    let modelCreatorAddress;
    let oracleAddresses = [];
    const holderAddresses = generateAddressArray(NUM_HOLDERS);
    let requestId;
    let wallets;
    
    before(async () => {
        signers = await ethers.getSigners();
        modelCreatorAddress = signers[0].address;
        for (let i = 1; i <= NUM_ORACLES; i++) {
            //console.log(signers[i]);
            oracleAddresses.push(signers[i].address);
        }
    });

    // Deploy the OracleQueue contract.
    it("Deploy OracleQueue", async function () {
        const OracleQueue = await ethers.getContractFactory("OracleQueue");
        queueContract = await OracleQueue.deploy();
        const queueReceipt = await queueContract.deploymentTransaction().wait();
        queueAddress = await queueContract.getAddress();
        console.log(`OracleQueue deployed at ${queueAddress} with gas used: ${queueReceipt.gasUsed.toString()}`);
    });

    // Deploy the OracleVerifier contract.
    it("Deploy OracleVerifier", async function () {
        const OracleVerifier = await ethers.getContractFactory("OracleVerifier");
        verifierContract = await OracleVerifier.deploy(oracleAddresses, NUM_FAULTY, CONFIG_DIGEST);
        const verifierReceipt = await verifierContract.deploymentTransaction().wait();
        verifierAddress = await verifierContract.getAddress();
        console.log(`OracleVerifier deployed at ${verifierAddress} with gas used: ${verifierReceipt.gasUsed.toString()}`);
    });
    
    it("Deploy RoyaltyManager", async function () {
        const RoyaltyManager = await ethers.getContractFactory("RoyaltyManager");
        royaltyManagerContract = await RoyaltyManager.deploy(holderAddresses, verifierAddress);
        const royaltyManagerReceipt = await royaltyManagerContract.deploymentTransaction().wait();
        royaltyManagerAddress = await royaltyManagerContract.getAddress();
        console.log(`RoyaltyManager deployed at ${royaltyManagerAddress} with gas used: ${royaltyManagerReceipt.gasUsed.toString()}`);
    });

    it("Deploy Aggregator", async function () {
        const Aggregator = await ethers.getContractFactory("Aggregator");
        aggregatorContract = await Aggregator.deploy(
            REQUEST_FEE, 
            ORACLE_REWARD, 
            MODEL_CREATOR_REWARD,
            verifierAddress,
            queueAddress,
            royaltyManagerAddress
        );
        const aggregatorReceipt = await aggregatorContract.deploymentTransaction().wait();
        aggregatorAddress = await aggregatorContract.getAddress();
        console.log(`Aggregator deployed at ${aggregatorAddress} with gas used: ${aggregatorReceipt.gasUsed.toString()}`);
    });

    // Link the contracts together.
    it("Link contracts together", async function () {
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
    });

    
    it("Call requestAttribution", async function () {
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
        requestId = event.args.requestId;
        console.log(`Request with ID: ${requestId.toString()} created with gas used: ${requestAttributionReceipt.gasUsed.toString()}`);
    });

    it("Call approveJob", async function () {
        const approveJobTx = await aggregatorContract.approveJob(requestId);
        const approveJobReceipt = await approveJobTx.wait();
        console.log(`Approved job with ID: ${requestId.toString()} with gas used: ${approveJobReceipt.gasUsed.toString()}`);
    });

    it("Call transmit", async function () {
        const signer = signers[1]; 
        const contractAsSigner = aggregatorContract.connect(signer);
        const seqNr = 1; // example sequence number
        const jobId = 0; // example job ID
        const vector = new Array(REPORT_SIZE).fill(0); // example result vector
        vector[0] = 1;
        const reportData = ethers.solidityPacked(["uint256", "int128[]"], [jobId, vector]);
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
    });
});