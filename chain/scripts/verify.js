const hre = require("hardhat");
require("dotenv").config({ path: "../.env" });

// Function to verify the first numbers of the array
async function main() {
  const CONTRACT_ADDR = process.env.AGGREGATOR_ADDRESS;

  console.log(`\nVerifying Oracle Data on Chain`);
  console.log(`Target Contract: ${CONTRACT_ADDR}`);

  // Connection to the contract
  const aggregator = await hre.ethers.getContractAt("Aggregator", CONTRACT_ADDR);

  try {
    // 1. Find the ID
    console.log("   -> Searching for 'JobCompleted' events...");
    
    // Create filter for the event
    const filter = aggregator.filters.JobCompleted();
    // Obtain the history of the events from the generation of block 0
    const events = await aggregator.queryFilter(filter);

    if (events.length === 0) {
        console.log("No outcomes found on chain yet.");
        return;
    }

    // Take the last event
    const latestEvent = events[events.length - 1];
    const requestId = latestEvent.args[0]; // First arg of the event is the req ID
    const submitterFromEvent = latestEvent.args[1];
    const lengthFromEvent = latestEvent.args[2];

    console.log(`\nFound latest Job ID: #${requestId}`);
    console.log(`   Submitter: ${submitterFromEvent}`);
    console.log(`   Declared Length: ${lengthFromEvent}`);

    // 2. Call the getResult function with the ID found
    console.log(`\nFetching data from Storage (getResult)...`);
    
    // Returns: [flatMatrix, submitter, timestamp]
    const result = await aggregator.getResult(requestId);
    
    const flatMatrix = result[0]; // BigInt Array
    const submitter = result[1];
    const timestamp = result[2];

    // 3. Convertion and visualization
    console.log("\nData Analysis:");
    console.log(`   - Timestamp: ${new Date(Number(timestamp) * 1000).toLocaleString()}`);
    console.log(`   - Saved By:  ${submitter}`);
    console.log(`   - Array Len: ${flatMatrix.length} items`);

    console.log("\nDecoded Values (First 10 items):");
    console.log("   (Converting fixed-point 1e18 to Float)");
    console.log("   ----------------------------------------");

    // Show the first 10 int, avoid to dipslay 2500 int
    const limit = Math.min(flatMatrix.length, 10);
    
    for (let i = 0; i < limit; i++) {
        // Data arrive as BigInt (es. 1000000000000000000n)
        // Use formatUnits to divide for 10^18; replaces the .
        const floatValue = hre.ethers.formatUnits(flatMatrix[i], 18);
        console.log(`   [${i}]: ${floatValue}`);
    }

    if (flatMatrix.length > 10) {
        console.log(`   ... (+ other ${flatMatrix.length - 10} items hidden)`);
        
        // Print the last for confirm
        const lastVal = hre.ethers.formatUnits(flatMatrix[flatMatrix.length - 1], 18);
        console.log(`   [LAST]: ${lastVal}`);
    }

    console.log("\nSUCCESS: Data verified on-chain!");

  } catch (error) {
    console.error("\nError:");
    // If Result not found, means that the ID doesn't exist or saved=false
    if (error.reason) console.error("Reason:", error.reason);
    console.error(error);
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
