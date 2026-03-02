package main

import (
	"OCR3-thesis/contracts"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	//"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	//"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	shell "github.com/ipfs/go-ipfs-api"
)

// =============================================================================
// BLOCKCHAIN LISTENER 
// =============================================================================

// startChainListener initializes the WebSocket connection to the blockchain and
// listens for OracleQueue events. It implements a resilient reconnection loop
// to ensure the oracle never misses an event even if the RPC node temporarily drops.
func startChainListener(ctx context.Context, rpcUrl, queueContractAddr string, ipfs *shell.Shell) {
	// Convert standard HTTP RPC URL to WebSocket for real-time event streaming
	wsUrl := strings.Replace(rpcUrl, "http://", "ws://", 1)
	
	fmt.Printf("Listener: Connecting to %s\n    -> Watching Queue: %s\n", wsUrl, queueContractAddr)
	// Resilient loop: automatic reconnection in case of error
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Listener: Context cancelled, shutting down")
			return	 
		default:
			// Blocking call that listens for events. Returns only on error.
			if err := runListener(ctx, wsUrl, queueContractAddr, ipfs); err != nil {
				log.Printf("Listener crashed: %v. Reconnecting in 5s...", err)
				time.Sleep(5 * time.Second)
			}
		}
	}
}

// runListener handles the actual ABI binding, subscription to the contract logs,
// and the parsing of the LogNewJob event.
func runListener(ctx context.Context, wsUrl, queueContractAddr string, ipfs *shell.Shell) error {
	client, err := ethclient.Dial(wsUrl)
	if err != nil {
		return fmt.Errorf("Failed RPC connection: %w", err)
	}
  	defer client.Close()

	qAddr := common.HexToAddress(queueContractAddr)

	// ABI Binding of the smart contract
	queueContract, err := contracts.NewOracleQueue(qAddr, client)
	if err != nil {
		return fmt.Errorf("Failed to bind OracleQueue: %w", err)
	}

	logs := make(chan gethtypes.Log)

	// Filter logs specifically for our OracleQueue contract
	query := ethereum.FilterQuery{
		Addresses:	[]common.Address{qAddr},
	}
	
	sub, err := client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return fmt.Errorf("Failed to subscribe: %w", err)
	}
	defer sub.Unsubscribe()

	fmt.Println("Listener: Listening for LogNewJob events...")

	for {
		select {
		case err := <-sub.Err():
			// Return the error to allow reconnection
			return fmt.Errorf("Subscription error: %w", err)

		case vLog := <-logs:
			// Automatic parsing of the event
			event, err := queueContract.ParseLogNewJobForOracles(vLog)
			if err != nil {
				// If not LogNewJobForOracles, ignore
				continue
			}
			
			// Handle the job without blocking the listener loop	
            if err := handleNewJob(event, ipfs); err != nil {
				log.Printf("Error handling event: %v", err)
			}

		case <-ctx.Done():
			fmt.Println("Listener: Shutdown signal received")
			return nil
		}
	}
}

// =============================================================================
// JOB HANDLING & CACHE MANAGEMENT
// =============================================================================

// handleNewJob receives the parsed event, updates the shared thread-safe cache,
// and uses a non-blocking goroutine. This allows the listener to return immediately
// to the WebSocket stream
func handleNewJob(event *contracts.OracleQueueLogNewJobForOracles, ipfs *shell.Shell) error {
	// Access directly to the field
	jobId := event.JobId
	ipfsCid := event.IpfsCid
	
	// Convert *big.Int to uint64 for map indexing
	jobId64 := jobId.Uint64()

	fmt.Printf("\n Listener: Detected Job #%s | CID: %s\n", jobId.String(), ipfsCid)

	// =========================================================
	// 1. THREAD-SAFE CACHE UPDATE
	// We lock the RWMutex to prevent race conditions with the 
	// OCR3 plugin which might be reading the cache concurrently.
	// =========================================================
	JobCache.Lock()
	
	// ignore if the node already processed this specific job
	if _, exists := JobCache.jobs[jobId64]; exists {
		JobCache.Unlock()
		return nil
	}

	// Update the latestjobId if this new job has a greater id
	if JobCache.LatestJobID.Cmp(big.NewInt(-1)) == 0 || jobId.Cmp(JobCache.LatestJobID) > 0 {
		JobCache.LatestJobID = jobId
	}

	// Create the job in cache with the state PENDING
	// OCR3 Observation() will return empty bytes while in this state.
	JobCache.jobs[jobId64] = &JobData{
		JobID:     jobId,
		CID:       ipfsCid,
		State:     StatePending,
		Processed: false,
	}
	JobCache.Unlock()

	fmt.Printf("[Listener] Job #%d saved as PENDING. Starting async IPFS & AI task...\n", jobId64)

	// =========================================================
	// 2. ASYNCHRONOUS DELEGATION (GOROUTINE)
	// Spawn a background thread for I/O and CPU bound tasks.
	// =========================================================	
	go processJobAsync(jobId64, ipfsCid, ipfs)

	return nil
}

