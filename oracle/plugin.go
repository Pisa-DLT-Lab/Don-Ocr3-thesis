package main

import (
	"context"
	//"encoding/json"
	"fmt"
	//"math"
	"math/big"
	"sort"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	"github.com/smartcontractkit/libocr/quorumhelper"
)

// =============================================================================
// REPORTING PLUGIN (OCR3 Application Logic)
// =============================================================================
type attributionPluginFactory struct {
	oracleID 	int
	numOracles 	int
}

// NewReportingPlugin is called by libocr when during node bootstrap,
// and it's configured with config, URL of the queue and http client
func (f attributionPluginFactory) NewReportingPlugin(_ context.Context, cfg ocr3types.ReportingPluginConfig) (ocr3types.ReportingPlugin[struct{}], ocr3types.ReportingPluginInfo, error) {
	return &attributionPlugin{
		cfg:      cfg,
		oracleIndex: f.oracleID,
		numOracles: f.numOracles,
	}, ocr3types.ReportingPluginInfo{
		Name: "attribution-matrix-median",
		Limits: ocr3types.ReportingPluginLimits{
			MaxQueryLength:       10 * 1024,
			MaxObservationLength: 256 * 1024, // 256KB limit
			MaxOutcomeLength:     256 * 1024,
			MaxReportLength:      256 * 1024,
			MaxReportCount:       1,
		},
	}, nil
}

type attributionPlugin struct {
	cfg      ocr3types.ReportingPluginConfig
	oracleIndex int
	numOracles int
}


// =============================================================================
// PHASE 1: QUERY (Leader proposes work)
// =============================================================================

// Query is called ONLY by the elected round leader.
// It scans the local thread-safe cache to find the oldest unprocessed job.
// Implementing a FIFO (First-In, First-Out) queue ensures no job is starved
// during periods of high traffic or concurrent requests.
func (p *attributionPlugin) Query(ctx context.Context, _ ocr3types.OutcomeContext) (ocrtypes.Query, error) {
	JobCache.RLock()
	defer JobCache.RUnlock()

	var targetJob *JobData
	var oldestID uint64
	found := false

	// Scan the cache to find the oldest unprocessed job
	for id, job := range JobCache.jobs {
		// Ignore if processed
		if !job.Processed {
			// If it's the first we find
			if !found || id < oldestID {
				oldestID = id
				targetJob = job
				found = true
			}
		}
	}

	// If the queue is empty or all jobs are already processed, skip the round.
	if !found {
		return nil, nil
	}

	fmt.Printf("QUERY (Leader): Proposing Job #%s (CID: %s)\n", targetJob.JobID, targetJob.CID)
	
	//Pack the JobID and CID into an ABI-encoded query for the follower nodes
	return QueryArgs.Pack(targetJob.JobID, targetJob.CID)
}

// =============================================================================
// PHASE 2: OBSERVATION (Nodes fetch data)
// =============================================================================

// Observation is executed by every node (including the leader) upon receiving the Query.
// This function acts as a bridge to our asynchronous worker. Instead of performing 
// blocking HTTP calls (which would trigger libocr timeouts), it simply checks
// the cache and returns empty bytes if the background task is still running.
func (p *attributionPlugin) Observation(ctx context.Context, _ ocr3types.OutcomeContext, query ocrtypes.Query) (ocrtypes.Observation, error) {
	// If the leader proposed an empty query, return empty observation
	if len(query) == 0 { return ocrtypes.Observation("{}"), nil }
	
	qVals, err := QueryArgs.Unpack(query)
	if err != nil { return ocrtypes.Observation("{}"), nil }
	
	reqID := qVals[0].(*big.Int)
	jobId64 := reqID.Uint64()

	// Thread-safe read from the shared cache
	JobCache.RLock()
	job, exists := JobCache.jobs[jobId64]
	JobCache.RUnlock()

	// CASE 1: The node hasn't received the WebSocket event yet
	if !exists { 
		return ocrtypes.Observation("{}"), nil 
	}

	// CASE 2: Asynchrounous computation is still running (can take approx. 10 mins)
	// Returning empty bytes avoids libocr MaxDurationObservation timeout
	if job.State == StatePending {
		// Avoid to print, because it will be full of logs
		return ocrtypes.Observation("{}"), nil
	}

	// CASE 3: The background worker encountered an error
	if job.State == StateFailed {
		fmt.Printf("OBSERVATION Oracle=%d: Job %d Failed: %v\n", p.cfg.OracleID, jobId64, job.Err)
		return ocrtypes.Observation("{}"), nil
	}

	// CASE 4: The computation is completed, pack the result.
	fmt.Printf("OBSERVATION Oracle=%d: Generated Vector len=%d. First: %s\n", p.cfg.OracleID, len(job.Result), job.Result[0].String())    

	// PACK ABI of the attribution method result
	packedObs, err := ObservationArgs.Pack(job.Result)
	if err != nil { return nil, err }
	
	return ocrtypes.Observation(packedObs), nil
}

// ValidateObservation is called on every received observation to reject malformed
// data early.
// Returning nil here means "accept everything".
func (p *attributionPlugin) ValidateObservation(context.Context, ocr3types.OutcomeContext, ocrtypes.Query, ocrtypes.AttributedObservation) error { return nil }


// =============================================================================
// PHASE 3: CONSENSUS & OUTCOME (Leader aggregates data)
// =============================================================================

