package main

import (
	"encoding/json"
	"math/big"
	"testing"
)

func TestAttributeRequestOmitsThreshold(t *testing.T) {
	payload, err := json.Marshal(AttributeRequest{
		JobId:        "7",
		Text:         "hello",
		FilterPolicy: filterTopValuesName,
	})
	if err != nil {
		t.Fatal(err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"job_id", "text", "filter_policy"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing key %s in payload %s", key, string(payload))
		}
	}
	if _, ok := decoded["threshold"]; ok {
		t.Fatalf("threshold should not be present in payload %s", string(payload))
	}
}

func TestSortedListArrayParsingAndPacking(t *testing.T) {
	body := []byte(`{
		"status": "completed",
		"result": {
			"sorted_list": [[2, "20"], [1, "10"]]
		}
	}`)

	var resp AttributeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}

	packed, err := packSortedList(resp.Result.SortedList)
	if err != nil {
		t.Fatal(err)
	}
	points, err := unpackHolderScores(packed)
	if err != nil {
		t.Fatal(err)
	}

	assertPoint(t, points[0], 1, 10)
	assertPoint(t, points[1], 2, 20)
}

func TestSortedListObjectParsing(t *testing.T) {
	body := []byte(`{
		"status": "completed",
		"result": {
			"sorted_list": [
				{"holder_id": "3", "score": "30"},
				{"holder_id": 1, "score": 10}
			]
		}
	}`)

	var resp AttributeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	packed, err := packSortedList(resp.Result.SortedList)
	if err != nil {
		t.Fatal(err)
	}
	points, err := unpackHolderScores(packed)
	if err != nil {
		t.Fatal(err)
	}

	assertPoint(t, points[0], 1, 10)
	assertPoint(t, points[1], 3, 30)
}

func TestOldHolderIDsScoresShapeIsRejected(t *testing.T) {
	body := []byte(`{
		"status": "completed",
		"result": {
			"holder_ids": [1],
			"scores": ["10"]
		}
	}`)

	var resp AttributeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	if _, err := packSortedList(resp.Result.SortedList); err == nil {
		t.Fatal("expected old holder_ids/scores response shape to fail")
	}
}

func TestPackUnpackHolderScore(t *testing.T) {
	packed, err := packHolderScore(HolderScore{HolderID: 42, Score: big.NewInt(123)})
	if err != nil {
		t.Fatal(err)
	}
	point, err := unpackHolderScore(packed)
	if err != nil {
		t.Fatal(err)
	}
	assertPoint(t, point, 42, 123)
}

func TestInvalidSortedListValuesRejected(t *testing.T) {
	cases := []string{
		`{"sorted_list": [[4294967296, "1"]]}`,
		`{"sorted_list": [[1, "-1"]]}`,
		`{"sorted_list": [[1, "79228162514264337593543950336"]]}`,
	}

	for _, body := range cases {
		var result AttributeResult
		if err := json.Unmarshal([]byte(body), &result); err == nil {
			t.Fatalf("expected invalid sorted_list to fail: %s", body)
		}
	}
}

func TestMedianMissingHolderDefaultsToZero(t *testing.T) {
	candidates := [][]*big.Int{
		mustPackPoints(t, []HolderScore{
			{HolderID: 1, Score: big.NewInt(10)},
			{HolderID: 2, Score: big.NewInt(30)},
		}),
		mustPackPoints(t, []HolderScore{
			{HolderID: 1, Score: big.NewInt(12)},
		}),
		mustPackPoints(t, []HolderScore{
			{HolderID: 1, Score: big.NewInt(14)},
			{HolderID: 2, Score: big.NewInt(6)},
		}),
	}

	points, err := medianHolderScores(candidates)
	if err != nil {
		t.Fatal(err)
	}

	assertPoint(t, points[0], 1, 12)
	assertPoint(t, points[1], 2, 6)
}

func TestTopNSelectionSortsByScoreThenReturnsHolderIDOrder(t *testing.T) {
	points := []HolderScore{
		{HolderID: 3, Score: big.NewInt(50)},
		{HolderID: 1, Score: big.NewInt(12)},
		{HolderID: 2, Score: big.NewInt(6)},
	}

	selected, err := selectTopHolderScores(points, filterTopValues, big.NewInt(2))
	if err != nil {
		t.Fatal(err)
	}

	assertPoint(t, selected[0], 1, 12)
	assertPoint(t, selected[1], 3, 50)
}

func TestTopHoldersMatchesHolderRankedOutput(t *testing.T) {
	points := []HolderScore{
		{HolderID: 3, Score: big.NewInt(50)},
		{HolderID: 1, Score: big.NewInt(12)},
		{HolderID: 2, Score: big.NewInt(6)},
	}

	selected, err := selectTopHolderScores(points, filterTopHolders, big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}

	assertPoint(t, selected[0], 3, 50)
}

func mustPackPoints(t *testing.T, points []HolderScore) []*big.Int {
	t.Helper()
	packed, err := packHolderScores(points)
	if err != nil {
		t.Fatal(err)
	}
	return packed
}

func assertPoint(t *testing.T, point HolderScore, holderID uint32, score int64) {
	t.Helper()
	if point.HolderID != holderID {
		t.Fatalf("holder id: got %d want %d", point.HolderID, holderID)
	}
	if point.Score.Cmp(big.NewInt(score)) != 0 {
		t.Fatalf("score for holder %d: got %s want %d", holderID, point.Score.String(), score)
	}
}
