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

// Opt-in quantiles for complete serial corpus scans (not Go benchmark averages).
func TestPerfQuantiles(t *testing.T) {
	if os.Getenv("FZFP_PERF") == "" {
		t.Skip("set FZFP_PERF=1")
	}
	for _, c := range fixture.PerfCases() {
		if pattern := os.Getenv("FZFP_PERF_CASE"); pattern != "" && !strings.Contains(c.Name, pattern) {
			continue
		}
		lines := strings.Split(strings.TrimSuffix(c.Corpus, "\n"), "\n")
		chars := make([]util.Chars, len(lines))
		for i, s := range lines {
			chars[i] = util.ToChars([]byte(s))
		}
		q, slab := []rune(c.Query), util.MakeSlab(100000, 2048)
		engines := []struct {
			name string
			fn   algo.Algo
		}{{"baseline", legacyFuzzy}, {"optimized", FuzzyMatchV2}}
		if strings.HasPrefix(c.Name, "english") {
			engines = append(engines, struct {
				name string
				fn   algo.Algo
			}{"upstream", algo.FuzzyMatchV2})
		}
		scan := func(fn algo.Algo) {
			for i := range chars {
				fn(false, true, true, &chars[i], q, c.Positions, slab)
			}
		}
		durations := make([][]float64, len(engines))
		bytes, allocs := make([]uint64, len(engines)), make([]uint64, len(engines))
		for j, f := range engines {
			scan(f.fn)
			durations[j] = make([]float64, 21)
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
				scan(engines[j].fn)
				durations[j][i] = float64(time.Since(start)) / 1e6
				runtime.ReadMemStats(&after)
				bytes[j] += after.TotalAlloc - before.TotalAlloc
				allocs[j] += after.Mallocs - before.Mallocs
			}
		}
		for j, f := range engines {
			slices.Sort(durations[j])
			record := struct {
				Case, Engine    string
				MedianMS, P95MS float64
				Bytes, Allocs   uint64
			}{c.Name, f.name, durations[j][10], durations[j][19], bytes[j] / 21, allocs[j] / 21}
			data, _ := json.Marshal(record)
			t.Log(string(data))
		}
	}
}

// Standard Go benchmark counterpart to the quantiles above, one full scan/op.
func BenchmarkPerfScan(b *testing.B) {
	for _, c := range fixture.PerfCases() {
		b.Run(c.Name, func(b *testing.B) {
			lines := strings.Split(strings.TrimSuffix(c.Corpus, "\n"), "\n")
			chars := make([]util.Chars, len(lines))
			for i, s := range lines {
				chars[i] = util.ToChars([]byte(s))
			}
			q, slab := []rune(c.Query), util.MakeSlab(100000, 2048)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for j := range chars {
					FuzzyMatchV2(false, true, true, &chars[j], q, c.Positions, slab)
				}
			}
		})
	}
}
