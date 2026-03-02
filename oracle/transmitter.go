package main

import (
	"context"
	"crypto/ecdsa"

	//"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"time"

	
	//"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	//gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc" 
	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	"OCR3-thesis/contracts"
)

// =============================================================================
// TRANSMITTER (Blockchain Interaction)
// =============================================================================
/*
type ethTransmitter struct {
	oracleID		int
	rpcUrl			string
	privKeyHex		string
	contractAddr	common.Address
	jobCache		*JobCache // pointer to the structure
}

// FromAccount tells libocr which transmitter identity we are using.
func (t *ethTransmitter) FromAccount(context.Context) (ocrtypes.Account, error) {
    // 1. Decode private key
    privateKey, err := crypto.HexToECDSA(t.privKeyHex)
    if err != nil {
        return "", err
    }

    // 2. Derive the public key and the address
    publicKey := privateKey.Public()
    publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
    if !ok {
        return "", errors.New("error casting public key to ECDSA")
    }

    address := crypto.PubkeyToAddress(*publicKeyECDSA)

    // 3. Return the real address in format Hex String
    return ocrtypes.Account(address.Hex()), nil
}

// VERSION WITHOUT VIEW

// Transmit is called by libocr once the report is attested and this oracle is
// selected by the transmission schedule.
//
// IMPORTANT: `sigs` is not "all signatures from all nodes"; libocr provides only
// the minimum quorum needed for on-chain verification (typically f+1) to save bytes.
func (t *ethTransmitter) Transmit(ctx context.Context, cd ocrtypes.ConfigDigest, seq uint64, report ocr3types.ReportWithInfo[struct{}], sigs []ocrtypes.AttributedOnchainSignature) error {

	// ---------------------------------------------------------
	// 1. Decode & Prepare Data
	// ---------------------------------------------------------
	var payload outcomePayload
	if err := json.Unmarshal(report.Report, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal report: %v", err)
	}

	// Serialize the matrix ONCE to be used for both logging and chain data
	matrixBytes, err := json.Marshal(payload.Matrix)
	if err != nil {
		return fmt.Errorf("failed to marshal matrix: %w", err)
	}
	matrixJsonString := string(matrixBytes)

	// =========================================================
    // 1.1 Check of the
    // =========================================================
	// Convert RequestID string -> BigInt -> Bytes32 for later
	currentJobId, ok := new(big.Int).SetString(payload.RequestID, 10)
	if !ok {
		return nil  // Invalid ID
	}

	t.jobCache.RLock()	// For reading
	cachedID := t.jobCache.LatestJobID
	isProcessed := t.jobCache.Processed
	t.jobCache.RUnlock()

	// If the ID in the cache matches the one I am sending
	// And the flag "Processed" is true (so the listener saw the event OutcomeReceived)
	// Stop
	if cachedID != nil && currentJobId.Cmp(cachedID) == 0 && isProcessed {
		fmt.Printf("\n Transaction canceled: Job #%s is alreafy processed on-chain. \n", payload.RequestID)
		return nil
	}

	fmt.Printf("\nTransmitting to Blockchain [Oracle %d]... Data len: %d\n", t.oracleID, len(matrixJsonString))

	// ---------------------------------------------------------
	// 2. Connect to Blockchain & Auth
	// ---------------------------------------------------------
	client, err := ethclient.Dial(t.rpcUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to chain: %w", err)
	}
	defer client.Close()

	// Load Private Key
	privateKey, err := crypto.HexToECDSA(t.privKeyHex)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	// Derive Public Key and Sender Address
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("error casting public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// ---------------------------------------------------------
	// 3. Retrieve Network State (Nonce, Gas, ChainID)
	// ---------------------------------------------------------
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to suggest gas price: %w", err)
	}

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get network ID: %w", err)
	}

	// ---------------------------------------------------------
	// 4. Prepare Smart Contract Arguments
	// ---------------------------------------------------------

	//var requestIdBytes [32]byte
	//currentJobId.FillBytes(requestIdBytes[:])

	// Define ABI inline (saveOutcome expects uint256 and string)
	const contractABI = `[{"inputs":[{"internalType":"uint256", "name": "requestId", "type":"uint256"},{"internalType":"string","name":"_matrixJson","type":"string"}],"name":"saveOutcome","outputs":[],"stateMutability":"nonpayable","type":"function"}]`

	parsedABI, err := abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		return fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Pack arguments
	inputData, err := parsedABI.Pack("saveOutcome", currentJobId, matrixJsonString)
	if err != nil {
		return fmt.Errorf("failed to pack ABI data: %w", err)
	}

	// ---------------------------------------------------------
	// 5. Sign & Send Transaction
	// ---------------------------------------------------------

	// Create transaction object
	tx := gethtypes.NewTransaction(nonce, t.contractAddr, big.NewInt(0), 5000000, gasPrice, inputData)

	// Sign transaction
	signedTx, err := gethtypes.SignTx(tx, gethtypes.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Broadcast transaction
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		// Handle "First-Come" logic: if it reverts, likely another oracle was faster
		if strings.Contains(err.Error(), "Request already fulfilled") || strings.Contains(err.Error(), "revert") {
			// Update the cache instantly, avoiding the listener to do this
			t.jobCache.Lock()
			if !t.jobCache.Processed {
				t.jobCache.Processed = true
				fmt.Printf("Job #%s completed (Revert). Block the future attempts. \n", payload.RequestID)
			}
			t.jobCache.Unlock()
			return nil
		}
		return fmt.Errorf("Tx FAILED: %w", err)
	}

	fmt.Printf("Tx SENT. Hash: %s\n\n", signedTx.Hash().Hex())
	return nil
}
*/

