package pinyin

import (
	"math/rand"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/junegunn/fzf/internal/fixture"
	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// Compare every field and exact position order, including nil vs empty. The
// frozen reference uses its own state, transitions and tie-breaking helpers.
// Default runs sample every corpus category plus the long-line regressions.
// Before merging changes to pinyin matching, run the complete corpus locally:
// FZFP_FULL_DIFF=1 go test -timeout 20m -count=1 -run '^TestLegacyDifferential$' -v ./src/pinyin
func TestLegacyDifferential(t *testing.T) {
	if raceEnabled {
		// The non-race phase covers this serial differential; the complete
		// corpus is an explicit local check. Race adds no concurrency coverage.
		t.Skip("serial differential runs without -race; race instrumentation adds no concurrency coverage")
	}
	randomCases, corpusLimit, scale := 2000, 128, "sample"
	if os.Getenv("FZFP_FULL_DIFF") == "1" {
		randomCases, corpusLimit, scale = 60000, 0, "full"
	}
	for _, scheme := range testSchemes(t) {
		algo.Init(scheme)
		rng := rand.New(rand.NewSource(20261123))
		count := 0
		fallbacks := []algo.Algo{algo.FuzzyMatchV2, algo.ExactMatchNaive, algo.PrefixMatch, algo.SuffixMatch, algo.EqualMatch, algo.ExactMatchBoundary}
		oldWrapped, newWrapped := make([]algo.Algo, 6), make([]algo.Algo, 6)
		for mode, f := range fallbacks {
			oldWrapped[mode] = legacyWrap(f, Mode(mode))
			newWrapped[mode] = Wrap(f, Mode(mode))
		}
		check := func(line, query string, mode Mode, cs, norm, forward, pos bool) {
			t.Helper()
			chars, q := util.ToChars([]byte(line)), []rune(query)
			for _, positions := range []bool{pos, false} {
				for _, allow := range []bool{true, false} {
					want, wp := legacyMatch(cs, norm, forward, &chars, q, positions, mode, allow)
					got, gp := match(cs, norm, forward, &chars, q, positions, mode, allow)
					if got != want || !reflect.DeepEqual(gp, wp) {
						t.Fatalf("%s mode=%d allow=%t cs=%t norm=%t forward=%t pos=%t line=%q query=%q: got %+v %v want %+v %v", scheme, mode, allow, cs, norm, forward, positions, line, query, got, gp, want, wp)
					}
					count++
				}
				want, wp := oldWrapped[mode](cs, norm, forward, &chars, q, positions, nil)
				got, gp := newWrapped[mode](cs, norm, forward, &chars, q, positions, nil)
				if got != want || !reflect.DeepEqual(gp, wp) {
					t.Fatalf("%s Wrap mode=%d forward=%t positions=%t line=%q query=%q: got %+v %v want %+v %v", scheme, mode, forward, positions, line, query, got, gp, want, wp)
				}
				count++
			}
		}
		tokens := []string{"中", "国", "北", "京", "重", "行", "乐", "长", "得", "单", "解", "曾", "区", "薄", "女", "绿", "略", "居", "鱼", "音", "万", "嗯", "呣", "噷", "欸", "呒", "嘸", "〇", "𠀀", "A", "b", "é", "Ü", "ü", "/", "-", " ", "🙂", "\t", "北京", "中国", "zhong", "Beijing", "Ω"}
		queries := []string{"zhongguo", "zg", "beijing", "bj", "zhongg", "chonghangyuezhang", "pr", "SDK", "北jing", "nü", "lv", "lue", "jv", "yv", "v", "in", "uan", "en", "ng", "mu", "hm", "e", "Ü", "a", "b", "国", " ", strings.Repeat("zhongguo", 6)}
		// The winning interval contains a huge gap; exercise live-trace GC as
		// well as interval replay, including negative scores clamped to one.
		for _, forward := range []bool{false, true} {
			check("中"+strings.Repeat("国", 100000)+"北", "zhongguobei", Fuzzy, false, true, forward, true)
			check("中a"+strings.Repeat("bc", 50000)+"d", "abcd", Fuzzy, false, true, forward, true)
		}
		// Above the retained scratch capacity, then shrink again in the random
		// cases below: oversized arrays must not leave stale state in the pool.
		check(strings.Repeat("中国", 40), strings.Repeat("zhongguo", 40), Fuzzy, false, true, true, true)
		for _, forward := range []bool{false, true} {
			for mode := Fuzzy; mode <= Boundary; mode++ {
				check("田"+strings.Repeat("abc", 33334), "abc", mode, false, true, forward, true)
				check("中a"+strings.Repeat("国", 100000)+"b", "ab", mode, false, true, forward, true)
			}
		}
		for i := 0; i < randomCases; i++ {
			var line strings.Builder
			for n := 1 + rng.Intn(45); n > 0; n-- {
				line.WriteString(tokens[rng.Intn(len(tokens))])
			}
			q := queries[rng.Intn(len(queries))]
			if i%3 == 0 {
				var s strings.Builder
				for n := 1 + rng.Intn(16); n > 0; n-- {
					s.WriteString(tokens[rng.Intn(len(tokens))])
				}
				q = s.String()
			}
			check(line.String(), q, Mode(rng.Intn(6)), rng.Intn(2) == 0, rng.Intn(2) == 0, rng.Intn(2) == 0, true)
		}
		// Sample across the entire corpus, including first/last rows and
		// nonmatches, rather than taking a prefix of the generated data.
		// Full mode checks every candidate. Both scales alternate directions;
		// check() always compares with and without positions and pinyin edges.
		for _, c := range fixture.PerfCases() {
			lines := strings.Split(strings.TrimSuffix(c.Corpus, "\n"), "\n")
			samples := len(lines)
			if corpusLimit > 0 {
				samples = min(samples, corpusLimit)
			}
			for j := 0; j < samples; j++ {
				index := 0
				if samples > 1 {
					index = j * (len(lines) - 1) / (samples - 1)
				}
				check(lines[index], c.Query, Fuzzy, false, true, j%2 == 0, true)
			}
			// Keep original extreme lengths in both scales. Every single-line
			// category (including 10k runes) gets all modes and both directions.
			if len(lines) == 1 {
				for _, mode := range []Mode{Fuzzy, Exact, Prefix, Suffix, Equal, Boundary} {
					for _, forward := range []bool{false, true} {
						check(lines[0], c.Query, mode, false, true, forward, true)
					}
				}
			}
			t.Logf("%s/%s: %d/%d corpus rows", scheme, c.Name, samples, len(lines))
		}
		t.Logf("%s/%s: %d random cases; %d score/range/positions comparisons", scheme, scale, randomCases, count)
	}
}
