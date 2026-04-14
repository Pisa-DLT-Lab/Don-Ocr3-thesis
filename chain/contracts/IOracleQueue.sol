// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IOracleQueue {
    function setAggregator(address _aggregatorAddress) external;
    function requestAttribution(string calldata _ipfsCid, address sender, uint256 value) external;
    function approveJob(uint256 _requestId) external;
}

