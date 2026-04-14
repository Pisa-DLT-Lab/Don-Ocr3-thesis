// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IAggregator {
    function requestAttribution(string calldata _ipfsCid) external payable;
    function approveJob(uint256 _requestId) external;
    function transmit(
        bytes32 configDigest,
        uint64 seqNr,
        bytes calldata report,
        bytes32[] calldata rs, 
        bytes32[] calldata ss, 
        bytes32 rawVs
    ) external;
    function rewardOracle(address payable _oracle) external;
}

