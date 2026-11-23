package algo

import "github.com/junegunn/fzf/src/util"

// BoundaryBonus exposes the active fzf scheme's positional bonus for adapters.
// Call Init before matching, exactly as for the upstream algorithm functions.
func BoundaryBonus(text *util.Chars, index int) int {
	if index == 0 {
		// path starts as a delimiter (9), not whitespace (8). bonusAt's
		// index-zero shortcut is for upstream exact-match heuristics.
		return int(bonusMatrix[initialCharClass][charClassOf(text.Get(0))])
	}
	return int(bonusAt(text, index))
}

// FoldRune exposes fzf's single-rune accent normalization to matching adapters.
func FoldRune(r rune) rune { return normalizeRune(r) }
