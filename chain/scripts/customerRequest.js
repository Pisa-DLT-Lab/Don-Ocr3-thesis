const hre = require("hardhat");
const { resolveCustomerSigner } = require("./lib/signers");

const FILTER_POLICIES = {
    TOP_VALUES: 0,
    TOP_HOLDERS: 1,
};

function argValue(name) {
    const prefix = `${name}=`;
    const found = process.argv.slice(2).find((arg) => arg.startsWith(prefix));
    if (found) {
        return found.slice(prefix.length);
    }
    const index = process.argv.indexOf(name);
    if (index !== -1 && process.argv[index + 1]) {
        return process.argv[index + 1];
    }
    return null;
}

function resolveRequestedFilterPolicy() {
    const raw = argValue("--filter-policy") || process.env.FILTER_POLICY_REQUEST;
    if (!raw || raw.trim() === "") {
        return null;
    }

    const normalized = raw.trim().toUpperCase();
    if (normalized === "0") {
        return { name: "TOP_VALUES", value: FILTER_POLICIES.TOP_VALUES };
    }
    if (normalized === "1") {
        return { name: "TOP_HOLDERS", value: FILTER_POLICIES.TOP_HOLDERS };
    }
    if (Object.prototype.hasOwnProperty.call(FILTER_POLICIES, normalized)) {
        return { name: normalized, value: FILTER_POLICIES[normalized] };
    }
    throw new Error(`Invalid filter policy: ${raw}. Use TOP_VALUES or TOP_HOLDERS.`);
}

function resolveRequestedThreshold() {
    const raw = argValue("--threshold") || process.env.FILTER_THRESHOLD_REQUEST;
    if (!raw || raw.trim() === "") {
        return null;
    }
    const trimmed = raw.trim();
    if (!/^\d+$/.test(trimmed)) {
        throw new Error(`Invalid threshold: ${raw}. Use a non-negative integer.`);
    }
    return BigInt(trimmed);
}

async function maybeSetFilterPolicy(aggregatorAddress) {
    const requestedPolicy = resolveRequestedFilterPolicy();
    const requestedThreshold = resolveRequestedThreshold();
    if (requestedPolicy === null && requestedThreshold === null) {
        return;
    }

    const [ownerWallet] = await hre.ethers.getSigners();
    const aggregatorAsOwner = await hre.ethers.getContractAt("Aggregator", aggregatorAddress, ownerWallet);
    const [currentPolicy, currentThreshold] = await aggregatorAsOwner.getFilterPolicy();

    const policy = requestedPolicy || {
        name: Number(currentPolicy) === FILTER_POLICIES.TOP_HOLDERS ? "TOP_HOLDERS" : "TOP_VALUES",
        value: Number(currentPolicy),
    };
    const threshold = requestedThreshold === null ? currentThreshold : requestedThreshold;

    console.log(`[CHAIN] Setting filter policy before request: ${policy.name}, threshold=${threshold.toString()}`);
    const tx = await aggregatorAsOwner.setFilterPolicy(policy.value, threshold);
    await tx.wait();
}

/**
 * customerRequest.js
 * Simulates a customer placing a request by uploading data to IPFS
 * and submitting a payable transaction to the Aggregator contract.
 */
async function main() {
    console.log("[CUSTOMER] Starting request and payment workflow...");

    const aggregatorAddress = process.env.AGGREGATOR_ADDRESS;
    await maybeSetFilterPolicy(aggregatorAddress);

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
