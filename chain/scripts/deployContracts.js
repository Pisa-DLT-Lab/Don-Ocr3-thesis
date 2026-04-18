const hre = require("hardhat");
// Use parseEther to convert the amount into Wei
const REQUEST_FEE = hre.ethers.parseEther("0.01"); // Fee paid by the customer
const ORACLE_REWARD = hre.ethers.parseEther("0.003"); // Reward to refund the oracle
const MODEL_CREATOR_REWARD = hre.ethers.parseEther("0.002"); // Reward for the model creator
const NUM_HOLDERS = 1000; // Number of data holders to simulate in the RoyaltyManager.
const FILTER_POLICIES = {
    TOP_VALUES: 0,
    TOP_HOLDERS: 1,
};

// Generates a random Ethereum address (for testing purposes only).
function randomAddress() {
    return hre.ethers.Wallet.createRandom().address;
}

function validateConfigDigest(digest) {
    const value = digest.startsWith("0x") || digest.startsWith("0X")
        ? digest.slice(2)
        : digest;

    if (!/^[0-9a-fA-F]{64}$/.test(value)) {
        throw new Error(`Invalid CONFIG_DIGEST: ${digest}`);
    }

    return `0x${value.toLowerCase()}`;
}

function resolveConfigDigest(numOracles) {
    if (process.env.CONFIG_DIGEST) {
        return validateConfigDigest(process.env.CONFIG_DIGEST);
    }

    if (numOracles === 7) {
        // The exact configDigest captured for the original 7-node network.
        return "0x0001019032f48532003abae19a45640493e4c80e455c96c1e3144ce7e7d33319";
    }

    if (numOracles === 4) {
        // The exact configDigest captured for the original 4-node network.
        return "0x0001eb5bd089a1e47f2e7bf9d2332c5b1d9717a4268f5441b3a61d2db8c05475";
    }

    // Safety fallback for custom configurations when no generated digest is supplied.
    return "0x0000000000000000000000000000000000000000000000000000000000000000";
}

function resolveFilterPolicy() {
    const rawPolicy = (process.env.FILTER_POLICY || "TOP_HOLDERS").trim().toUpperCase();
    if (rawPolicy === "0") {
        return { name: "TOP_VALUES", value: FILTER_POLICIES.TOP_VALUES };
    }
    if (rawPolicy === "1") {
        return { name: "TOP_HOLDERS", value: FILTER_POLICIES.TOP_HOLDERS };
    }
    if (Object.prototype.hasOwnProperty.call(FILTER_POLICIES, rawPolicy)) {
        return { name: rawPolicy, value: FILTER_POLICIES[rawPolicy] };
    }
    throw new Error(`Invalid FILTER_POLICY: ${process.env.FILTER_POLICY}. Use TOP_HOLDERS or TOP_VALUES.`);
}

function resolveFilterThreshold() {
    const rawThreshold = (process.env.FILTER_THRESHOLD || "100").trim();
    if (!/^[0-9]+$/.test(rawThreshold)) {
        throw new Error(`Invalid FILTER_THRESHOLD: ${rawThreshold}. Use a non-negative integer.`);
    }
    return BigInt(rawThreshold);
}

async function assertAggregatorWiring(aggregator, expectedQueue, expectedVerifier, expectedManager) {
    const deployedQueue = await aggregator.queue();
    const deployedVerifier = await aggregator.verifier();
    const deployedManager = await aggregator.manager();

    console.log("Aggregator.queue():", deployedQueue);
    console.log("Aggregator.verifier():", deployedVerifier);
    console.log("Aggregator.manager():", deployedManager);

    if (deployedQueue.toLowerCase() !== expectedQueue.toLowerCase()) {
        throw new Error(`Aggregator.queue mismatch: expected ${expectedQueue}, got ${deployedQueue}`);
    }
    if (deployedVerifier.toLowerCase() !== expectedVerifier.toLowerCase()) {
        throw new Error(`Aggregator.verifier mismatch: expected ${expectedVerifier}, got ${deployedVerifier}`);
    }
    if (deployedManager.toLowerCase() !== expectedManager.toLowerCase()) {
        throw new Error(`Aggregator.manager mismatch: expected ${expectedManager}, got ${deployedManager}`);
    }
}

