package pinyin

import (
	"encoding/json"
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/junegunn/fzf/internal/fixture"
	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// Isolate the extra literal pass using exactly Wrap's eligibility condition.
// Selection is outside timing; this measures only the extra DP work, not a
// different matcher with different results. Retained indices exist only here.
func TestPerfLiteralPass(t *testing.T) {
	if os.Getenv("FZFP_PERF") == "" {
		t.Skip("set FZFP_PERF=1")
	}
	for _, c := range fixture.PerfCases() {
		if strings.HasPrefix(c.Name, "english") {
			continue
		}
		lines := strings.Split(strings.TrimSuffix(c.Corpus, "\n"), "\n")
		q, slab := []rune(c.Query), util.MakeSlab(100000, 2048)
		selected := []util.Chars{}
		for _, line := range lines {
			chars := util.ToChars([]byte(line))
			native, _ := algo.FuzzyMatchV2(false, true, true, &chars, q, false, slab)
			if native.Start < 0 || !HasHan(&chars) {
				continue
			}
			raw, _ := match(false, true, true, &chars, q, false, Fuzzy, true)
			if raw.Start >= 0 && raw.Score > native.Score {
				selected = append(selected, chars)
			}
		}
		engines := []struct {
			name string
			fn   func(*util.Chars)
		}{
			{"baseline", func(chars *util.Chars) { legacyMatch(false, true, true, chars, q, c.Positions, Fuzzy, false) }},
			{"combined", func(chars *util.Chars) { match(false, true, true, chars, q, c.Positions, Fuzzy, false) }},
		}
		scan := func(i int) {
			for j := range selected {
				engines[i].fn(&selected[j])
			}
		}
		durations := make([][]float64, len(engines))
		allocated, allocs := make([]uint64, len(engines)), make([]uint64, len(engines))
		for i := range engines {
			scan(i)
			durations[i] = make([]float64, 21)
		}
		for i := 0; i < 21; i++ {
			for k := range engines {
				j := k
				if i%2 != 0 {
					j = len(engines) - 1 - k
				}
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				start := time.Now()
				scan(j)
				durations[j][i] = float64(time.Since(start)) / 1e6
				runtime.ReadMemStats(&after)
				allocated[j] += after.TotalAlloc - before.TotalAlloc
				allocs[j] += after.Mallocs - before.Mallocs
			}
		}
		for i, e := range engines {
			slices.Sort(durations[i])
			r := struct {
				Case, Engine    string
				Calls           int
				MedianMS, P95MS float64
				Bytes, Allocs   uint64
			}{c.Name, e.name, len(selected), durations[i][10], durations[i][19], allocated[i] / 21, allocs[i] / 21}
			data, _ := json.Marshal(r)
			t.Log(string(data))
		}
	}
}
