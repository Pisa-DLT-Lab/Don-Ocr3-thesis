// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract OracleQueue {
    // The model creator is the owner of the contract
    address public owner;
    address public aggregator;

    modifier onlyOwner() {
        require(msg.sender == owner, "Only Owner allowed");
        _;
    }

    modifier onlyAggregator() {
        require(msg.sender == aggregator, "Only Aggregator allowed");
        _;
    }

    function setAggregator(address _aggregatorAddress) external onlyOwner {
        require(_aggregatorAddress != address(0), "Invalid address");
        aggregator = _aggregatorAddress;
    }

    constructor() {
        owner = msg.sender;
    }

    // Separated counters for the two mappings
    uint256 public requestCounter;
    uint256 public oracleJobCounter;

    // Structure for the Customer queue
    struct CustomerRequest {
        string ipfsCid;
        address requester;
        uint256 payment;
        bool isProcessed; // To track if the model creator has already processed it
    }

    struct OracleJob {
        uint256 originalRequestId;
        string ipfsCid;
    }

    // Mapping to keep track of the jobs
    mapping(uint256 => CustomerRequest) public customerQueue;
    mapping(uint256 => OracleJob) public oracleQueue;

    // Event 1: Emitted when the Customer pays and asks for a model computation
    // Listened off-chain by the model creator
    event LogNewCustomerRequest(
        uint256 indexed requestId, 
        string ipfsCid, 
        address requester, 
        uint256 payment
    );

    // Event 2: Emitted when the Model Creator approves the job
    // Listened off-chain by the Oracle Network
    event LogNewJobForOracles(
        uint256 indexed jobId, 
        string ipfsCid
    );

    // =========================================================
    // PHASE 1: CUSTOMER
    // =========================================================    
    function requestAttribution(string calldata _ipfsCid, address sender, uint256 value) external onlyAggregator {
        uint256 currentReqId = requestCounter;

        // Save the new request in the queue
        customerQueue[currentReqId] = CustomerRequest({
            ipfsCid: _ipfsCid,
            requester: sender,
            payment: value,
            isProcessed: false
        });

        emit LogNewCustomerRequest(currentReqId, _ipfsCid, sender, value);
        requestCounter++;
    }

    // =========================================================
    // PHASE 2: MODEL CREATOR
    // =========================================================
    // The Model Creator calls this function after validating the request
    function approveJob(uint256 _requestId) external onlyAggregator {
        // Fetch from the customer queue
        CustomerRequest storage req = customerQueue[_requestId];
        require(!req.isProcessed, "Request already approved");

        // Update the state
        req.isProcessed = true;

        // Prepare new ID for the Oracle queue
        uint256 currentOracleJobId = oracleJobCounter;
        // Insert the approved job into the oracleQueue
        oracleQueue[currentOracleJobId] = OracleJob({
            originalRequestId: _requestId,
            ipfsCid: req.ipfsCid
        });

        // Emit the event that wakes up the Oracles network
        emit LogNewJobForOracles(currentOracleJobId, req.ipfsCid);

        oracleJobCounter++;
    }
}
