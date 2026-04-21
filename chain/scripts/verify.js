const hre = require("hardhat");
require("dotenv").config({ path: "../.env" });

const SCORE_BITS = 96n;
const SCORE_MASK = (1n << SCORE_BITS) - 1n;

function decodeHolderScore(packedValue) {
  const packed = BigInt(packedValue.toString());
  if (packed < 0n) {
    throw new Error(`Packed holder-score cannot be negative: ${packed}`);
  }
  return {
    holderId: packed >> SCORE_BITS,
    score: packed & SCORE_MASK,
  };
}

async function main() {
  const CONTRACT_ADDR = process.env.AGGREGATOR_ADDRESS;

  console.log(`\nVerifying Oracle Data on Chain`);
  console.log(`Target Contract: ${CONTRACT_ADDR}`);

  // Connection to the contracts. Aggregator is the root address; Verifier stores results.
  const aggregator = await hre.ethers.getContractAt("Aggregator", CONTRACT_ADDR);
  const verifierAddress = await aggregator.verifier();
  const verifier = await hre.ethers.getContractAt("OracleVerifier", verifierAddress);

  try {
    // 1. Find the ID
    console.log("   -> Searching for 'JobCompleted' events...");
    
    // Create filter for the event
    const filter = verifier.filters.JobCompleted();
    // Obtain the history of the events from the generation of block 0
    const events = await verifier.queryFilter(filter);

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
    const result = await verifier.getResult(requestId);
    
    const flatMatrix = result[0]; // BigInt Array
    const submitter = result[1];
    const timestamp = result[2];

    // 3. Convertion and visualization
    console.log("\nData Analysis:");
    console.log(`   - Timestamp: ${new Date(Number(timestamp) * 1000).toLocaleString()}`);
    console.log(`   - Saved By:  ${submitter}`);
    console.log(`   - Array Len: ${flatMatrix.length} items`);

    console.log("\nDecoded Final Scores:");
    console.log("   ------------------------------------");

    for (let i = 0; i < flatMatrix.length; i++) {
        const { holderId, score } = decodeHolderScore(flatMatrix[i]);
        const scoreFixed = hre.ethers.formatUnits(score, 18);
        console.log(`   [${i}]: holder=${holderId.toString()} score=${scoreFixed}`);
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
