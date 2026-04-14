// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./IOracleQueue.sol";
import "./IOracleVerifier.sol";

contract Aggregator {
    address public owner;
    uint256 public queryFee;
    uint256 public oracleReward;
    IOracleVerifier public verifier;
    IOracleQueue public queue;

    modifier onlyOwner() {
        require(msg.sender == owner, "Only Owner allowed");
        _;
    }

    modifier onlyVerifier() {
        require(msg.sender == address(verifier), "Only Verifier allowed");
        _;
    }

    // Contract constructor.
    constructor(uint256 _queryFee, uint256 _oracleReward, address _verifierAddress, address _queueAddress) {
        require(_oracleReward <= _queryFee, "The reward cannot exceed the fee");
        owner = msg.sender;
        queryFee = _queryFee;
        oracleReward = _oracleReward;
        verifier = IOracleVerifier(_verifierAddress);
        queue = IOracleQueue(_queueAddress);
    }

    // Forwards an attribution request to the OracleQueue contract.
    // Requires the end user to pay a fee.
    function requestAttribution(string calldata _ipfsCid) external payable {
        // Base check on the exact payment
        require(msg.value == queryFee, "Amount error: must pay the right queryFee"); 
        // Forward request to OracleQueue
        queue.requestAttribution(_ipfsCid, msg.sender, msg.value);
    }

    // Forwards a request approval to the OracleQueue contract.
    // This function can only be called by the model creator.
    function approveJob(uint256 _requestId) external onlyOwner {
        queue.approveJob(_requestId);
    }

    // Forwards the transmission request to the OracleVerifier contract.
    // This function is used to write the DON result on-chain.
    function transmit(
        bytes32 configDigest,
        uint64 seqNr,
        bytes calldata report,
        bytes32[] calldata rs, 
        bytes32[] calldata ss, 
        bytes32 rawVs
    ) external {
        // Forward to verifier
        verifier.transmit(
            configDigest,
            seqNr,
            report,
            rs,
            ss,
            rawVs
        );
    }

    // This function is called automatically by the OracleVerifier contract
    // at the end of the "transmit" function to refund the Oracle that executed the job
    function rewardOracle(address payable _oracle) external onlyVerifier {
        // If there is a fee, reimburse the Oracle for the spent gas
        if (oracleReward > 0) {
            (bool success, ) = _oracle.call{value: oracleReward}("");
            require(success, "Refund to the oracle failed");
        }
    }
}