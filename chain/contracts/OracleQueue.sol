// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract OracleQueue {
    // The model creator is the owner of the contract
    address public modelCreator;

    // Separated counters for the two mappings
    uint256 public requestCounter;
    uint256 public oracleJobCounter;

    // Structure for the Cosutomer queue
    struct CustomerRequest {
        string ipfsCid;
        address requester;
        uint256 payment;
        bool isProcessed; // To know if the model creator has already process it
    }

    struct OracleJob {
        uint256 originalRequestId;
        string ipfsCid;
    }

    // Mapping to keep track of the jobs
    mapping(uint256 => CustomerRequest) public customerQueue;
    mapping(uint256 => OracleJob) public oracleQueue;

    // Event 1: Emitted when the Customer pays and asks for a model computation
    // Listened off-chain by the modelCreator
    event LogNewCustomerRequest(
        uint256 indexed requestId, 
        string ipfsCid, 
        address requester, 
        uint256 payment
    );

    // Event 2: Emitted when the Model Creator approves (listned by the oracles)
    event LogNewJobForOracles(
        uint256 indexed jobId, 
        string ipfsCid
    );

    modifier onlyOwner() {
        require(msg.sender == modelCreator, "Only Model Creator can do this");
        _;
    }

    constructor() {
        modelCreator = msg.sender; // The one who deploys is the model creator
    }

    // =========================================================
    // PHASE 1: CUSTOMER
    // =========================================================    
    function requestAttribution(string calldata _ipfsCid) external payable {
        require(msg.value > 0, "Payment required"); // Base check on the payment

        uint256 currentReqId = requestCounter;

        // Save the new request in the queue
        customerQueue[currentReqId] = CustomerRequest({
            ipfsCid: _ipfsCid,
            requester: msg.sender,
            payment: msg.value,
            isProcessed: false
        });

        emit LogNewCustomerRequest(currentReqId, _ipfsCid, msg.sender, msg.value);
        requestCounter++;
    }

    // =========================================================
    // FASE 2: MODEL CREATOR
    // =========================================================
    // The Model Creator calls this function after validating the request
    function approveJob(uint256 _requestId) external onlyOwner {
        // Reads from customer queue
        CustomerRequest storage req = customerQueue[_requestId];
        require(!req.isProcessed, "Request already approved");

        // Update the state
        req.isProcessed = true;

        // Prepare new id for the Oraclequeue
        uint256 currentOracleJobId = oracleJobCounter;
        // Insert in the queue the new ID for the oracleQueue
        oracleQueue[currentOracleJobId] = OracleJob({
            originalRequestId: _requestId,
            ipfsCid: req.ipfsCid
        });

        // Emit the event that wakes up the Oracles network
        emit LogNewJobForOracles(currentOracleJobId, req.ipfsCid);

        oracleJobCounter++;
    }
    
}