// ObservationQuorum decides when the leader has enough valid observations to
// propose an outcome. The default OCR2-style rule is "2f+1 observations".
//
// With n=4, f=1 this means 3 valid observations are required to proceed.
func (p *attributionPlugin) ObservationQuorum(_ context.Context, _ ocr3types.OutcomeContext, _ ocrtypes.Query, aos []ocrtypes.AttributedObservation) (bool, error) {
	return quorumhelper.ObservationCountReachesObservationQuorum(quorumhelper.QuorumTwoFPlusOne, p.cfg.N, p.cfg.F, aos), nil
}

// Outcome deterministically aggregates the valid observations into a single agreed outcome.
// It acts as the final gatekeeper, discarding empty observations (from nodes still computing)
// and enforcing the BFT quorum requirement.
func (p *attributionPlugin) Outcome(ctx context.Context, _ ocr3types.OutcomeContext, query ocrtypes.Query, attrObservation []ocrtypes.AttributedObservation) (ocr3types.Outcome, error) {
	if len(query) == 0 { 
		return ocr3types.Outcome([]byte{}), nil 
	}

	// Important if we have two identical vectors (very difficult)
	sort.Slice(attrObservation, func(i, j int) bool {
		return attrObservation[i].Observer < attrObservation[j].Observer
	})
	
	// Unpack Query to get the ID
	qVals, err := QueryArgs.Unpack(query)
	if err != nil { return ocr3types.Outcome([]byte{}), nil }
	reqID := qVals[0].(*big.Int)

	// Decode and store all the received vectors
	// by casting as ObservationArgs the elements of aos (array of observations)
	var candidates [][]*big.Int

	for _, ao := range attrObservation {
        // Unpack Observation ABI
		vals, err := ObservationArgs.Unpack(ao.Observation)
		if err != nil { continue }
        
        // Extract the vector (index 0)
		vector := vals[0].([]*big.Int)
		if len(vector) > 0 {
			candidates = append(candidates, vector)
		}
	}

	// Quorum control (BFT Security)
	// If F=1, we should have at least 3 valid vectors. If we have less abort
	//minSupport := 2*p.cfg.F + 1
	if len(candidates) < (2*p.cfg.F + 1) {
		fmt.Println("WARN: Not enough observations")
		return ocr3types.Outcome([]byte{}), nil
	}

	// ==================================================================
	// COMPUTE MEDIAN VECTOR
	// ==================================================================

	// 1. Take the length of the first vector as reference
	vectorLen := len(candidates[0])
	winner := make([]*big.Int, vectorLen)

	// 2. For every position in the vector
	for i := 0; i < vectorLen; i++ {
		var valuesAtPosition []*big.Int

		// Take all the values for that position
		for _, vec := range candidates {
			// Security check: if an oracle send a vector less long, ignore it for this index
			if len(vec) > i {
				valuesAtPosition = append(valuesAtPosition, vec[i])
			}
		}
		// 3. Order the values from the smallest to the biggest (useful for the median)
		sort.Slice(valuesAtPosition, func(a, b int) bool {
			return valuesAtPosition[a].Cmp(valuesAtPosition[b]) < 0
		})

		// 4. Take the central value
		medianIndex := len(valuesAtPosition)/2

		// Assign this value to the winner
		if len(valuesAtPosition) > 0 {
			winner[i] = valuesAtPosition[medianIndex]
		} else {
			// Fallback if the list is empty (must not happen)
			winner[i] = big.NewInt(0)
		}
	}

	fmt.Printf("OUTCOME (Leader): Selected Vector len %d\n", len(winner))

    // PACK ABI OUTCOME
	packedOutcome, err := OutcomeArgs.Pack(reqID, winner)
	if err != nil { return nil, err }
	return ocr3types.Outcome(packedOutcome), nil
}

// =============================================================================
// PHASE 4: REPORT GENERATION
// =============================================================================

// Reports converts the agreed Outcome into a report to be signed by off-chain keys.
func (p *attributionPlugin) Reports(_ context.Context, _ uint64, outcome ocr3types.Outcome) ([]ocr3types.ReportPlus[struct{}], error) {
	return []ocr3types.ReportPlus[struct{}]{{ReportWithInfo: ocr3types.ReportWithInfo[struct{}]{Report: ocrtypes.Report(outcome), Info: struct{}{}}}}, nil
}

// ShouldAcceptAttestedReport is called after a report has collected enough
// onchain signatures to be considered "attested" (typically f+1 signers).
//
// Returning true means "this report is acceptable for potential transmission".
func (p *attributionPlugin) ShouldAcceptAttestedReport(context.Context, uint64, ocr3types.ReportWithInfo[struct{}]) (bool, error) { return true, nil }

// ShouldTransmitAcceptedReport is called right before transmission.
// Returning true means "actually transmit now (subject to the schedule)".
func (p *attributionPlugin) ShouldTransmitAcceptedReport(ctx context.Context, seqNr uint64, r ocr3types.ReportWithInfo[struct{}]) (bool, error) {
	if len(r.Report) < 32 { return false, nil } // Check base

    // UNPACK REPORT
	values, err := OutcomeArgs.Unpack(r.Report)
	if err != nil { return false, nil }
	
    // Verifica validità ID
    requestID := values[0].(*big.Int)
	if requestID == nil { return false, nil }

    // Logica Round Robin
	//designatedTransmitter := int(seqNr) % p.numOracles
	//if designatedTransmitter == p.oracleIndex {
	//	return true, nil
	//}
	return true, nil
}
func (p *attributionPlugin) Close() error { return nil }
