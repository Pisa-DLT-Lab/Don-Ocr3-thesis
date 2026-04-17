// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./IAggregator.sol";
import "./IOracleQueue.sol";
import "./IOracleVerifier.sol";
import "./IRoyaltyManager.sol";

contract Aggregator is IAggregator {
    address public owner; // Model creator address.
    uint256 public queryFee; // Fee paid by end user for a request.
    uint256 public oracleReward; // Reward for the oracle that executes the job.
    uint256 public modelCreatorReward; // Reward for model creator.
    IOracleVerifier public verifier;
    IOracleQueue public queue;
    IRoyaltyManager public manager;
    FilterType public filterType; // Type of filter to apply on the data scores for reward distribution.
    uint256 public filterThreshold; // Threshold value for filtering (e.g., number of top values/holders).

    modifier onlyOwner() {
        require(msg.sender == owner, "Only Owner allowed");
        _;
    }

    modifier onlyVerifier() {
        require(msg.sender == address(verifier), "Only Verifier allowed");
        _;
    }

    // Contract constructor.
    constructor(
        uint256 _queryFee, 
        uint256 _oracleReward, 
        uint256 _modelCreatorReward, 
        address _verifierAddress, 
        address _queueAddress,
        address _managerAddress
    ) {
        require((0 < _oracleReward + _modelCreatorReward) && (_oracleReward + _modelCreatorReward < _queryFee), 
        "Model creator and oracle rewards should be > 0 and cannot exceed the fee");
        owner = msg.sender;
        queryFee = _queryFee;
        oracleReward = _oracleReward;
        modelCreatorReward = _modelCreatorReward;
        verifier = IOracleVerifier(_verifierAddress);
        queue = IOracleQueue(_queueAddress);
        manager = IRoyaltyManager(_managerAddress);
        filterType = FilterType.TOP_VALUES; // Default filter type.
        filterThreshold = 100; // Default threshold.
    }

    // Forwards an attribution request to the OracleQueue contract.
    // Requires the end user to pay a fee.
    function requestAttribution(string calldata _ipfsCid) external override payable returns (uint256) {
        // Base check on the exact payment
        require(msg.value == queryFee, "Amount error: must pay the right queryFee"); 
        // Forward request to OracleQueue
        uint256 requestId = queue.requestAttribution(_ipfsCid, msg.sender, msg.value);
        emit LogNewCustomerRequest(requestId, _ipfsCid, msg.sender, msg.value);
        return requestId;
    }

    // Forwards a request approval to the OracleQueue contract.
    // This function can only be called by the model creator.
    function approveJob(uint256 _requestId) external override onlyOwner returns (uint256) {
        (uint256 jobId, string memory ipfsCid) = queue.approveJob(_requestId);
        emit LogNewJobForOracles(jobId, ipfsCid);
        return jobId;
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
    ) external override {
        // Forward to OracleVerifier.
        verifier.transmit(
            configDigest,
            seqNr,
            report,
            rs,
            ss,
            rawVs,
            msg.sender
        );
    }

    // This function is called automatically by the OracleVerifier contract
    // at the end of the "transmit" function to refund the Oracle that executed the job
    // and distribute the rewards to the model creator.
    function distributeRewards(address payable _oracle, uint256 _jobId, uint256 _vectorLength) external override onlyVerifier {
        // Check current balance.
        uint256 balance = address(this).balance;
        require(balance >= queryFee, "No funds");
        // First, reimburse the Oracle for the spent gas.
        (bool success, ) = _oracle.call{value: oracleReward}("");
        require(success, "Refund to the oracle failed");
        // Secondly, reward the model creator.
        (success, ) = owner.call{value: modelCreatorReward}("");
        require(success, "Reward for the model creator failed");
        // Finally, distribute the remaining funds to the data holders 
        // through the RoyaltyManager contract.
        uint256 holdersReward = queryFee - (oracleReward + modelCreatorReward); 
        manager.rewardHolders{value: holdersReward}(_jobId);
        emit JobCompleted(_jobId, _oracle, _vectorLength, block.timestamp);
    }

    function getResult(uint256 _jobId) external override view returns (int128[] memory, address, uint256) {
        return verifier.getResult(_jobId);
    }

    function isCompleted(uint256 _jobId) external override view returns (bool) {
        return verifier.isCompleted(_jobId);
    }

    function setFilterPolicy(FilterType _filterType, uint256 _threshold) external override onlyOwner {
        filterType = _filterType;
        filterThreshold = _threshold;
    }

    function getFilterPolicy() external override view returns (FilterType, uint256) {
        return (filterType, filterThreshold);
    }
}
