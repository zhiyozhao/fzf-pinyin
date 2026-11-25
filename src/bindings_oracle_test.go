package fzf

// This is a test-only oracle, not a library API. CLI --filter emits text, not
// score/positions. Order is checked byte-for-byte against the real executable;
// alignment/score come from the CLI's BuildPattern/MatchItem/buildResult path,
// using direction observed by a spy in the real Run patternBuilder. No binding
// matcher or direction helper participates in producing expected results.
import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/junegunn/fzf/bindings/core"
	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

type bindingCLIVector struct {
	Name     string        `json:"name"`
	Request  core.Request  `json:"request"`
	Expected core.Response `json:"expected"`
}

func TestBindingsCLIDifferential(t *testing.T) {
	if scheme := os.Getenv("FZFP_CLI_ORACLE_CHILD"); scheme != "" {
		bindingsCLIChild(t, scheme)
		return
	}
	dir := os.Getenv("FZFP_CLI_ORACLE_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	cli := os.Getenv("FZFP_CLI")
	if cli == "" {
		cli = filepath.Join(t.TempDir(), "fzfp")
		cmd := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-o", cli, "..")
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build CLI: %s %v", b, err)
		}
	}
	cli, err = filepath.Abs(cli)
	if err != nil {
		t.Fatal(err)
	}
	for _, scheme := range []string{"default", "path", "history"} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestBindingsCLIDifferential$", "-test.v")
		cmd.Env = append(os.Environ(), "FZFP_CLI_ORACLE_CHILD="+scheme, "FZFP_CLI_ORACLE_DIR="+dir, "FZFP_CLI="+cli)
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: %s %v", scheme, b, err)
		} else {
			t.Log(string(b))
		}
	}
}

// Observe the actual core.go direction/withPos loop via Run. The spy sees the
// arguments supplied by Pattern to the native fallback inside pinyin.Wrap.
func probeBindingCLI(t *testing.T, scheme, tiebreak string, extra ...string) (*Options, bool, bool) {
	t.Helper()
	args := []string{"--scheme=" + scheme, "--no-extended", "-i", "--filter=st"}
	if tiebreak != "" {
		args = append(args, "--tiebreak="+tiebreak)
	}
	args = append(args, extra...)
	opts, err := ParseOptions(false, args)
	if err != nil {
		t.Fatal(err)
	}
	opts.Input = make(chan string, 1)
	opts.Input <- "case-test_qux"
	close(opts.Input)
	opts.Output = make(chan string, 4)
	var forward, withPos, called bool
	var mu sync.Mutex
	opts.FuzzyAlgo = func(cs, norm, f bool, chars *util.Chars, q []rune, p bool, slab *util.Slab) (algo.Result, *[]int) {
		mu.Lock()
		forward, withPos, called = f, p, true
		mu.Unlock()
		return algo.FuzzyMatchV2(cs, norm, f, chars, q, p, slab)
	}
	// Reset schemes to pristine snapshots before Run's own Init. A standalone
	// CLI starts pristine; calling Init(path), Init(default) in one test does not.
	algo.WithLibraryScheme(scheme, func() {
		if code, err := Run(opts); err != nil || code != 0 {
			t.Fatalf("Run probe: %d %v", code, err)
		}
	})
	if !called {
		t.Fatal("direction spy was not called")
	}
	opts.FuzzyAlgo = algo.FuzzyMatchV2
	return opts, forward, withPos
}

