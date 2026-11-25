package core

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDirectionCriteria(t *testing.T) {
	b, err := os.ReadFile("../testdata/direction.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Criteria      []string
		Forward       bool
		WithPositions bool `json:"with_positions"`
	}
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		f, p := matchingPolicy(c.Criteria)
		if f != c.Forward || p != c.WithPositions {
			t.Fatalf("%v: %v/%v", c.Criteria, f, p)
		}
	}
}

func TestPathBackwardRegression(t *testing.T) {
	texts := []string{"case-test_qux", "test-cherry.index", "test_cherry-world", "test_finder-match", "grape-peach test", "score-lemon-filter"}
	out, err := Execute(Request{Operation: "filter", Candidates: texts, Query: "st", Options: Options{Scheme: "path"}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Results[3].Index != 0 || out.Results[3].Start != 7 || !equalInts(out.Results[3].Positions, []int{7, 8}) {
		t.Fatalf("backward path regression: %+v", out.Results)
	}
}
