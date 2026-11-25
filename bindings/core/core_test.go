package core

import (
	"encoding/json"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/pinyin"
	"github.com/junegunn/fzf/src/util"
)

func TestGolden(t *testing.T) {
	b, err := os.ReadFile("../testdata/golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Vectors []struct {
			Name     string
			Request  Request
			Expected Response
		}
	}
	if err := json.Unmarshal(b, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, v := range fixture.Vectors {
		out, err := Execute(v.Request)
		actual, _ := json.Marshal(out)
		expected, _ := json.Marshal(v.Expected)
		if err != nil || string(actual) != string(expected) {
			t.Fatalf("%s: got %+v, expected %+v; %v", v.Name, out, v.Expected, err)
		}
	}
	t.Logf("verified %d golden vectors", len(fixture.Vectors))
}

// Oracle initializes bare upstream globals in a fresh process, independently
// of the scheme snapshot adapter and foreign wrappers.
func TestPrimitiveOracle(t *testing.T) {
	scheme := os.Getenv("FZFP_TEST_ORACLE_SCHEME")
	if scheme == "" {
		for _, scheme := range []string{"default", "path", "history"} {
			cmd := exec.Command(os.Args[0], "-test.run=^TestPrimitiveOracle$", "-test.v")
			cmd.Env = append(os.Environ(), "FZFP_TEST_ORACLE_SCHEME="+scheme)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%s: %s %v", scheme, output, err)
			}
		}
		return
	}
	algo.Init(scheme)
	b, err := os.ReadFile("../testdata/golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Vectors []struct {
			Name     string
			Request  Request
			Expected Response
		}
	}
	if err := json.Unmarshal(b, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, v := range fixture.Vectors {
		r := v.Request
		if r.Operation != "match" || r.Options.Scheme != scheme {
			continue
		}
		q := r.Query
		if !r.Options.CaseSensitive {
			q = strings.ToLower(q)
		}
		norm := r.Options.Normalize == nil || *r.Options.Normalize
		for _, c := range q {
			if algo.FoldRune(c) != c {
				norm = false
			}
		}
		text := util.ToChars([]byte(r.Text))
		raw, positions := pinyin.FuzzyMatchV2(r.Options.CaseSensitive, norm, scheme != "path", &text, []rune(q), true, nil)
		want := v.Expected.Match
		if raw.Start != want.Start || raw.End != want.End || raw.Score != want.Score {
			t.Fatalf("%s primitive=%+v facade=%+v", v.Name, raw, want)
		}
		if r.Options.Positions == nil || *r.Options.Positions {
			var pp []int
			if positions != nil {
				pp = append(pp, (*positions)...)
				slices.Sort(pp)
			}
			if !slices.Equal(pp, want.Positions) {
				t.Fatalf("%s positions: %v != %v", v.Name, pp, want.Positions)
			}
		}
	}
}

func TestConcurrentMixedSchemes(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 120; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			scheme := []string{"default", "path", "history"}[i%3]
			out, err := Execute(Request{Operation: "match", Text: "/马超", Query: "machao", Options: Options{Scheme: scheme}})
			want := 158
			if scheme == "history" {
				want = 151
			}
			if err != nil || out.Match.Score != want {
				t.Errorf("%s: %+v %v", scheme, out, err)
			}
		}(i)
	}
	wg.Wait()
}

func TestInvalidRequests(t *testing.T) {
	for _, req := range []Request{{Operation: "unknown"}, {Operation: "match", Text: string([]byte{0xff})}, {Operation: "filter", Candidates: []string{string([]byte{0xff})}}, {Operation: "match", Options: Options{Scheme: "invalid"}}} {
		if _, err := Execute(req); err == nil {
			t.Fatalf("accepted %+v", req)
		}
	}
}
