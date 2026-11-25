package pinyin

import (
	"testing"
	"unicode"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
	data "github.com/mozillazg/go-pinyin"
)

func TestLibraryMaskSpellings(t *testing.T) {
	// Every dictionary entry/IME form and every ASCII-foldable Unicode scalar
	// must be represented. The mask can over-include, never under-include.
	for c := range data.PinyinDict {
		r := rune(c)
		chars := util.RunesToChars([]rune{r})
		mask := LibraryLetterMask(&chars)
		for _, f := range spellings(r) {
			for _, q := range f.text {
				if mask&(1<<(q-'a')) == 0 {
					t.Fatalf("%U %s", r, f.text)
				}
			}
		}
	}
	for r := rune(0); r <= unicode.MaxRune; r++ {
		for _, v := range []rune{r, unicode.ToLower(r), algo.FoldRune(r), algo.FoldRune(unicode.ToLower(r))} {
			v = unicode.ToLower(v)
			if v >= 'a' && v <= 'z' {
				chars := util.RunesToChars([]rune{r})
				if LibraryLetterMask(&chars)&(1<<(v-'a')) == 0 {
					t.Fatalf("literal %U -> %c", r, v)
				}
			}
		}
	}
}
