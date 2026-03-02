const hre = require("hardhat");

async function main() {
  console.log("[MODEL CREATOR] Start validation and approval...");

  const queueAddress = process.env.ORACLE_QUEUE_ADDRESS || "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512";
  
  // Otteniamo gli account. signers[0] è l'Owner/Model Creator.
  const signers = await hre.ethers.getSigners();
  const creatorWallet = signers[0];

  const OracleQueue = await hre.ethers.getContractFactory("OracleQueue");
  const queueContract = OracleQueue.attach(queueAddress).connect(creatorWallet);

  console.log(`[MODEL CREATOR] Listening to LogNewCustomerRequest event...`);

  // Listening to LogNewCostumerRequest event
  queueContract.on("LogNewCustomerRequest", async (...args) => {
    // last element passed by Ethers is always the EventLog
    const requestId = args[0];  // uint256
    const ipfsCid = args[1];    // string
    const customer = args[2];   // Customer Address 
    const payment = args[3];    // Payment in wei

    const formattedEth = ethers.formatEther(payment);
    console.log(`\n[Event Received] New request from the customer!`);
    console.log(`   Job ID: ${requestId}`);
    console.log(`   CID: ${ipfsCid}`);
    console.log(`   Customer: ${customer}`);
    console.log(`   Payment: ${formattedEth} ETH`);
    
    try {
        console.log(`[MODEL CREATOR] Approval of the job #${requestId}...`);
        
        // Call automatically the function approveJob
        const tx = await queueContract.approveJob(requestId);
        console.log(`[MODEL CREATOR] Transaction sent... Hash: ${tx.hash}`);
        
        await tx.wait();
        
        console.log(`[MODEL CREATOR] Success! Event emitted. The Oracles will wake up for the job #${requestId}.`);
        
    } catch (error) {
        console.error("Error in approval phase:", error.message);
    }
})

  // This keeps the script alive in a infinite loop
  await new Promise(() => {});
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});