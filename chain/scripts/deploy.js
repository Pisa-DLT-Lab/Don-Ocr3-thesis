const hre = require("hardhat");

async function main() {
  const Verifier = await hre.ethers.getContractFactory("OracleVerifier");
  const verifier = await Verifier.deploy();

  await verifier.waitForDeployment();

  const address = await verifier.getAddress();
    console.log("----------------------------------------------------");
    console.log(" OracleVerifier deployed succesfully!");
    console.log(` Address: ${address}`);
    console.log("----------------------------------------------------");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});