type ethTransmitter struct {
	oracleID		int
	rpcUrl			string
	privKeyHex		string
	contractAddr	common.Address
	jobCache		*jobCache // pointer to the shared thread-safe cache
}

// FromAccount retrieves the identity of the oracle node
// LibOcr uses this method to verify which on-chain account corresponds
// to this transmitter during the signing and transmission phase
func (t *ethTransmitter) FromAccount(context.Context) (ocrtypes.Account, error) {
    // 1. Private key decoding
    privateKey, err := crypto.HexToECDSA(t.privKeyHex)
    if err != nil {
        return "", err
    }

    // 2. Public key derivation
	// Extract the public key to compute the Ethereum address
    publicKey := privateKey.Public()
    publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
    if !ok {
        return "", errors.New("error casting public key to ECDSA")
    }
    
    address := crypto.PubkeyToAddress(*publicKeyECDSA)
    
    // 3. Identity return
	// Return the address as a Hex String
    return ocrtypes.Account(address.Hex()), nil
}

// VERSION WITH VIEW V.1
/*
// Transmit is called by libocr once the report is attested and this oracle is
// selected by the transmission schedule.
//
// IMPORTANT: `sigs` is not "all signatures from all nodes"; libocr provides only
// the minimum quorum needed for on-chain verification (typically f+1) to save bytes.
func (t *ethTransmitter) Transmit(ctx context.Context, cd ocrtypes.ConfigDigest, seq uint64, report ocr3types.ReportWithInfo[struct{}], sigs []ocrtypes.AttributedOnchainSignature) error {
    
    // ---------------------------------------------------------
    // 1. Decode & Prepare Data
    // ---------------------------------------------------------
    if len(report.Report) < 32 { return nil }

    values, err := OutcomeArgs.Unpack(report.Report)
    if err != nil {
        return fmt.Errorf("failed to unpack report ABI: %v", err)
    }

    requestID := values[0].(*big.Int)
    vector := values[1].([]*big.Int)

    // Check rapido della cache locale prima di fare qualsiasi cosa
    t.jobCache.RLock()
    isProcessedLocal := t.jobCache.Processed
    t.jobCache.RUnlock()

    if isProcessedLocal {
        return nil
    }

    // ---------------------------------------------------------
    // 2. RANDOM SLEEP (1s - 6s) - L'attesa avviene per prima!
    // ---------------------------------------------------------
    // Genera un'attesa casuale tra 1 e 6 secondi
    sleepTime := time.Duration(1000 + rand.Intn(5001)) * time.Millisecond
    fmt.Printf("[Oracle %d] Sleeping for %v before checking chain...\n", t.oracleID, sleepTime)
    
    select {
    case <-time.After(sleepTime):
        // Continua dopo aver dormito
    case <-ctx.Done():
        return nil // Esce se il contesto muore durante l'attesa
    }

    // =========================================================
    // DA QUI IN POI IL NODO È SVEGLIO
    // =========================================================

    // Ricontrolla la cache locale (magari un altro oracolo ha finito mentre dormiva)
    t.jobCache.RLock()
    isNowProcessed := t.jobCache.Processed
    t.jobCache.RUnlock()
    
    if isNowProcessed {
        fmt.Printf("[Oracle %d] Woke up, but job #%s was marked processed in local cache. Aborting TX.\n", t.oracleID, requestID)
        return nil
    }

    // ---------------------------------------------------------
    // 3. Connect to Blockchain via RPC per controllare la View
    // ---------------------------------------------------------
    rpcClient, err := rpc.Dial(t.rpcUrl)
    if err != nil {
        fmt.Printf("[Oracle %d] RPC dial failed (%v). Proceeding anyway...\n", t.oracleID, err)
    } else {
        defer rpcClient.Close()

        // Encode job ID as 32 bytes (come nel tuo codice originale)
        idBytes := make([]byte, 32)
        requestID.FillBytes(idBytes)
        data := "0x7a41984b" + hex.EncodeToString(idBytes)

        var result string
        callErr := rpcClient.CallContext(ctx, &result, "eth_call", map[string]string{
            "to":   t.contractAddr.Hex(),
            "data": data,
        }, "latest")

        if callErr == nil {
            isDone := result == "0x0000000000000000000000000000000000000000000000000000000000000001"
            
            // SE IL JOB È GIÀ FATTO, FERMATI QUI!
            if isDone {
                fmt.Printf("[Oracle %d] Job #%s already completed on-chain. Skipping TX.\n", t.oracleID, requestID)
                
                // Aggiorna la cache in modo che le prossime chiamate vengano droppate subito (Punto 1)
                t.jobCache.Lock()
                t.jobCache.Processed = true
                t.jobCache.Unlock()
                
                return nil // <--- QUESTO IMPEDISCE DI CHIAMARE SAVEOUTCOME
            }
        } else {
            fmt.Printf("[Oracle %d] View check failed (%v). Proceeding anyway...\n", t.oracleID, callErr)
        }
    }

    // ---------------------------------------------------------
    // 4. Se arriviamo qui: il Job non è fatto, dobbiamo inviare
    // ---------------------------------------------------------
    fmt.Printf("[Oracle %d] Transmitting Job #%s... Vector Len: %d\n", t.oracleID, requestID, len(vector))

    client, err := ethclient.Dial(t.rpcUrl)
    if err != nil {
        return fmt.Errorf("failed to connect ethclient: %w", err)
    }
    defer client.Close()

    verifier, err := contracts.NewOracleVerifier(t.contractAddr, client)
    if err != nil {
        return fmt.Errorf("binding error: %w", err)
    }

    privateKey, err := crypto.HexToECDSA(t.privKeyHex)
    if err != nil {
        return fmt.Errorf("invalid private key: %w", err)
    }

    chainID, err := client.NetworkID(ctx)
    if err != nil {
        return fmt.Errorf("failed to get chainID: %w", err)
    }

    auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
    if err != nil {
        return fmt.Errorf("failed to create transactor: %w", err)
    }

    auth.GasPrice, _ = client.SuggestGasPrice(ctx)
    auth.GasFeeCap = nil
    auth.GasTipCap = nil
    auth.GasLimit = 30000000

    // ---------------------------------------------------------
    // 5. Send Transaction (SaveOutcome)
    // ---------------------------------------------------------
    tx, err := verifier.SaveOutcome(auth, requestID, vector)
    if err != nil {
        errMsg := err.Error()
        // Cattura eventuali reversioni millimetriche
        if strings.Contains(errMsg, "Request already fulfilled") || strings.Contains(errMsg, "revert") || strings.Contains(errMsg, "nonce too low") {
            fmt.Printf("[Oracle %d] Race Condition: Another node won at the last millisecond! (TX Reverted)\n", t.oracleID)
            
            t.jobCache.Lock()
            t.jobCache.Processed = true
            t.jobCache.Unlock()
            
            return nil
        }
        return fmt.Errorf("Tx FAILED: %w", err)
    }

    fmt.Printf("[Oracle %d] Tx Sent! Hash: %s\n", t.oracleID, tx.Hash().Hex())

    t.jobCache.Lock()
    t.jobCache.Processed = true
    t.jobCache.Unlock()
    
    return nil
}

*//*

// VERSION WITH VIEW V.2
func (t *ethTransmitter) Transmit(ctx context.Context, cd ocrtypes.ConfigDigest, seq uint64, report ocr3types.ReportWithInfo[struct{}], sigs []ocrtypes.AttributedOnchainSignature) error {

	// =========================================================
	// 1. DECODE REPORT
	// Unpack the ABI-encoded report to extract the job ID and
	// the result vector produced by the reporting plugin.
	// =========================================================
	if len(report.Report) < 32 {
		return nil
	}

	values, err := OutcomeArgs.Unpack(report.Report)
	if err != nil {
		return fmt.Errorf("failed to unpack report ABI: %v", err)
	}

	requestID := values[0].(*big.Int)
	vector := values[1].([]*big.Int)

	// ======================================================================
	// 2. LOCAL CACHE CHECK
	// Check the in-memory cache before making any call to the smart contract
	// If this oracle already processed this job, skip early.
	// ======================================================================
	t.jobCache.RLock()
	isProcessedLocal := t.jobCache.Processed
	t.jobCache.RUnlock()

	if isProcessedLocal {
		return nil
	}

	// =========================================================================
	// 3. RANDOM BACKOFF
	// Each oracle sleeps for a random duration (1s-6s) to reduce
	// the chance of multiple oracles submitting the same tx at exact same time
	// and to simulate the latency of the transmission into the mainnet
	// =========================================================================
	sleepTime := time.Duration(1000+rand.Intn(5001)) * time.Millisecond
	fmt.Printf("[Oracle %d] Sleeping %v before on-chain check...\n", t.oracleID, sleepTime)

	select {
	case <-time.After(sleepTime):
	case <-ctx.Done():
		return nil
	}

	// ==================================================================================
	// 4. ON-CHAIN VIEW CHECK
	//
	// I intentionally avoid ethclient.CallContract here.
	// When using Hardhat as the development node, ethclient misinterprets valid responses 
	// from view functions as reverts, returning "execution reverted" even when the
	// contract executes correctly. 
	//
	// The workaround is to use rpc.CallContext directly, which sends a raw eth_call 
	// JSON-RPC request without any additional interpretation layer
	// ==================================================================================
	rpcClient, err := rpc.Dial(t.rpcUrl)
	if err != nil {
		fmt.Printf("[Oracle %d] Raw RPC connection failed: %v. Proceeding to transmit...\n", t.oracleID, err)
	} else {
		defer rpcClient.Close()

		// Build the calldata: function selector + ABI-encoded uint256 parameter.
		// Using abi.JSON ensures the encoding is always spec-compliant.
		const isCompletedABI = `[{"inputs":[{"internalType":"uint256","name":"_requestId","type":"uint256"}],"name":"isCompleted","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}]`
		parsedABI, _ := abi.JSON(strings.NewReader(isCompletedABI))

		inputData, err := parsedABI.Pack("isCompleted", requestID)
		if err == nil {
			callMsg := map[string]string{
				"to":   t.contractAddr.Hex(),
				"data": hexutil.Encode(inputData),
			}

			var resultHex string
			err = rpcClient.CallContext(ctx, &resultHex, "eth_call", callMsg, "latest")

			if err != nil {
				fmt.Printf("[Oracle %d] On-chain view check failed: %v. Proceeding to transmit...\n", t.oracleID, err)
			} else {
				// Decode the raw hex response back into a Go bool.
				resultBytes, err := hexutil.Decode(resultHex)
				if err == nil {
					var isDone bool
					err = parsedABI.UnpackIntoInterface(&isDone, "isCompleted", resultBytes)

					if err == nil && isDone {
						// Job already finalized on-chain, no need to submit again.
						fmt.Printf("[Oracle %d] Job #%s already completed on-chain. Skipping transmission.\n", t.oracleID, requestID)
						t.jobCache.Lock()
						t.jobCache.Processed = true
						t.jobCache.Unlock()
						return nil
					}
				}
			}
		}
	}

	// =========================================================
	// 5. CONNECT & BUILD TRANSACTION
	// ethclient is used here only for sending signed transactions,
	// where it works correctly. The view check above uses the
	// raw RPC client instead (see step 4).
	// =========================================================
	fmt.Printf("[Oracle %d] Transmitting Job #%s... Vector length: %d\n", t.oracleID, requestID, len(vector))

	client, err := ethclient.Dial(t.rpcUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to chain: %w", err)
	}
	defer client.Close()

	verifier, err := contracts.NewOracleVerifier(t.contractAddr, client)
	if err != nil {
		return fmt.Errorf("failed to instantiate contract binding: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(t.privKeyHex)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve chain ID: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return fmt.Errorf("failed to create transactor: %w", err)
	}

	// For Hardhat compatibility.
	auth.GasPrice, _ = client.SuggestGasPrice(ctx)
	auth.GasFeeCap = nil
	auth.GasTipCap = nil
	auth.GasLimit = 30000000

	// =========================================================
	// 7. SEND TRANSACTION
	// Submit the result on-chain. If another oracle already
	// submitted (race condition), the contract will revert with
	// "Request already fulfilled" and we handle it gracefully.
	// =========================================================
	tx, err := verifier.SaveOutcome(auth, requestID, vector)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "Request already fulfilled") || strings.Contains(errMsg, "revert") {
			fmt.Printf("[Oracle %d] Race condition: another oracle submitted first. TX reverted gracefully.\n", t.oracleID)
			t.jobCache.Lock()
			t.jobCache.Processed = true
			t.jobCache.Unlock()
			return nil
		}
		return fmt.Errorf("transaction failed: %w", err)
	}

	fmt.Printf("[Oracle %d] Tx Sent! Hash: %s\n", t.oracleID, tx.Hash().Hex())

	t.jobCache.Lock()
	t.jobCache.Processed = true
	t.jobCache.Unlock()

	return nil
}
*/

