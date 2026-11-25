package pinyin

import (
	"unicode"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// LibraryLetterMask is a conservative corpus prefilter, not a matcher. It
// includes every literal ASCII letter after case/accent folding plus all
// synthetic spelling letters, including IME aliases. It deliberately ignores
// order, multiplicity, case mode and normalization mode: false positives are
// fine, false negatives are not. Non-ASCII query runes impose no constraint.
// Compute once when ingesting a fixed corpus, outside the scheme lock.
func LibraryLetterMask(chars *util.Chars) uint32 {
	var mask uint32
	add := func(r rune) {
		if r >= 'a' && r <= 'z' {
			mask |= 1 << (r - 'a')
		}
	}
	for i := 0; i < chars.Length(); i++ {
		r := chars.Get(i)
		add(unicode.ToLower(r))
		add(algo.FoldRune(unicode.ToLower(r)))
		add(unicode.ToLower(algo.FoldRune(r)))
		add(unicode.ToLower(algo.FoldRune(unicode.ToLower(r))))
		for _, f := range spellings(r) {
			for _, c := range f.text {
				add(c)
			}
		}
	}
	return mask
}
