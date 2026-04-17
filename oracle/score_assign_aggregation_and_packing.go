package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

const (
	filterTopValues      uint8 = 0
	filterTopHolders     uint8 = 1
	filterTopValuesName        = "TOP_VALUES"
	filterTopHoldersName       = "TOP_HOLDERS"
	scoreBitWidth              = 96
)

var (
	uint32Max  = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 32), big.NewInt(1))
	uint96Max  = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 96), big.NewInt(1))
	uint128Max = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
	uint96Mask = new(big.Int).Set(uint96Max)
)

type AttributeResult struct {
	SortedList []HolderScore `json:"sorted_list"`
}

type HolderScore struct {
	HolderID uint32
	Score    *big.Int
}

func (h HolderScore) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string{
		new(big.Int).SetUint64(uint64(h.HolderID)).String(),
		h.Score.String(),
	})
}

func (h *HolderScore) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return errors.New("empty holder-score entry")
	}

	var holderRaw json.RawMessage
	var scoreRaw json.RawMessage

	switch data[0] {
	case '[':
		var pair []json.RawMessage
		if err := json.Unmarshal(data, &pair); err != nil {
			return fmt.Errorf("invalid sorted_list pair: %w", err)
		}
		if len(pair) != 2 {
			return fmt.Errorf("sorted_list pair should have 2 entries, got %d", len(pair))
		}
		holderRaw = pair[0]
		scoreRaw = pair[1]
	case '{':
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(data, &obj); err != nil {
			return fmt.Errorf("invalid sorted_list object: %w", err)
		}
		var ok bool
		holderRaw, ok = firstRaw(obj, "holder_id", "holderId", "owner_id", "ownerId")
		if !ok {
			return errors.New("sorted_list object missing holder_id")
		}
		scoreRaw, ok = firstRaw(obj, "score", "value")
		if !ok {
			return errors.New("sorted_list object missing score")
		}
	default:
		return errors.New("sorted_list entry must be an object or two-item array")
	}

	holder, err := parseBoundedUint(holderRaw, uint32Max, "holder_id")
	if err != nil {
		return err
	}
	score, err := parseBoundedUint(scoreRaw, uint96Max, "score")
	if err != nil {
		return err
	}

	h.HolderID = uint32(holder.Uint64())
	h.Score = score
	return nil
}

func firstRaw(obj map[string]json.RawMessage, keys ...string) (json.RawMessage, bool) {
	for _, key := range keys {
		if raw, ok := obj[key]; ok {
			return raw, true
		}
	}
	return nil, false
}

func parseBoundedUint(raw json.RawMessage, max *big.Int, field string) (*big.Int, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, fmt.Errorf("%s is required", field)
	}

	var value string
	if raw[0] == '"' {
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("invalid %s string: %w", field, err)
		}
	} else {
		value = string(raw)
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, ".eE") {
		return nil, fmt.Errorf("%s must be an unsigned integer", field)
	}

	parsed, ok := new(big.Int).SetString(value, 10)
	if !ok || parsed.Sign() < 0 {
		return nil, fmt.Errorf("%s must be an unsigned integer", field)
	}
	if parsed.Cmp(max) > 0 {
		return nil, fmt.Errorf("%s exceeds maximum %s", field, max.String())
	}
	return parsed, nil
}

func filterPolicyName(filterType uint8) (string, error) {
	switch filterType {
	case filterTopValues:
		return filterTopValuesName, nil
	case filterTopHolders:
		return filterTopHoldersName, nil
	default:
		return "", fmt.Errorf("unknown filter policy enum: %d", filterType)
	}
}

func cloneThreshold(threshold *big.Int) *big.Int {
	if threshold == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Set(threshold)
}

func packSortedList(sortedList []HolderScore) ([]*big.Int, error) {
	if len(sortedList) == 0 {
		return nil, errors.New("sorted_list is required and cannot be empty")
	}
	return packHolderScores(sortedList)
}

func packHolderScores(points []HolderScore) ([]*big.Int, error) {
	points = cloneHolderScores(points)
	sortHolderScoresByID(points)

	packed := make([]*big.Int, 0, len(points))
	seen := make(map[uint32]struct{}, len(points))
	for _, point := range points {
		if _, ok := seen[point.HolderID]; ok {
			return nil, fmt.Errorf("duplicate holder_id %d", point.HolderID)
		}
		seen[point.HolderID] = struct{}{}

		value, err := packHolderScore(point)
		if err != nil {
			return nil, err
		}
		packed = append(packed, value)
	}
	return packed, nil
}

func packHolderScore(point HolderScore) (*big.Int, error) {
	if point.Score == nil {
		return nil, fmt.Errorf("score missing for holder_id %d", point.HolderID)
	}
	if point.Score.Sign() < 0 {
		return nil, fmt.Errorf("score for holder_id %d is negative", point.HolderID)
	}
	if point.Score.Cmp(uint96Max) > 0 {
		return nil, fmt.Errorf("score for holder_id %d exceeds uint96", point.HolderID)
	}

	packed := new(big.Int).SetUint64(uint64(point.HolderID))
	packed.Lsh(packed, scoreBitWidth)
	packed.Or(packed, new(big.Int).Set(point.Score))
	return packed, nil
}

