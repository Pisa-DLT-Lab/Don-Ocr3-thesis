
require("@nomicfoundation/hardhat-ethers");
require("dotenv").config({path: "../.env"})

/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: "0.8.24",
  networks: {
    hardhat: {
      chainId: 31337
    },
    localhost: {
      url: "http://127.0.0.1:8545",
    },
    docker: {
      url: process.env.CHAIN_RPC_URL || "http://chain:8545",    
    }
  },
};