async function assertFilterPolicy(aggregator, expectedPolicy, expectedThreshold) {
    const [actualPolicy, actualThreshold] = await aggregator.getFilterPolicy();
    const actualPolicyNumber = Number(actualPolicy);

    console.log("Aggregator.getFilterPolicy():", actualPolicyNumber, actualThreshold.toString());

    if (actualPolicyNumber !== expectedPolicy.value) {
        throw new Error(`Aggregator filter policy mismatch: expected ${expectedPolicy.value}, got ${actualPolicyNumber}`);
    }
    if (actualThreshold.toString() !== expectedThreshold.toString()) {
        throw new Error(`Aggregator filter threshold mismatch: expected ${expectedThreshold.toString()}, got ${actualThreshold.toString()}`);
    }
}

async function assertChildAggregatorLinks(queue, verifier, royaltyManager, expectedAggregator) {
    const queueAggregator = await queue.aggregator();
    const verifierAggregator = await verifier.aggregator();
    const royaltyManagerAggregator = await royaltyManager.aggregator();

    console.log("OracleQueue.aggregator():", queueAggregator);
    console.log("OracleVerifier.aggregator():", verifierAggregator);
    console.log("RoyaltyManager.aggregator():", royaltyManagerAggregator);

    if (queueAggregator.toLowerCase() !== expectedAggregator.toLowerCase()) {
        throw new Error(`OracleQueue aggregator mismatch: expected ${expectedAggregator}, got ${queueAggregator}`);
    }
    if (verifierAggregator.toLowerCase() !== expectedAggregator.toLowerCase()) {
        throw new Error(`OracleVerifier aggregator mismatch: expected ${expectedAggregator}, got ${verifierAggregator}`);
    }
    if (royaltyManagerAggregator.toLowerCase() !== expectedAggregator.toLowerCase()) {
        throw new Error(`RoyaltyManager aggregator mismatch: expected ${expectedAggregator}, got ${royaltyManagerAggregator}`);
    }
}


