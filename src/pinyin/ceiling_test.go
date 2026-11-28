package pinyin

import (
	"reflect"
	"strings"
	"testing"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// Assert the production bound itself, not just observable matches: a one-point
// underestimate may fall between reachable native scores and evade black-box tests.
func TestScoreCeilingFormula(t *testing.T) {
	for _, tc := range []struct {
		scheme string
		want   [5]int // query lengths 0, 1, 2, 3, 100
	}{
		{"default", [5]int{10, 36, 62, 88, 2610}},
		{"path", [5]int{9, 34, 59, 84, 2509}},
		{"history", [5]int{8, 32, 56, 80, 2408}},
	} {
		algo.WithLibraryScheme(tc.scheme, func() {
			for i, n := range []int{0, 1, 2, 3, 100} {
				if got := scoreCeiling(n); got != tc.want[i] {
					t.Errorf("%s/%d: ceiling=%d want %d", tc.scheme, n, got, tc.want[i])
				}
			}
		})
	}
}

func TestScoreCeilingNearLimitCompetition(t *testing.T) {
	for _, tc := range []struct {
		scheme string
		score  int
	}{
		{"default", 61}, {"path", 59}, {"history", 56},
	} {
		algo.WithLibraryScheme(tc.scheme, func() {
			chars := util.ToChars([]byte("安/an"))
			for _, forward := range []bool{false, true} {
				for _, positions := range []bool{false, true} {
					got, gp := FuzzyMatchV2(false, true, forward, &chars, []rune("an"), positions, nil)
					want, wp := legacyWrap(algo.FuzzyMatchV2, Fuzzy)(false, true, forward, &chars, []rune("an"), positions, nil)
					if got.Score != tc.score || got != want || !reflect.DeepEqual(gp, wp) {
						t.Fatalf("%s forward=%t positions=%t: got %v/%v want %v/%v score=%d", tc.scheme, forward, positions, got, gp, want, wp, tc.score)
					}
				}
			}
		})
	}
}

func TestScoreCeilingPreservesNativeAlignment(t *testing.T) {
	for _, scheme := range []string{"default", "path", "history"} {
		algo.WithLibraryScheme(scheme, func() {
			fallbacks := []algo.Algo{algo.FuzzyMatchV2, algo.ExactMatchNaive, algo.PrefixMatch, algo.SuffixMatch, algo.EqualMatch, algo.ExactMatchBoundary}
			for mode, fallback := range fallbacks {
				for _, pair := range [][2]string{{"abc abc中国", "abc"}, {" abc/abc 中国", "abc"}, {"fooBar 中", "fb"}, {"中国café", "cafe"}, {"田abc", "abc"}, {"abc" + strings.Repeat("0", 70000) + "中国abc", "abc"}} {
					chars := util.ToChars([]byte(pair[0]))
					for _, forward := range []bool{false, true} {
						for _, positions := range []bool{false, true} {
							got, gp := Wrap(fallback, Mode(mode))(false, true, forward, &chars, []rune(pair[1]), positions, nil)
							want, wp := legacyWrap(fallback, Mode(mode))(false, true, forward, &chars, []rune(pair[1]), positions, nil)
							if got != want || !reflect.DeepEqual(gp, wp) {
								t.Fatalf("%s mode=%d forward=%t positions=%t: got %v/%v want %v/%v", scheme, mode, forward, positions, got, gp, want, wp)
							}
						}
					}
				}
			}
		})
	}
}

func TestSparseEligibilityCacheOverflow(t *testing.T) {
	var runes []rune
	for r := rune(0x3400); r < 0x3400+4500; r++ {
		runes = append(runes, r)
	}
	chars := util.ToChars([]byte(string(runes) + "ſKéÜ女𰻝"))
	for _, scheme := range []string{"default", "path", "history"} {
		algo.WithLibraryScheme(scheme, func() {
			for _, query := range []string{"sk", "cafe", "nv", "nü", "biang"} {
				for _, forward := range []bool{false, true} {
					q := []rune(query)
					got, gp := matchRange(false, true, forward, &chars, q, true, Fuzzy, true, 0, chars.Length(), false)
					want, wp := denseMatchRange(false, true, forward, &chars, q, true, Fuzzy, true, 0, chars.Length(), false)
					if got != want || !reflect.DeepEqual(gp, wp) {
						t.Fatalf("%s/%s/%t: got %v/%v want %v/%v", scheme, query, forward, got, gp, want, wp)
					}
				}
			}
		})
	}
}
