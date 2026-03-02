const hre = require("hardhat");

async function main() {
    console.log("[CUSTOMER] Avvio script di richiesta e pagamento...");

    // ===========================================================
    // 1. IPFS CONNECTION & UPLOAD
    // ===========================================================
    // Dynamic import is REQUIRED because 'kubo-rpc-client' is an ESM-only module
    // while Hardhat uses CommonJS by default. This fixes the "require is not defined" error.
    const { create } = await import('kubo-rpc-client');

    // Determine IPFS URL (Localhost for your PC, internal DNS for Docker)
    const ipfsUrl = process.env.IPFS_API_URL || 'http://127.0.0.1:5001';
    const ipfs = create({ url: ipfsUrl });
    
    const simplePayload = "Once upon a time...";
    let cidString = "";

    try {
        // Add the string directly to IPFS
        const { cid } = await ipfs.add(simplePayload);

        cidString = cid.toString();

        console.log("\nIPFS Upload Complete!");
        console.log("------------------------------------------------");
        console.log(` CID: ${cidString}`);
        console.log(` Verify URL: http://127.0.0.1:8080/ipfs/${cidString}`);
        console.log("------------------------------------------------\n");
    } catch (error) {
        console.error("Errore IPFS:", error.message);
        return;
    }

    // ==========================================================
    // 2. SMART CONTRACT TRIGGER 
    // ==========================================================    
    console.log("Contacting Blockchain to wake up oracles...");

    // Get Queue Contract Address from ENV 
    const queueAddress = process.env.ORACLE_QUEUE_ADDRESS || "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512";

    // Get Contract Instance via Hardhat
    const OracleQueue = await hre.ethers.getContractFactory("OracleQueue");
    
    // After hardhat init we get 20 addresses
    // signers[0] is the deployer (Model Creator).
    // To simulate a customer we use signer[5] because 1,2,3,4 are for the oracles
    const signers = await hre.ethers.getSigners();
    const customerWallet = signers[5]; 
    
    // Connect the contract to the wallet of the customer
    const queueContract = OracleQueue.attach(queueAddress).connect(customerWallet);

    try {
        console.log("[CUSTOMER] Send the request to the blockchain with the payment (0.01 ETH)...");
        
        // Define the cost of the service
        const paymentAmount = hre.ethers.parseEther("0.01");

        // Call to the payable function of the smart contract
        const tx = await queueContract.requestAttribution(cidString, { value: paymentAmount });
        console.log(`[CUSTOMER] Transaction sent... Hash: ${tx.hash}`);

        const receipt = await tx.wait();
        console.log(`[CUSTOMER] Success! The job in the queue is in PENDING state.`);
        
    } catch (error) {
        console.error("Smart Contract error:", error.message);
    }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});