func unpackHolderScore(packed *big.Int) (HolderScore, error) {
	if packed == nil {
		return HolderScore{}, errors.New("packed holder-score is nil")
	}
	if packed.Sign() < 0 {
		return HolderScore{}, errors.New("packed holder-score is negative")
	}
	if packed.Cmp(uint128Max) > 0 {
		return HolderScore{}, errors.New("packed holder-score exceeds uint128")
	}

	holder := new(big.Int).Rsh(new(big.Int).Set(packed), scoreBitWidth)
	if holder.Cmp(uint32Max) > 0 {
		return HolderScore{}, errors.New("packed holder_id exceeds uint32")
	}

	score := new(big.Int).And(new(big.Int).Set(packed), uint96Mask)
	return HolderScore{
		HolderID: uint32(holder.Uint64()),
		Score:    score,
	}, nil
}

func unpackHolderScores(packedValues []*big.Int) ([]HolderScore, error) {
	points := make([]HolderScore, 0, len(packedValues))
	seen := make(map[uint32]struct{}, len(packedValues))
	for _, packed := range packedValues {
		point, err := unpackHolderScore(packed)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[point.HolderID]; ok {
			return nil, fmt.Errorf("duplicate holder_id %d", point.HolderID)
		}
		seen[point.HolderID] = struct{}{}
		points = append(points, point)
	}
	sortHolderScoresByID(points)
	return points, nil
}

func cloneHolderScores(points []HolderScore) []HolderScore {
	out := make([]HolderScore, len(points))
	for i, point := range points {
		var score *big.Int
		if point.Score != nil {
			score = new(big.Int).Set(point.Score)
		}
		out[i] = HolderScore{
			HolderID: point.HolderID,
			Score:    score,
		}
	}
	return out
}

func sortHolderScoresByID(points []HolderScore) {
	sort.Slice(points, func(i, j int) bool {
		return points[i].HolderID < points[j].HolderID
	})
}

func sortHolderScoresByScore(points []HolderScore) {
	sort.Slice(points, func(i, j int) bool {
		cmp := points[i].Score.Cmp(points[j].Score)
		if cmp != 0 {
			return cmp > 0
		}
		return points[i].HolderID < points[j].HolderID
	})
}

func thresholdLimit(threshold *big.Int, size int) int {
	if threshold == nil || threshold.Sign() <= 0 || size <= 0 {
		return 0
	}
	if !threshold.IsUint64() {
		return size
	}
	n := threshold.Uint64()
	if n > uint64(size) {
		return size
	}
	return int(n)
}

func selectTopHolderScores(points []HolderScore, filterType uint8, threshold *big.Int) ([]HolderScore, error) {
	switch filterType {
	case filterTopValues, filterTopHolders:
	default:
		return nil, fmt.Errorf("unknown filter policy enum: %d", filterType)
	}

	ordered := cloneHolderScores(points)
	sortHolderScoresByScore(ordered)

	limit := thresholdLimit(threshold, len(ordered))
	selected := ordered[:limit]
	selected = cloneHolderScores(selected)
	sortHolderScoresByID(selected)
	return selected, nil
}

func medianHolderScores(candidateVectors [][]*big.Int) ([]HolderScore, error) {
	candidates := make([]map[uint32]*big.Int, 0, len(candidateVectors))
	holderSet := make(map[uint32]struct{})

	for _, vector := range candidateVectors {
		points, err := unpackHolderScores(vector)
		if err != nil {
			return nil, err
		}
		candidate := make(map[uint32]*big.Int, len(points))
		for _, point := range points {
			candidate[point.HolderID] = new(big.Int).Set(point.Score)
			holderSet[point.HolderID] = struct{}{}
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return nil, errors.New("no valid candidate vectors")
	}

	holderIDs := make([]uint32, 0, len(holderSet))
	for holderID := range holderSet {
		holderIDs = append(holderIDs, holderID)
	}
	sort.Slice(holderIDs, func(i, j int) bool {
		return holderIDs[i] < holderIDs[j]
	})

	medianPoints := make([]HolderScore, 0, len(holderIDs))
	for _, holderID := range holderIDs {
		values := make([]*big.Int, 0, len(candidates))
		for _, candidate := range candidates {
			if score, ok := candidate[holderID]; ok {
				values = append(values, new(big.Int).Set(score))
			} else {
				values = append(values, big.NewInt(0))
			}
		}
		sort.Slice(values, func(i, j int) bool {
			return values[i].Cmp(values[j]) < 0
		})
		medianPoints = append(medianPoints, HolderScore{
			HolderID: holderID,
			Score:    new(big.Int).Set(values[len(values)/2]),
		})
	}
	return medianPoints, nil
}

func alterPackedScoresByPercent(packedValues []*big.Int, numerator int64, denominator int64) ([]*big.Int, error) {
	if denominator == 0 {
		return nil, errors.New("denominator cannot be zero")
	}
	points, err := unpackHolderScores(packedValues)
	if err != nil {
		return nil, err
	}
	for i := range points {
		points[i].Score.Mul(points[i].Score, big.NewInt(numerator))
		points[i].Score.Div(points[i].Score, big.NewInt(denominator))
		if points[i].Score.Cmp(uint96Max) > 0 {
			points[i].Score = new(big.Int).Set(uint96Max)
		}
	}
	return packSortedList(points)
}
