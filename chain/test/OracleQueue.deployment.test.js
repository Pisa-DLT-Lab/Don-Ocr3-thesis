// This test measures the gas used for deploying the OracleQueue contract.

const { ethers } = require("hardhat");

describe("OracleQueue Deployment Gas", function () {
  it("Deploy OracleQueue", async function () {
    const OracleQueue = await ethers.getContractFactory("OracleQueue");
    const contract = await OracleQueue.deploy();
    await contract.waitForDeployment();
  });
});