func bindingsCLIChild(t *testing.T, scheme string) {
	opts, forward, withPos := probeBindingCLI(t, scheme, "")
	if forward != (scheme != "path") || withPos != (scheme == "path") {
		t.Fatalf("unexpected scheme policy %s %v %v", scheme, forward, withPos)
	}
	{
		b, err := os.ReadFile("../bindings/testdata/direction.json")
		if err != nil {
			t.Fatal(err)
		}
		var cases []struct {
			Tiebreak  string
			Forward   bool
			Positions bool `json:"with_positions"`
		}
		if err := json.Unmarshal(b, &cases); err != nil {
			t.Fatal(err)
		}
		for _, c := range cases {
			_, f, p := probeBindingCLI(t, scheme, c.Tiebreak)
			if f != c.Forward || p != c.Positions {
				t.Fatalf("CLI criteria %s: %v/%v", c.Tiebreak, f, p)
			}
		}
	}
	for _, flags := range [][]string{{"--tac"}, {"--no-sort"}, {"--scheme=" + scheme, "--tiebreak=begin"}, {"--tiebreak=begin", "--scheme=" + scheme}} {
		_, f, _ := probeBindingCLI(t, scheme, "", flags...)
		want := scheme != "path"
		if slices.Contains(flags, "--tiebreak=begin") && flags[len(flags)-1] == "--tiebreak=begin" {
			want = true
		}
		if f != want {
			t.Fatalf("CLI flags %s %v: forward=%v want=%v", scheme, flags, f, want)
		}
	}
	rng := rand.New(rand.NewSource(20261125))
	queries := []string{"st", "t", "s", "aa", "a", "b", "ab", "tt", "bj", "zg", "z", "g", "lv", "en", "ch", "sh", "foo", "fb", "e", ""}
	latin := []string{"case", "test", "qux", "cherry", "world", "finder", "match", "grape", "peach", "score", "lemon", "filter", "aab", "abb", "aa", "fooBar", "foofoo"}
	han := []string{"北京", "中中国", "中国国", "绿绿茶", "嗯嗯好", "长沙沙", "重庆庆", "音乐乐", "中北京国", "巨女略"}
	seps := []string{"/", "\\", "_", "-", " ", ".", ""}
	data := make([]bindingCLIVector, 0, 420)
	for n := 0; n < 420; n++ {
		texts := []string{"case-test_qux", "test-cherry.index", "test_cherry-world", "test_finder-match", "grape-peach test", "score-lemon-filter", "北京北京", "中中国国", "😀北京北京", "e\u0301中国中国"}
		pool := latin
		if n%3 == 1 {
			pool = han
		} else if n%3 == 2 {
			pool = append(slices.Clone(latin), han...)
		}
		count := 24 + rng.Intn(70)
		if n%100 == 0 {
			count = 1100
		}
		for i := 0; i < count; i++ {
			var s strings.Builder
			for j := 0; j < 2+rng.Intn(4); j++ {
				if j > 0 {
					s.WriteString(seps[rng.Intn(len(seps))])
				}
				s.WriteString(pool[rng.Intn(len(pool))])
			}
			text := s.String()
			if i%7 == 0 {
				text = " " + text + " "
			}
			texts = append(texts, text)
			if i%9 == 0 {
				texts = append(texts, text)
			}
		}
		q := queries[n%len(queries)]
		if n == 0 {
			q = "st"
		}
		yes := true
		req := core.Request{Operation: "filter", Candidates: texts, Query: q, Options: core.Options{Scheme: scheme, Normalize: &yes, Positions: &yes}}
		var expected core.Response
		algo.WithLibraryScheme(scheme, func() {
			sortCriteria = opts.Criteria
			pattern := BuildPattern(NewChunkCache(), make(map[string]*Pattern), true, algo.FuzzyMatchV2, false, CaseIgnore, true, forward, withPos, false, nil, Delimiter{}, revision{}, []rune(q), nil, 0)
			items := make([]Item, len(texts))
			results := []Result{}
			alignments := make(map[int32]core.Match)
			slab := util.MakeSlab(slab16Size, slab32Size)
			for i, text := range texts {
				items[i].text = util.ToChars([]byte(text))
				items[i].text.Index = int32(i)
				r, offsets, pp := pattern.MatchItem(&items[i], true, slab)
				if r.item == nil {
					continue
				}
				positions := []int{}
				if pp != nil {
					positions = append(positions, (*pp)...)
					slices.Sort(positions)
				}
				m := core.Match{Matched: true, Score: int(65535 - r.points[3]), Start: int(offsets[0][0]), End: int(offsets[0][1]), Positions: positions, UTF8Positions: []int{}, UTF16Positions: []int{}}
				p, ri, u := 0, 0, 0
				for b, c := range text {
					if p < len(positions) && positions[p] == ri {
						m.UTF8Positions = append(m.UTF8Positions, b)
						m.UTF16Positions = append(m.UTF16Positions, u)
						p++
					}
					ri++
					u++
					if c > 0xffff {
						u++
					}
				}
				alignments[int32(i)] = m
				results = append(results, r)
			}
			if q != "" {
				sort.Sort(ByRelevance(results))
			}
			expected.Results = []core.Hit{}
			for _, r := range results {
				expected.Results = append(expected.Results, core.Hit{Index: int(r.Index()), Match: alignments[r.Index()]})
			}
		})
		var input, want bytes.Buffer
		for _, s := range texts {
			input.WriteString(s)
			input.WriteByte(0)
		}
		for _, h := range expected.Results {
			want.WriteString(texts[h.Index])
			want.WriteByte(0)
		}
		cmd := exec.Command(os.Getenv("FZFP_CLI"), "--scheme="+scheme, "--no-extended", "-i", "--read0", "--print0", "--filter="+q)
		cmd.Env = append(os.Environ(), "FZF_DEFAULT_OPTS=", "FZF_DEFAULT_OPTS_FILE=/dev/null")
		cmd.Stdin = &input
		got, err := cmd.Output()
		if err != nil {
			if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 || len(expected.Results) != 0 {
				t.Fatalf("CLI %s/%d: %v", scheme, n, err)
			}
		}
		if !bytes.Equal(got, want.Bytes()) {
			t.Fatalf("CLI pipeline order mismatch %s/%d %q", scheme, n, q)
		}
		out, err := core.Execute(req)
		a, _ := json.Marshal(out)
		b, _ := json.Marshal(expected)
		if err != nil || !bytes.Equal(a, b) {
			t.Fatalf("binding Execute vs CLI pipeline %s/%d %q: %v\ngot %s\nwant %s", scheme, n, q, err, a, b)
		}
		for _, workers := range []int{1, 4} {
			x, err := core.NewIndex(texts, req.Options, workers)
			if err != nil {
				t.Fatal(err)
			}
			for _, limit := range []int{-1, 1, 7, 50} {
				for _, positions := range []bool{false, true} {
					hits, _, err := x.Find(q, limit, positions, false)
					if err != nil {
						t.Fatal(err)
					}
					want := expected.Results
					if limit >= 0 && len(want) > limit {
						want = want[:limit]
					}
					if len(hits) != len(want) {
						t.Fatal("Index hit count")
					}
					for i, h := range hits {
						w := want[i]
						if h.Index != w.Index || h.Score != w.Score || positions && (h.Start != w.Start || h.End != w.End || !slices.Equal(h.Positions, w.Positions)) {
							t.Fatalf("Index vs CLI pipeline %s/%d workers=%d limit=%d pos=%v: %+v != %+v", scheme, n, workers, limit, positions, h, w)
						}
					}
				}
			}
		}
		data = append(data, bindingCLIVector{fmt.Sprintf("cli:%s:%d", scheme, n), req, expected})
	}
	b, err := json.Marshal(struct {
		Vectors []bindingCLIVector `json:"vectors"`
	}{data})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(os.Getenv("FZFP_CLI_ORACLE_DIR"), scheme+".json"), b, 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s: %d tie-heavy CLI/pipeline/Go datasets, exact scores/bounds/ascending positions", scheme, len(data))
}
