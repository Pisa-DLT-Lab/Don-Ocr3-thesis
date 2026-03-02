const hre = require("hardhat");

async function main() {
  console.log("Starting Upload & Trigger Script...");

  // ===========================================================
  // 1. IPFS CONNECTION & UPLOAD
  // ===========================================================
  
  // Dynamic import is REQUIRED because 'kubo-rpc-client' is an ESM-only module
  // while Hardhat uses CommonJS by default. This fixes the "require is not defined" error.
  const { create } = await import('kubo-rpc-client');

  // Determine IPFS URL (Localhost for your PC, internal DNS for Docker)
  const ipfsUrl = process.env.IPFS_API_URL || 'http://127.0.0.1:5001';
  const ipfs = create({ url: ipfsUrl });

  // --- PAYLOAD: SIMPLE STRING ---
  const simplePayload = "Once upon a time"

  let cidString = "";

  try {
    console.log(`Uploading payload: "${simplePayload}"`);

    // Add the string directly to IPFS
    const { cid } = await ipfs.add(simplePayload);
    cidString = cid.toString();

    console.log("\nIPFS Upload Complete!");
    console.log("------------------------------------------------");
    console.log(` CID: ${cidString}`);
    console.log(` Verify URL: http://127.0.0.1:8080/ipfs/${cidString}`);
    console.log("------------------------------------------------\n");

  } catch (error) {
    console.error("IPFS Connection Error:", error.message);
    console.error("(Make sure the Docker IPFS node is running and port 5001 is mapped)");
    return;
  }

  // ==========================================================
  // 2. SMART CONTRACT TRIGGER 
  // ==========================================================
  
  console.log("Contacting Blockchain to wake up oracles...");

  // Get Queue Contract Address from ENV (Check your docker logs if this fails!)
  const queueAddress = process.env.ORACLE_QUEUE_ADDRESS || "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512";

  // Get Contract Instance via Hardhat
  const OracleQueue = await hre.ethers.getContractFactory("OracleQueue");
  const queueContract = OracleQueue.attach(queueAddress);

  try {
    // Call the requestComputation function with the CID
    const tx = await queueContract.requestComputation(cidString);
    console.log(`Transaction sent... Hash: ${tx.hash}`);

    // Wait for block confirmation
    await tx.wait();

    console.log("SUCCESS! Event emitted. Watch your Oracle logs now!");

  } catch (error) {
    console.error("Smart Contract Interaction Error:", error.message);
  }
}

// Handle async/await errors
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});