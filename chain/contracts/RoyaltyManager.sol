// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./IOracleVerifier.sol";

contract RoyaltyManager {
    address[] public holders; // Array of data holders' addresses.
    IOracleVerifier public verifier; // Link to the OracleVerifier contract.
    mapping(address => uint256) public balances; // Stores account balances for each holder.
    event RewardsDistributed(uint256 indexed jobId); // Emitted when rewards are distributed.

    constructor(address[] memory _holders, address _verifierAddress) {
        require(_holders.length > 0, "No holders provided");
        holders = _holders;
        verifier = IOracleVerifier(_verifierAddress);
    }

    // Distributes rewards to data holders based on the job result.
    function rewardHolders(uint256 _jobId) external payable {
        require(msg.value > 0, "No funds available for distribution");
        // Fetch job result to determine holders' shares.
        (int128[] memory data, , ) = verifier.getResult(_jobId);
        // Distribute rewards proportionally based on data scores.
        uint128 u;
        uint32 chunkId;
        uint32 ownerId;
        uint64 score;
        for (uint i = 0; i < data.length; i++) {
            // Decode result data based on the 
            // [chunkId (32 bits) | ownerId (32 bits) | score (64 bits)] encoding scheme.
            u = uint128(data[i]);
            chunkId = uint32(u >> 96); // first 32 bits
            ownerId = uint32((u >> 64) & 0xFFFFFFFF); // next 32 bits
            score = uint64(u); // last 64 bits
            uint256 payment = (msg.value * uint256(score)) / type(uint64).max;
            balances[holders[ownerId]] += payment;
        }
        emit RewardsDistributed(_jobId); // Notify reward distribution.
    }

    // Allows holders to withdraw their accumulated rewards.
    function withdraw() external {
        uint256 amount = balances[msg.sender];
        require(amount > 0, "Nothing to withdraw");
        (bool success, ) = payable(msg.sender).call{value: amount}("");
        require(success, "Transfer failed");
        balances[msg.sender] = 0;
    }

}