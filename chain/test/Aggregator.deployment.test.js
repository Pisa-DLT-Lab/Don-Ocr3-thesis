// This test measures the gas used for deploying the Aggregator contract.

const { ethers } = require("hardhat");
const CONTRACT_NAME = "Aggregator";
const REQUEST_FEE = ethers.parseEther("0.01"); // Fee paid by the customer
const ORACLE_REWARD = ethers.parseEther("0.003"); // Reward to refund the oracle
const MODEL_CREATOR_REWARD = ethers.parseEther("0.002"); // Reward for the model creator
const VERIFIER_ADDRESS = "0x0000000000000000000000000000000000000010";
const QUEUE_ADDRESS = "0x0000000000000000000000000000000000000020";
const ROYALTY_MANAGER_ADDRESS = "0x0000000000000000000000000000000000000030";

describe("Aggregator Deployment Gas", function () {
    it("Deploy Aggregator", async function () {
        const Aggregator = await ethers.getContractFactory(CONTRACT_NAME);
        const contract = await Aggregator.deploy(
            REQUEST_FEE, 
            ORACLE_REWARD, 
            MODEL_CREATOR_REWARD, 
            VERIFIER_ADDRESS, 
            QUEUE_ADDRESS, 
            ROYALTY_MANAGER_ADDRESS
        );
        await contract.waitForDeployment();
    });
});