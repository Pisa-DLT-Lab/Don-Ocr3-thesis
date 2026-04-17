const hre = require("hardhat");
const { ethers } = hre;

/**
 * ModelCreatorApprove.js
 * Acts as the 'Model Creator' authority. It monitors the Aggregator for new
 * customer requests and formally approves them to trigger the DON consensus.
 */
async function main() {
  console.log("[MODEL CREATOR] Initializing validation and approval service...");

  const aggregatorAddress = process.env.AGGREGATOR_ADDRESS;

  const [creatorWallet] = await hre.ethers.getSigners();

  // Attach to the Aggregator using the Creator's identity
  const aggregatorContract = (await hre.ethers.getContractAt("Aggregator", aggregatorAddress)).connect(creatorWallet);

  console.log(`[MODEL CREATOR] Monitoring 'LogNewCustomerRequest' events...`);

  const MAX_RETRIES = 3;
  
  // Sequential Job Queue: ensures jobs are approved and finalized one by one
  // to prevent nonce collisions and maintain deterministic benchmark results.
  let jobProcessingPipeline = Promise.resolve();

  aggregatorContract.on("LogNewCustomerRequest", async (requestId, ipfsCid, customer, payment) => {
    console.log(`\n[EVENT] New Job Detected: #${requestId}`);
    console.log(`       CID:      ${ipfsCid}`);
    console.log(`       Value:    ${ethers.formatEther(payment)} ETH`);

    // Add job to the sequential pipeline
    jobProcessingPipeline = jobProcessingPipeline.then(async () => {
      
      // --- STEP 1: ON-CHAIN APPROVAL ---
      for (let attempt = 1; attempt <= MAX_RETRIES; attempt++) {
        try {
          console.log(`[PROCESS] Approving job #${requestId} (Attempt ${attempt}/${MAX_RETRIES})...`);
          const tx = await aggregatorContract.approveJob(requestId);
          const receipt = await tx.wait();
          let oracleJobId = requestId;
          for (const log of receipt.logs) {
            try {
              const parsed = aggregatorContract.interface.parseLog(log);
              if (parsed.name === "LogNewJobForOracles") {
                oracleJobId = parsed.args[0];
                break;
              }
            } catch (e) { /* Skip unrelated logs */ }
          }
          console.log(`[SUCCESS] Request #${requestId} approved as oracle job #${oracleJobId}.`);
          requestId = oracleJobId;
          break;
        } catch (error) {
          if (attempt === MAX_RETRIES) {
            console.error(`[ERROR] Job #${requestId} approval failed permanently: ${error.message}`);
            return;
          }
          await new Promise(res => setTimeout(res, 2000));
        }
      }

      // --- STEP 2: WAIT FOR DON FULFILLMENT ---
      // We use a Promise with .once() to wait for the consensus result to land on-chain.
      console.log(`[WAIT] Awaiting OCR consensus for job #${requestId}...`);
      await new Promise((resolve, reject) => {
        let completionListener;
        const timeout = setTimeout(() => {
          aggregatorContract.off("JobCompleted", completionListener);
          reject(new Error("OCR fulfillment timeout (10m)"));
        }, 600000);

        completionListener = (completedId, submitter) => {
          if (completedId.toString() === requestId.toString()) {
            clearTimeout(timeout);
            aggregatorContract.off("JobCompleted", completionListener);
            console.log(`[DONE] Job #${requestId} finalized by Oracle: ${submitter}.`);
            resolve();
          }
        };

        aggregatorContract.on("JobCompleted", completionListener);
      });
    });
  });

  // Keep the process alive
  await new Promise(() => {});
}

main().catch((error) => {
  console.error("Fatal service error:", error);
  process.exit(1);
});