// =============================================================================
// ASYNCHRONOUS WORKER (IPFS & AI Computation)
// =============================================================================
// processJobAsync is the background worker. It handles heavy operations (IPFS, HTTP calls)
// asynchronously. When finished, it updates the shared cache.

// Configuration URL for the python server
const MODEL_SERVICE_URL = "http://host.docker.internal:50100"

func processJobAsync(jobIdUint uint64, ipfsCid string, ipfs *shell.Shell) {
	// ==========================================
	// PHASE 1: IPFS DOWNLOAD
	// ==========================================
	ipfsReader, err := ipfs.Cat(ipfsCid)
	if err != nil {
		setJobError(jobIdUint, fmt.Errorf("IPFS download error: %w", err))
		return
	}
	defer ipfsReader.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(ipfsReader); err != nil {
		setJobError(jobIdUint, fmt.Errorf("IPFS read error: %w", err))
		return
	}
	inputText := buf.String()
	fmt.Printf("[Async Task] Job #%d - IPFS Data Downloaded: %d bytes\n", jobIdUint, len(inputText))

	// =========================================================
	// PHASE 2: Attribution Method
	// This block represents the interaction with the external 
	// Attribution function on Satoshi virtual machine
	// Since this takes ~10 minutes, running it here prevents 
	// libocr timeout loops.
	// =========================================================
	// Convert numerical ID in string to use it as ID in python
	jobIdStr := fmt.Sprintf("%d", jobIdUint)

	// 1. Send Post request
	fmt.Printf("[Async Task] Job #%d - Simulazione calcolo AI in corso...\n", jobIdUint)
	
	reqData := AttributeRequest{
		JobId: jobIdStr,
		Text: inputText,
	}
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		fmt.Printf("Error: json Marshal error")
	}

	res, err := http.Post(MODEL_SERVICE_URL+"/attribute", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		setJobError(jobIdUint, fmt.Errorf("Python connection failed: %w", err))
		return
	}

	res.Body.Close()

	if res.StatusCode != 200 && res.StatusCode != 202 {
		setJobError(jobIdUint, fmt.Errorf("Python server returned status: %d", res.StatusCode))
        return
	}

	// 2. Polling 
	// Control every 30 seconds
	fmt.Printf("[Async Task] Job #%d - Waiting AI Computation...\n", jobIdUint)

	ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    var finalResultStrings []string

	// Infinite loop till we have a result or an error
    for {
        <-ticker.C // Wait 30s
        
        // http GET for checking the state
        statusUrl := fmt.Sprintf("%s/result/%s", MODEL_SERVICE_URL, jobIdStr)
        resp, err := http.Get(statusUrl)
        if err != nil {
            fmt.Printf("[Async Task] Job #%d - Warning: Check status failed (%v). Retrying...\n", jobIdUint, err)
            continue // Retry if there are pronlems (connection loss)
        }

        var pyResp AttributeResponse
        if err := json.NewDecoder(resp.Body).Decode(&pyResp); err != nil {
            resp.Body.Close()
            fmt.Printf("[Async Task] Job #%d - JSON Decode Error. Retrying...\n", jobIdUint)
            continue
        }
        resp.Body.Close()

        // Control the state returned
        if pyResp.Status == "completed" {
            finalResultStrings = pyResp.Result
            break
        } else if pyResp.Status == "error" {
            setJobError(jobIdUint, fmt.Errorf("Python AI Error: %s", pyResp.Error))
            return
        } else {
            // If it's still processing, log and continue waiting
            fmt.Printf("[Async Task] Job #%d - Status: %s. Waiting...\n", jobIdUint, pyResp.Status)
        }
    }

	// =========================================================
	// PHASE 3: CACHE COMMIT
	// Lock the memory and expose the final vector to the OCR3 round.
	// =========================================================	
	// Convert the received strings from Python in big.Int for solidity
	var finalVector []*big.Int
	for _, strVal := range finalResultStrings {
		val := new(big.Int)
		val, success := val.SetString(strVal, 10)
		if !success {
			setJobError(jobIdUint, fmt.Errorf("Failed to parse BigInt from string: %s", strVal))
            return
		}
		finalVector = append(finalVector, val)
	}

	// Update the cache
	JobCache.Lock()
	if job, exists := JobCache.jobs[jobIdUint]; exists {
		job.State = StateCompleted // Flagging the job as ready for consensus
		job.Result = finalVector
	}
	JobCache.Unlock()

	fmt.Printf("[Async Task] Job #%d - COMPLETED! Generated vector of %d elements.\n", jobIdUint, len(finalVector))
}

