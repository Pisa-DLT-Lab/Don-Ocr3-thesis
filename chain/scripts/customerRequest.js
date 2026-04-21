const hre = require("hardhat");
const { resolveCustomerSigner } = require("./lib/signers");

/**
 * customerRequest.js
 * Simulates a customer placing a request by uploading data to IPFS
 * and submitting a payable transaction to the Aggregator contract.
 */
async function main() {
    console.log("[CUSTOMER] Starting request and payment workflow...");

    const aggregatorAddress = process.env.AGGREGATOR_ADDRESS;
    const { signer: customerWallet, index, signerCount } = await resolveCustomerSigner(hre, aggregatorAddress);
    console.log(`[CHAIN] Using customer signer #${index}/${signerCount - 1}: ${customerWallet.address}`);

    const aggregatorContract = await hre.ethers.getContractAt("Aggregator", aggregatorAddress, customerWallet);

    // --- PHASE 1: IPFS UPLOAD ---
    const { create } = await import('kubo-rpc-client');
    const ipfs = create({ url: process.env.IPFS_API_URL || 'http://127.0.0.1:5001' });
    
    const payload = "To be or not to be that is the question";
    let cid;

    try {
        const result = await ipfs.add(payload);
        cid = result.cid.toString();
        console.log(`[IPFS] Upload Successful. CID: ${cid}`);
    } catch (error) {
        console.error("[IPFS] Upload failed:", error.message);
        return;
    }

    // --- PHASE 2: BLOCKCHAIN SUBMISSION ---
    try {
        const payment = await aggregatorContract.queryFee();
        console.log(`[CHAIN] Sending request with ${hre.ethers.formatEther(payment)} ETH payment...`);
        
        const tx = await aggregatorContract.requestAttribution(cid, { value: payment });
        console.log(`[CHAIN] Transaction broadcasted. Hash: ${tx.hash}`);

        await tx.wait();
        console.log("[SUCCESS] Request accepted by the Smart Contract. Job state: PENDING.");
    } catch (error) {
        console.error("[CHAIN] Contract call failed:", error.message);
    }
}

main().catch(console.error);
