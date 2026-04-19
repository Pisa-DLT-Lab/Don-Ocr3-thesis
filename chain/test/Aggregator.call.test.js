// This test measures the gas used for calling the functions of the Aggregator contract.

const { ethers } = require("hardhat");
const REQUEST_FEE = ethers.parseEther("0.01"); // Fee paid by the customer
const ORACLE_REWARD = ethers.parseEther("0.003"); // Reward to refund the oracle
const MODEL_CREATOR_REWARD = ethers.parseEther("0.002"); // Reward for the model creator
const NUM_ORACLES = 4;
const NUM_FAULTY = 1;
const CONFIG_DIGEST = "0x0001eb5bd089a1e47f2e7bf9d2332c5b1d9717a4268f5441b3a61d2db8c05475";
const NUM_HOLDERS = 100;
const IPFS_EXAMPLE_CID = "bafybeigrf2dwtpjkiovnigysyto3d55opf6qkdikx6d65onrqnfzwgdkfa"

// Generates an array of random Ethereum addresses of the specified size.
function generateAddressArray(size) {
    const array = [];
    for (let i = 0; i < size; i++) {
        array.push(ethers.Wallet.createRandom().address);
    }
    return array;
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

    before(async () => {
        signers = await ethers.getSigners();
        modelCreatorAddress = signers[0].address;
        for (let i = 1; i <= NUM_ORACLES; i++) {
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

    
});