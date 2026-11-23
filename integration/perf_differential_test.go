package integration

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/junegunn/fzf/internal/fixture"
)

// Set FZFP_BASELINE_BIN to a 76553e1d build. Compare full bytes, not top-k or
// membership alone; the historical and stress fixtures are shared with perf.
func TestPerfLegacyCLI(t *testing.T) {
	baseline := os.Getenv("FZFP_BASELINE_BIN")
	if baseline == "" {
		t.Skip("set FZFP_BASELINE_BIN to 76553e1d binary")
	}
	current := binary(t, "FZFP_BIN", "fzfp")
	count := 0
	for _, scheme := range []string{"default", "path", "history"} {
		compare := func(corpus, query string, options ...string) {
			t.Helper()
			args := append([]string{"--scheme=" + scheme, "--filter=" + query}, options...)
			want, got := filter(t, baseline, corpus, args...), filter(t, current, corpus, args...)
			if want != got {
				t.Fatalf("scheme=%s query=%q options=%q CLI bytes/exit differ", scheme, query, options)
			}
			count++
		}
		for _, c := range fixture.PerfCases() {
			if !c.Positions {
				compare(c.Corpus, c.Query)
			}
		}
		rng := rand.New(rand.NewSource(20261123))
		tokens := []string{"中", "国", "北京", "重行乐长得单解曾区薄", "马超", "女绿", "zhong", "BJ", "Go", "é", "🙂", "Ω", "/", "-", " ", "aa"}
		var corpus strings.Builder
		for i := 0; i < 100000; i++ {
			fmt.Fprintf(&corpus, "generated/%06d/", i)
			for n := 1 + rng.Intn(20); n > 0; n-- {
				corpus.WriteString(tokens[rng.Intn(len(tokens))])
			}
			corpus.WriteByte('\n')
		}
		for _, q := range []string{"zg", "zhongguo", "bj", "beijing", "chonghangyue", "mh", "Go", "BJ", "nv", "北jing", "zhongg", "'zg", "^zg", "zg$", "^zg$", "'zg'", "!zg", "zg | bj", "zg !bj", "国"} {
			compare(corpus.String(), q)
		}
		for _, opts := range [][]string{{"--algo=v1"}, {"--exact"}, {"+i", "--literal"}, {"--tac"}, {"--no-sort"}, {"--delimiter=/", "--nth=3..", "--with-nth=3,1"}, {"--tiebreak=begin,end,length"}} {
			compare(corpus.String(), "zg", opts...)
		}
	}
	t.Logf("%d complete CLI byte/exit comparisons", count)
}
