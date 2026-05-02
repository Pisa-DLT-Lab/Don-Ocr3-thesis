/**
 * This script can be used to perform a benchmark of the DON latency by automating the entire request lifecycle.
 * It simulates a customer submitting a request with an IPFS-stored payload, waits for the model creator to approve it,
 * and then listens for the final fulfillment event.
 */

const hre = require("hardhat");
const fs = require("fs");
const yargs = require("yargs/yargs");
const { hideBin } = require("yargs/helpers");
const { resolveCustomerSigner } = require("./lib/signers");

// Parse command line arguments.
const args = yargs(hideBin(process.argv))
    .option("numRequests", {
        description: "Number of requests to perform",
        type: "number",
        default: 15
    })
    .option("outputFile", {
        description: "Path of the output file",
        type: "string",
        default: "benchmark_results.csv"
    })
    .argv;

const NUM_REQUESTS = args.numRequests; // Number of requests to perform.
const OUTPUT_FILE = args.outputFile; // Path of the output file.

async function main() {
    console.log("[BENCHMARK] Starting automated DON latency evaluation...\n");

    // Dynamic import for 'kubo-rpc-client' (ESM module compatibility)
    const { create } = await import('kubo-rpc-client');
    const ipfsUrl = process.env.IPFS_API_URL || 'http://127.0.0.1:5001';
    const ipfs = create({ url: ipfsUrl });

    const aggregatorAddress = process.env.AGGREGATOR_ADDRESS;

    const { signer: customerWallet, index, signerCount } = await resolveCustomerSigner(hre, aggregatorAddress);
    console.log(`[CHAIN] Using customer signer #${index}/${signerCount - 1}: ${customerWallet.address}`);

    const aggregatorContract = await hre.ethers.getContractAt("Aggregator", aggregatorAddress, customerWallet);
    const queueAddress = await aggregatorContract.queue();
    const verifierAddress = await aggregatorContract.verifier();
    const queueContract = await hre.ethers.getContractAt("OracleQueue", queueAddress);
    const verifierContract = await hre.ethers.getContractAt("OracleVerifier", verifierAddress);

    // Initialize CSV Telemetry Data
    const csvHeader = "Iteration,Timestamp,OffChain_Storage(ms),OnChain_RequestTx(ms),OnChain_ApprovalTx(ms),OCR_Consensus_And_FulfillmentTx(ms),Total_Latency(ms)\n";
    fs.writeFileSync(OUTPUT_FILE, csvHeader);
    const promptVariants = [
        "To be or not to be that is the question",
        "O brave new world that has such people in it",
        "The course of true love never did run smooth",
        "All the world is a stage and all the men and women merely players",
    ];

    for (let i = 1; i <= NUM_REQUESTS; i++) {
        console.log(`========================================`);
        console.log(` ITERATION ${i} / ${NUM_REQUESTS}`);
        console.log(`========================================`);

        const simplePayload = promptVariants[(i - 1) % promptVariants.length];
        const t0 = performance.now();

        // ---------------------------------------------------------------------
        // PHASE 1: Off-Chain Storage (IPFS Upload)
        // ---------------------------------------------------------------------
        process.stdout.write('[1/4] Off-Chain Storage (IPFS Upload)... ');
        const { cid } = await ipfs.add(simplePayload);
        const cidString = cid.toString();
        const tIpfs = performance.now();
        console.log(`[+] ${(tIpfs - t0).toFixed(2)} ms | CID: ${cidString}`);

        // ---------------------------------------------------------------------
        // PHASE 2: Customer Request (On-Chain Transaction)
        // ---------------------------------------------------------------------
        // Pre-register listener for the Approval event to avoid race conditions
        const approvalPromise = new Promise((resolve, reject) => {
            const timeout = setTimeout(() => reject(new Error("Timeout: LogNewJobForOracles event missed")), 45000);
            queueContract.once("LogNewJobForOracles", (jobId) => {
                clearTimeout(timeout);
                resolve(jobId);
            });
        });

        process.stdout.write(`[2/4] On-Chain Customer Request (Tx)... `);
        const paymentAmount = await aggregatorContract.queryFee();
        
        const tx = await aggregatorContract.requestAttribution(cidString, { value: paymentAmount });
        const receipt = await tx.wait();
        const tPhase1 = performance.now();
        console.log(`[+] ${(tPhase1 - tIpfs).toFixed(0)} ms`);

        // Extract RequestID from the transaction logs
        let currentJobId = null;
        for (const log of receipt.logs) {
            try {
                const parsed = queueContract.interface.parseLog(log);
                if (parsed.name === "LogNewCustomerRequest") {
                    currentJobId = parsed.args[0];
                    break;
                }
            } catch (e) { /* Skip unrelated logs */ }
        }

        // ---------------------------------------------------------------------
        // PHASE 3: Validation & Approval (Model Creator Tx)
        // ---------------------------------------------------------------------
        process.stdout.write(`[3/4] On-Chain Approval (Model Creator Tx)... `);
        const approvedJobId = await approvalPromise; // Resolves when the separate Creator script approves the job
        const tPhase2 = performance.now();
        console.log(`[+] ${(tPhase2 - tPhase1).toFixed(2)} ms`);

        // ---------------------------------------------------------------------
        // PHASE 4: OCR Network Execution (AI Inference + P2P Consensus)
        // ---------------------------------------------------------------------
        console.log(`[4/4] Off-Chain Reporting (AI + BFT Consensus)... Waiting`);
        let winnerAddress = "";

        await new Promise((resolve, reject) => {
            const timeout = setTimeout(() => {
                reject(new Error(`Timeout: JobCompleted event missed for Job #${currentJobId}`));
            }, 600000); // 10-minute threshold for AI computation and P2P rounds

            const fulfillmentListener = (jobId, submitter) => {
                if (approvedJobId !== null && jobId.toString() === approvedJobId.toString()) {
                    clearTimeout(timeout);
                    winnerAddress = submitter;
                    verifierContract.off("JobCompleted", fulfillmentListener);
                    resolve();
                }
            };
            verifierContract.on("JobCompleted", fulfillmentListener);
        });

        const tPhase3 = performance.now();
        console.log(`      OCR Fulfillment On-Chain Tx detected in ${(tPhase3 - tPhase2).toFixed(2)} ms`);
        console.log(`      Transmitter Node Identity: ${winnerAddress}\n`);

        // ---------------------------------------------------------------------
        // TELEMETRY EXPORT
        // ---------------------------------------------------------------------
        const timeIpfs     = (tIpfs   - t0).toFixed(2);
        const timePhase1   = (tPhase1 - tIpfs).toFixed(2);
        const timePhase2   = (tPhase2 - tPhase1).toFixed(2);
        const timePhase3   = (tPhase3 - tPhase2).toFixed(2);
        const totalTime    = (tPhase3 - t0).toFixed(2);

        const csvRow = `${i},${Date.now()},${timeIpfs},${timePhase1},${timePhase2},${timePhase3},${totalTime}\n`;
        fs.appendFileSync(OUTPUT_FILE, csvRow);
        
        console.log(`[OK] Iteration ${i} saved. Total Latency: ${totalTime} ms\n`);

        // RPC Cooldown period
        await new Promise(resolve => setTimeout(resolve, 3000));
    }
    
    console.log(`\n Benchmark completed successfully! Data exported to: ${OUTPUT_FILE}`);
}

main().catch((error) => {
    console.error("Benchmark encountered a fatal error:", error.message);
    process.exit(1);
});
