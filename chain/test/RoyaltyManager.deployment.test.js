// This test measures the gas used for deploying the RoyaltyManager contract 
// with different sizes of the input array (which contains the daya holder addresses).

const { ethers } = require("hardhat");
const CONTRACT_NAME = "RoyaltyManager";
const DEFAULT_ADDRESS = "0x0000000000000000000000000000000000000000";
const SIZES = [1, 5, 10, 50, 100, 500, 1000, 1100, 1200, 1300, 1400, 1500,2000]; // Different sizes of the input array to test.

// Generates an array of random Ethereum addresses of the specified size.
function generateAddressArray(size) {
    const array = [];
    for (let i = 0; i < size; i++) {
        array.push(ethers.Wallet.createRandom().address);
    }
    return array;
}

describe("RoyaltyManager Deployment Gas", function () {
    let royaltyManager;
    
    before(async () => {
        royaltyManager = await ethers.getContractFactory(CONTRACT_NAME);
    });

    async function deployAndMeasure(size) {
        const holdersArray = generateAddressArray(size);
        const contract = await royaltyManager.deploy(holdersArray, DEFAULT_ADDRESS);
        const receipt = await contract.deploymentTransaction().wait();
        return receipt.gasUsed;
    }

    it("Deploys OracleVerifier for different numbers of holders", async () => {
        for (const size of SIZES) {
            const gasUsed = await deployAndMeasure(size);
            console.log(`${size}\t${gasUsed.toString()}`);
        }
    });
});