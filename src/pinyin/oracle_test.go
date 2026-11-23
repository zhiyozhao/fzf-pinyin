package pinyin

import (
	"math/rand"
	"strings"
	"testing"
	"unicode"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// An exhaustive alignment oracle: enumerate complete paths, then score each
// path as a whole. This does not share the production DP, its state pruning,
// dictionary lookup, or spelling builder. Small inputs make exhaustive search
// practical and expose mistakes when competing adjacent/gapped paths merge.
func exhaustive(text, q string) (int, bool) {
	type edge struct{ index, letters, cost int }
	type form struct {
		text string
		cost int
	}
	known := map[rune][]form{
		'中': {{"zhong", 1}, {"zh", 2}, {"z", 3}},
		'国': {{"guo", 1}, {"g", 2}},
		'安': {{"an", 1}, {"a", 3}},
	}
	runes, qr := []rune(text), []rune(q)
	c := util.RunesToChars(runes)
	best, found := 0, false
	var search func(int, int, []edge)
	search = func(start, offset int, path []edge) {
		if offset == len(qr) {
			score := 0
			for j, e := range path {
				bonus := algo.BoundaryBonus(&c, e.index)
				if e.index > 0 {
					a, b := runes[e.index-1], runes[e.index]
					if unicode.Is(unicode.Han, a) && unicode.Is(unicode.Han, b) {
						bonus = max(bonus, 8)
					}
					if unicode.IsLetter(a) && unicode.IsLetter(b) && unicode.Is(unicode.Han, a) != unicode.Is(unicode.Han, b) {
						bonus = max(bonus, 8)
					}
				}
				score += 16*e.letters + 4*(e.letters-1) - e.cost
				if j == 0 {
					score += 2 * bonus
				} else {
					gap := e.index - path[j-1].index - 1
					if gap == 0 {
						score += max(8, bonus)
					} else {
						score += bonus - 3 - (gap - 1)
					}
				}
			}
			best, found = max(best, max(1, score)), true
			return
		}
		for i := start; i < len(runes); i++ {
			forms := append([]form{{string(runes[i]), 0}}, known[runes[i]]...)
			for _, f := range forms {
				spelling := []rune(f.text)
				k := min(len(spelling), len(qr)-offset)
				if string(spelling[:k]) != string(qr[offset:offset+k]) {
					continue
				}
				cost := f.cost
				if k < len(spelling) {
					cost = max(4, cost)
				}
				search(i+1, offset+k, append(path, edge{i, k, cost}))
			}
		}
	}
	search(0, 0, nil)
	return best, found
}

func TestExhaustiveOptimality(t *testing.T) {
	rng := rand.New(rand.NewSource(1729))
	alphabet := []rune("中国安gza- 🙂")
	queries := []string{"z", "zh", "zho", "zhong", "g", "guo", "a", "an", "zg", "zz", "gz", "aa", "zhg", "zhog", "zgg", "agg", "zzz", "中国", "中g"}
	for trial := 0; trial < 4000; trial++ {
		text := "中"
		for j, n := 0, 1+rng.Intn(6); j < n; j++ {
			text += string(alphabet[rng.Intn(len(alphabet))])
		}
		q := queries[rng.Intn(len(queries))]
		if trial%2 == 0 {
			q += queries[rng.Intn(7)]
		}
		want, recalled := exhaustive(text, q)
		// The public adapter preserves native scoring for queries containing
		// no phonetic letters (including purely literal Chinese queries).
		if !strings.ContainsAny(q, "abcdefghijklmnopqrstuvwxyzü") {
			want, recalled = 0, false
		}
		native, _ := run(algo.FuzzyMatchV2, text, q, false)
		if native.Start >= 0 {
			want = max(want, native.Score)
			recalled = true
		}
		got, _ := run(FuzzyMatchV2, text, q, trial%2 == 0)
		if (got.Start >= 0) != recalled || recalled && got.Score != want {
			t.Fatalf("trial=%d text=%q q=%q: got %+v; exhaustive score=%d recall=%v", trial, text, q, got, want, recalled)
		}
	}
	// The maximum score should come from a later, adjacent alignment when
	// an earlier high-boundary but gapped alignment would otherwise mask it.
	for _, text := range []string{"中中-国", "中-中国", "中中---中国", "中 国中国"} {
		want, _ := exhaustive(text, "zg")
		got, _ := run(FuzzyMatchV2, text, "zg", false)
		if got.Score != want {
			t.Fatalf("%s: %d != %d", strings.TrimSpace(text), got.Score, want)
		}
	}
}
