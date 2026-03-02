const hre = require("hardhat")

async function main() {
    console.log(" Deploying OracleQueue.sol contract...")
    // 1. Prepare the contract
    const Oraclequeue = await hre.ethers.getContractFactory("OracleQueue")

    // 2. Execute the deployment
    const queue = await Oraclequeue.deploy();

    // 3. Wait to get mined by the blockchain
    await queue.waitForDeployment();

    const address = await queue.getAddress();
    console.log("----------------------------------------------------");
    console.log(" OracleQueue deployed succesfully!");
    console.log(` Address: ${address}`);
    console.log("----------------------------------------------------");
}

main().catch((error) => {
    console.error(error)
    process.exitCode = 1;
});