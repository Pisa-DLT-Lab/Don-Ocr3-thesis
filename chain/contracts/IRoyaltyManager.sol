// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IRoyaltyManager {
    function rewardHolders(uint256 _jobId) external payable;
    function withdraw() external;
}