async function main() {
    const [deployer] = await hre.ethers.getSigners();
    console.log("\n");
    console.log(`Deploying contracts with the account: ${deployer.address}`);
    console.log("\n");

    // =========================================================
    // DEPLOY ORACLE QUEUE
    // =========================================================
    console.log("Deploying OracleQueue...");
    const OracleQueue = await hre.ethers.getContractFactory("OracleQueue");
    const queue = await OracleQueue.deploy();
    const queueReceipt = await queue.deploymentTransaction().wait();
    const queueAddress = await queue.getAddress();
    console.log("OracleQueue deployed to:", queueAddress);
    console.log("Gas used:", queueReceipt.gasUsed.toString());

    console.log("\n");

    // =========================================================
    // DEPLOY ORACLE VERIFIER
    // =========================================================
    console.log("Deploying OracleVerifier...");
    // Read the number of oracles from the .env file (defaults to 4 if not found)
    const NUM_ORACLES = 4; //parseInt(process.env.NUM_ORACLES || "4");
    const REAL_DIGEST = resolveConfigDigest(NUM_ORACLES);
    const signers = await hre.ethers.getSigners();
    const modelCreator = signers[0]; // Wallet 0
    // 1. Extract wallets dynamically (starting from Wallet 1)
    const oraclesArray = [];
    for(let i = 1; i <= NUM_ORACLES; i++) {
        oraclesArray.push(signers[i].address);
    }
    // 2. Calculate the Byzantine fault tolerance f automatically
    const fValue = parseInt(process.env.FAULT_TOLERANCE || `${Math.floor((NUM_ORACLES - 1) / 3)}`, 10);    
    console.log(`Deploying for a ${NUM_ORACLES}-node network (f=${fValue})...`);
    console.log("- ModelCreator (Deployer):", modelCreator.address);
    console.log("- Oracles Array:", oraclesArray);
    console.log("- Calculated 'f' value:", fValue);
    console.log("- Config digest:", REAL_DIGEST);      
    // DEPLOY: Pass oracles array, f, digest, and the QUEUE ADDRESS
    const OracleVerifier = await hre.ethers.getContractFactory("OracleVerifier", modelCreator);
    const verifier = await OracleVerifier.deploy(oraclesArray, fValue, REAL_DIGEST);
    const verifierReceipt = await verifier.deploymentTransaction().wait();
    const verifierAddress = await verifier.getAddress();
    console.log("OracleVerifier deployed to:", verifierAddress);
    console.log("Gas used:", verifierReceipt.gasUsed.toString());

    console.log("\n");

    // =========================================================
    // DEPLOY ROYALTYMANAGER CONTRACT
    // =========================================================
    console.log("Deploying RoyaltyManager...");
    console.log(`Number of data holders: ${NUM_HOLDERS}`);
    const holdersArray = Array.from({ length: NUM_HOLDERS }, randomAddress);
    const RoyaltyManager = await hre.ethers.getContractFactory("RoyaltyManager", modelCreator);
    const royaltyManager = await RoyaltyManager.deploy(holdersArray, verifierAddress);
    const royaltyManagerReceipt = await royaltyManager.deploymentTransaction().wait();
    const royaltyManagerAddress = await royaltyManager.getAddress();
    console.log("RoyaltyManager deployed to:", royaltyManagerAddress);
    console.log("Gas used:", royaltyManagerReceipt.gasUsed.toString());

    console.log("\n");

    // =========================================================
    // DEPLOY AGGREGATOR CONTRACT
    // =========================================================
    console.log("Deploying Aggregator...");
    const Aggregator = await hre.ethers.getContractFactory("Aggregator", modelCreator);
    const xaggregator = await Aggregator.deploy(
        REQUEST_FEE, 
        ORACLE_REWARD, 
        MODEL_CREATOR_REWARD, 
        verifierAddress, 
        queueAddress, 
        royaltyManagerAddress
    );
    const aggregatorReceipt = await xaggregator.deploymentTransaction().wait();
    const aggregatorAddress = await xaggregator.getAddress();
    console.log("Aggregator deployed to:", aggregatorAddress);
    console.log("Gas used:", aggregatorReceipt.gasUsed.toString());
    await assertAggregatorWiring(xaggregator, queueAddress, verifierAddress, royaltyManagerAddress);

    console.log("\n");

    // =========================================================
    // LINKING CONTRACTS
    // =========================================================
    console.log("Linking Aggregator in Queue...");
    const tx1 = await queue.setAggregator(aggregatorAddress);
    const receipt1 = await tx1.wait();
    console.log("Transaction hash:", tx1.hash);
    console.log("Gas used:", receipt1.gasUsed.toString());
    console.log("Linking Aggregator in Verifier...");
    const tx2 = await verifier.setAggregator(aggregatorAddress);
    const receipt2 = await tx2.wait();
    console.log("Transaction hash:", tx2.hash);
    console.log("Gas used:", receipt2.gasUsed.toString());
    console.log("Linking Aggregator in RoyaltyManager...");
    const tx3 = await royaltyManager.setAggregator(aggregatorAddress);
    const receipt3 = await tx3.wait();
    console.log("Transaction hash:", tx3.hash);
    console.log("Gas used:", receipt3.gasUsed.toString());
    await assertChildAggregatorLinks(queue, verifier, royaltyManager, aggregatorAddress);

    console.log("Setting Aggregator filter policy...");
    const filterPolicy = resolveFilterPolicy();
    const filterThreshold = resolveFilterThreshold();
    const tx4 = await xaggregator.setFilterPolicy(filterPolicy.value, filterThreshold);
    const receipt4 = await tx4.wait();
    console.log(`Filter policy: ${filterPolicy.name}, threshold: ${filterThreshold.toString()}`);
    console.log("Transaction hash:", tx4.hash);
    console.log("Gas used:", receipt4.gasUsed.toString());
    await assertFilterPolicy(xaggregator, filterPolicy, filterThreshold);


    console.log("\n=======================================================");
    console.log(" DEPLOYMENT COMPLETED SUCCESSFULLY!");
    console.log(` QUEUE_ADDRESS=${queueAddress}`);
    console.log(` VERIFIER_ADDRESS=${verifierAddress}`);
    console.log(` ROYALTY_MANAGER_ADDRESS=${royaltyManagerAddress}`);
    console.log(` AGGREGATOR_ADDRESS=${aggregatorAddress}`);
    console.log(` FILTER_POLICY=${filterPolicy.name}`);
    console.log(` FILTER_THRESHOLD=${filterThreshold.toString()}`);
    console.log("=======================================================\n");
}

main().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});
