// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IOracleVerifier {
    function setAggregator(address _aggregatorAddress) external;
    function transmit(
        bytes32 configDigest,
        uint64 seqNr,
        bytes calldata report,
        bytes32[] calldata rs, 
        bytes32[] calldata ss, 
        bytes32 rawVs,
        address transmitter
    ) external;
    function getResult(uint256 _jobId) external view returns (int128[] memory, address, uint256);
    function isCompleted(uint256 _jobId) external view returns (bool);
}
