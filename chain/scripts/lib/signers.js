function parseSignerIndex(envName) {
    const raw = process.env[envName];
    if (raw === undefined || raw.trim() === "") {
        return null;
    }

    if (!/^\d+$/.test(raw.trim())) {
        throw new Error(`${envName} must be a non-negative integer, got '${raw}'`);
    }

    return Number(raw);
}

async function resolveCustomerSigner(hre, aggregatorAddress, options = {}) {
    if (!aggregatorAddress || !hre.ethers.isAddress(aggregatorAddress)) {
        throw new Error(`Invalid AGGREGATOR_ADDRESS: ${aggregatorAddress || "<empty>"}`);
    }

    const signers = await hre.ethers.getSigners();
    if (signers.length === 0) {
        throw new Error("No unlocked signers are available on the selected Hardhat network");
    }

    const envName = options.envName || "CUSTOMER_SIGNER_INDEX";
    const envIndex = parseSignerIndex(envName);
    if (envIndex !== null) {
        const signer = signers[envIndex];
        if (!signer) {
            throw new Error(`${envName}=${envIndex} is unavailable; RPC exposes ${signers.length} signer(s)`);
        }
        return { signer, index: envIndex, signerCount: signers.length };
    }

    const reader = await hre.ethers.getContractAt("Aggregator", aggregatorAddress, signers[0]);
    const verifier = await hre.ethers.getContractAt("OracleVerifier", await reader.verifier(), signers[0]);

    let oracleCount = 0;
    for (let i = 1; i < signers.length; i++) {
        if (!(await verifier.isOracle(signers[i].address))) {
            break;
        }
        oracleCount++;
    }

    const customerIndex = oracleCount + 1;
    const signer = signers[customerIndex];
    if (!signer) {
        throw new Error(
            `Missing customer signer #${customerIndex}. Detected ${oracleCount} oracle signer(s), ` +
            `so the local chain must expose at least ${customerIndex + 1} accounts ` +
            `(deployer + oracles + customer). Regenerate the Hardhat config and restart the chain.`
        );
    }

    if (await verifier.isOracle(signer.address)) {
        throw new Error(`Configured customer signer #${customerIndex} is also an oracle: ${signer.address}`);
    }

    return { signer, index: customerIndex, signerCount: signers.length };
}

module.exports = {
    resolveCustomerSigner,
};
