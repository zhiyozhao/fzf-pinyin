package pinyin

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/junegunn/fzf/internal/fixture"
	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// Compare every field and exact position order, including nil vs empty. The
// frozen reference uses its own state, transitions and tie-breaking helpers.
func TestLegacyDifferential(t *testing.T) {
	for _, scheme := range testSchemes(t) {
		algo.Init(scheme)
		rng := rand.New(rand.NewSource(20261123))
		count := 0
		check := func(line, query string, mode Mode, cs, norm, forward, pos bool) {
			t.Helper()
			chars, q := util.ToChars([]byte(line)), []rune(query)
			want, wp := legacyMatch(cs, norm, forward, &chars, q, pos, mode)
			got, gp := match(cs, norm, forward, &chars, q, pos, mode)
			if got != want || !reflect.DeepEqual(gp, wp) {
				t.Fatalf("%s mode=%d cs=%t norm=%t forward=%t pos=%t line=%q query=%q: got %+v %v want %+v %v", scheme, mode, cs, norm, forward, pos, line, query, got, gp, want, wp)
			}
			count++
			// Position-free calls take a distinct path for large inputs.
			want, wp = legacyMatch(cs, norm, forward, &chars, q, false, mode)
			got, gp = match(cs, norm, forward, &chars, q, false, mode)
			if got != want || !reflect.DeepEqual(gp, wp) {
				t.Fatalf("%s mode=%d query=%q without positions: got %+v %v want %+v %v", scheme, mode, query, got, gp, want, wp)
			}
			count++
		}
		tokens := []string{"中", "国", "北", "京", "重", "行", "乐", "长", "得", "单", "解", "曾", "区", "薄", "女", "绿", "〇", "𠀀", "A", "b", "é", "Ü", "ü", "/", "-", " ", "🙂", "\t", "北京", "中国", "zhong", "Beijing", "Ω"}
		queries := []string{"zhongguo", "zg", "beijing", "bj", "zhongg", "chonghangyuezhang", "pr", "SDK", "北jing", "nü", "lv", "Ü", "a", "b", "国", " ", strings.Repeat("zhongguo", 6)}
		// The winning interval contains a huge gap; exercise live-trace GC as
		// well as interval replay, including negative scores clamped to one.
		for _, forward := range []bool{false, true} {
			check("中"+strings.Repeat("国", 100000)+"北", "zhongguobei", Fuzzy, false, true, forward, true)
		}
		// Above the retained scratch capacity, then shrink again in the random
		// cases below: oversized arrays must not leave stale state in the pool.
		check(strings.Repeat("中国", 40), strings.Repeat("zhongguo", 40), Fuzzy, false, true, true, true)
		for i := 0; i < 60000; i++ {
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
		// Every 100k candidate, not just matched candidates, plus generated extremes.
		for _, c := range fixture.PerfCases() {
			for _, line := range strings.Split(strings.TrimSuffix(c.Corpus, "\n"), "\n") {
				check(line, c.Query, Fuzzy, false, true, true, true)
			}
			if len(c.Corpus) > 100000 && !strings.Contains(c.Corpus[:len(c.Corpus)-1], "\n") {
				for _, mode := range []Mode{Fuzzy, Exact, Prefix, Suffix, Equal, Boundary} {
					for _, forward := range []bool{false, true} {
						check(strings.TrimSuffix(c.Corpus, "\n"), c.Query, mode, false, true, forward, true)
					}
				}
			}
		}
		t.Logf("%s: %d score/range/positions comparisons", scheme, count)
	}
}
