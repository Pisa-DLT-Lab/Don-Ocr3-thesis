// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract OracleVerifier {
    // struct for keeping track of the matrixes
    struct Result {
        int128[] flatMatrix;
        address submitter;
        uint256 timestamp;
        bool saved;
    }

    // mapping for the history of the results
    // Grant access directly to the data without cycle. So that the cost is low
    mapping(uint256 => Result) public results;

    //string public latestOutcome;
    //uint256 public reportCount;
    // Mapping to remember which RequestID are already saved
    //mapping(bytes32 => bool) public processedRequests;

    event JobCompleted(uint256 indexed jobId, address indexed submitter, uint256 vectorLenght, uint256 timestamp);

    // This is the function called by the oracles in GO
    function saveOutcome(uint256 jobId, int128[] calldata _flatMatrix) external {
        require(!results[jobId].saved, "Request already fulfilled");

        // Save on-chain (storage)
        results[jobId] = Result({
            flatMatrix:     _flatMatrix,
            submitter:  msg.sender,
            timestamp:  block.timestamp,
            saved:     true
        });
        emit JobCompleted(jobId, msg.sender,_flatMatrix.length, block.timestamp);
    }

    // Getter to retrieve the matrix (for the smart licenses)
    function getResult(uint256 _jobId) external view returns (int128[] memory, address, uint256) {
        require(results[_jobId].saved, "Result not found");

        Result memory r = results[_jobId];
        return (r.flatMatrix, r.submitter, r.timestamp);
    }

    // Check if the job is already completed (for the transmitter)
    function isCompleted(uint256 _jobId) external view returns (bool) {
        return results[_jobId].saved;
    }

}