func (t *ethTransmitter) Transmit(ctx context.Context, cd ocrtypes.ConfigDigest, seq uint64, report ocr3types.ReportWithInfo[struct{}], sigs []ocrtypes.AttributedOnchainSignature) error {

	// =========================================================
	// 1. DECODE REPORT
	// Unpack the ABI-encoded report to extract the job ID and
	// the result vector produced by the reporting plugin.
	// =========================================================
	if len(report.Report) < 32 {
		return nil
	}

	values, err := OutcomeArgs.Unpack(report.Report)
	if err != nil {
		return fmt.Errorf("failed to unpack report ABI: %v", err)
	}

	requestID := values[0].(*big.Int)
	vector := values[1].([]*big.Int)

	// ======================================================================
	// 2. LOCAL CACHE CHECK
	// Check the in-memory cache before making any call to the smart contract
	// If this oracle already processed this job, skip early.
	// ======================================================================
	jobId64 := requestID.Uint64()

	t.jobCache.RLock()
	job, exists := t.jobCache.jobs[jobId64]
	isProcessedLocal := exists && job.Processed
	t.jobCache.RUnlock()

	if isProcessedLocal {
		return nil
	}

	// =========================================================================
	// 3. RANDOM BACKOFF
	// Each oracle sleeps for a random duration (1s-6s) to reduce
	// the chance of multiple oracles submitting the same tx at exact same time
	// and to simulate the latency of the transmission into the mainnet
	// =========================================================================
	sleepTime := time.Duration(1000+rand.Intn(5001)) * time.Millisecond
	fmt.Printf("[Oracle %d] Sleeping %v before on-chain check...\n", t.oracleID, sleepTime)

	select {
	case <-time.After(sleepTime):
	case <-ctx.Done():
		return nil
	}

	// ==================================================================================
	// 4. ON-CHAIN VIEW CHECK
	//
	// I intentionally avoid ethclient.CallContract here.
	// When using Hardhat as the development node, ethclient misinterprets valid responses 
	// from view functions as reverts, returning "execution reverted" even when the
	// contract executes correctly. 
	//
	// The workaround is to use rpc.CallContext directly, which sends a raw eth_call 
	// JSON-RPC request without any additional interpretation layer
	// ==================================================================================
	rpcClient, err := rpc.Dial(t.rpcUrl)
	if err != nil {
		fmt.Printf("[Oracle %d] Raw RPC connection failed: %v. Proceeding to transmit...\n", t.oracleID, err)
	} else {
		defer rpcClient.Close()

		// Build the calldata: function selector + ABI-encoded uint256 parameter.
		// Using abi.JSON ensures the encoding is always spec-compliant.
		const isCompletedABI = `[{"inputs":[{"internalType":"uint256","name":"_requestId","type":"uint256"}],"name":"isCompleted","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}]`
		parsedABI, _ := abi.JSON(strings.NewReader(isCompletedABI))

		inputData, err := parsedABI.Pack("isCompleted", requestID)
		if err == nil {
			callMsg := map[string]string{
				"to":   t.contractAddr.Hex(),
				"data": hexutil.Encode(inputData),
			}

			var resultHex string
			err = rpcClient.CallContext(ctx, &resultHex, "eth_call", callMsg, "latest")

			if err != nil {
				fmt.Printf("[Oracle %d] On-chain view check failed: %v. Proceeding to transmit...\n", t.oracleID, err)
			} else {
				// Decode the raw hex response back into a Go bool.
				resultBytes, err := hexutil.Decode(resultHex)
				if err == nil {
					var isDone bool
					err = parsedABI.UnpackIntoInterface(&isDone, "isCompleted", resultBytes)

					if err == nil && isDone {
						// Job already finalized on-chain, no need to submit again.
						fmt.Printf("[Oracle %d] Job #%s already completed on-chain. Skipping transmission.\n", t.oracleID, requestID)
						
						MarkJobAsProcessed(jobId64)
						return nil
					}
				}
			}
		}
	}

	// =========================================================
	// 5. CONNECT & BUILD TRANSACTION
	// ethclient is used here only for sending signed transactions,
	// where it works correctly. The view check above uses the
	// raw RPC client instead (see step 4).
	// =========================================================
	fmt.Printf("[Oracle %d] Transmitting Job #%s... Vector length: %d\n", t.oracleID, requestID, len(vector))

	client, err := ethclient.Dial(t.rpcUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to chain: %w", err)
	}
	defer client.Close()

	verifier, err := contracts.NewOracleVerifier(t.contractAddr, client)
	if err != nil {
		return fmt.Errorf("failed to instantiate contract binding: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(t.privKeyHex)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve chain ID: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return fmt.Errorf("failed to create transactor: %w", err)
	}

	// For Hardhat compatibility.
	auth.GasPrice, _ = client.SuggestGasPrice(ctx)
	auth.GasFeeCap = nil
	auth.GasTipCap = nil
	auth.GasLimit = 30000000

	// =========================================================
	// 7. SEND TRANSACTION
	// Submit the result on-chain. If another oracle already
	// submitted (race condition), the contract will revert with
	// "Request already fulfilled" and we handle it gracefully.
	// =========================================================
	tx, err := verifier.SaveOutcome(auth, requestID, vector)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "Request already fulfilled") || strings.Contains(errMsg, "revert") {
			fmt.Printf("[Oracle %d] Race condition: another oracle submitted first. TX reverted gracefully.\n", t.oracleID)
			
			MarkJobAsProcessed(jobId64)
			return nil
		}
		return fmt.Errorf("transaction failed: %w", err)
	}

	fmt.Printf("[Oracle %d] Tx Sent! Hash: %s\n", t.oracleID, tx.Hash().Hex())

	MarkJobAsProcessed(jobId64)

	return nil
}