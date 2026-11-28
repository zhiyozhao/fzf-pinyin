package pinyin

import (
	"reflect"
	"strings"
	"testing"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

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
