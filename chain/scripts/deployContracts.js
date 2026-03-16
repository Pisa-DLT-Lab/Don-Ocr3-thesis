const hre = require("hardhat");

async function main() {
    console.log("\n=======================================================");
    console.log(" STARTING DEPLOYMENT: OracleQueue & OracleVerifier");
    console.log("=======================================================\n");

    const [deployer] = await hre.ethers.getSigners();
    console.log(`Deploying contracts with the account: ${deployer.address}`);

    // =========================================================
    // 1. DEPLOY ORACLE QUEUE
    // =========================================================
    console.log("\n[1/3] Deploying OracleQueue...");
    const OracleQueue = await hre.ethers.getContractFactory("OracleQueue");
    
    // Use parseEther to convert the amount into Wei
    // Fee paid by the customer
    const feeInWei = hre.ethers.parseEther("0.02"); 
    // Reward to refund the oracle
    const rewardInWei = hre.ethers.parseEther("0.018"); 

    const queue = await OracleQueue.deploy(feeInWei, rewardInWei);
    await queue.waitForDeployment();
    const queueAddress = await queue.getAddress();
    
    console.log(`OracleQueue deployed at: ${queueAddress}`);

    // =========================================================
    // 2. DEPLOY ORACLE VERIFIER
    // =========================================================
    console.log("\n[2/3] Deploying OracleVerifier...");
    
    // Read the number of oracles from the .env file (defaults to 4 if not found)
    const NUM_ORACLES = parseInt(process.env.NUM_ORACLES || "4");

    let REAL_DIGEST;

    if (NUM_ORACLES === 7) {
    // The exact configDigest captured for the 7-node network
        REAL_DIGEST = "0x0001019032f48532003abae19a45640493e4c80e455c96c1e3144ce7e7d33319";
    } else if (NUM_ORACLES === 4) {
    // Set it for 4 nodes
        REAL_DIGEST = "0x0001eb5bd089a1e47f2e7bf9d2332c5b1d9717a4268f5441b3a61d2db8c05475"; 
    } else {
    // Safety fallback with zeros for custom configurations
        REAL_DIGEST = "0x0000000000000000000000000000000000000000000000000000000000000000";
    }

    const signers = await hre.ethers.getSigners();
    const modelCreator = signers[0]; // Wallet 0
    
    // 1. Extract wallets dynamically (starting from Wallet 1)
    const oraclesArray = [];
    for(let i = 1; i <= NUM_ORACLES; i++) {
        oraclesArray.push(signers[i].address);
    }

    // 2. Calculate the Byzantine fault tolerance f automatically
    const fValue = Math.floor((NUM_ORACLES - 1) / 3);
    
    console.log(`\nDeploying for a ${NUM_ORACLES}-node network (f=${fValue})...`);
    console.log("- ModelCreator (Deployer):", modelCreator.address);
    console.log("- Oracles Array:", oraclesArray);
    console.log("- Calculated 'f' value:", fValue);
    //console.log("- CONFIG DIGEST UTILIZZATO:", REAL_DIGEST);
      
    // DEPLOY: Pass oracles array, f, digest, and the QUEUE ADDRESS
    const OracleVerifier = await hre.ethers.getContractFactory("OracleVerifier", modelCreator);
    const verifier = await OracleVerifier.deploy(oraclesArray, fValue, REAL_DIGEST, queueAddress);
    await verifier.waitForDeployment();
    const verifierAddress = await verifier.getAddress();

    console.log(`OracleVerifier deployed at: ${verifierAddress}`);

    // =========================================================
    // 3. AUTHORIZATION (LINKING CONTRACTS)
    // =========================================================
    console.log("\n[3/3] Authorizing Verifier in the Queue...");
    
    // The Model Creator (deployer) calls the setVerifierAddress function on OracleQueue
    const authTx = await queue.setVerifierAddress(verifierAddress);
    await authTx.wait(); // Wait for the transaction to be mined
    
    console.log(`Authorization complete! Queue now trusts Verifier.`);

    console.log("\n=======================================================");
    console.log(" DEPLOYMENT FINISHED SUCCESSFULLY!");
    console.log(` QUEUE_ADDRESS=${queueAddress}`);
    console.log(` VERIFIER_ADDRESS=${verifierAddress}`);
    console.log("=======================================================\n");
}

main().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});