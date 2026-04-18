// This test measures the gas used for deploying the OracleVerifier contract 
// with different sizes of the input array (which contains the oracle addresses).

const { ethers } = require("hardhat");
const CONTRACT_NAME = "OracleVerifier";
const CONFIG_DIGEST = "0x0000000000000000000000000000000000000000000000000000000000000000";
const F_MAX = 40;

// Generates a random Ethereum address (for testing purposes only).
function randomAddress() {
    return ethers.Wallet.createRandom().address;
}

// Generates an array of random Ethereum addresses of the specified size.
function generateAddressArray(size) {
    const array = [];
    for (let i = 0; i < size; i++) {
        array.push(randomAddress());
    }
    return array;
}

describe("OracleVerifier Deployment Gas", function () {
    let oracleVerifier;
    before(async () => {
        oracleVerifier = await ethers.getContractFactory(CONTRACT_NAME);
    });

    async function deployAndMeasure(f) {
        const size = 3 * f + 1;
        const addressArray = generateAddressArray(size);
        const contract = await oracleVerifier.deploy(addressArray, f, CONFIG_DIGEST);
        const receipt = await contract.deploymentTransaction().wait();
        return receipt.gasUsed;
    }

    it("Deploys OracleVerifier for different values of f", async () => {
        for (let f = 1; f <= F_MAX; f++) {
            const gasUsed = await deployAndMeasure(f);
            console.log(`${f}\t${gasUsed.toString()}`);
        }
    });
});