package algo

import "github.com/junegunn/fzf/src/util"

// BoundaryBonus exposes the active fzf scheme's positional bonus for adapters.
// Call Init before matching, exactly as for the upstream algorithm functions.
func BoundaryBonus(text *util.Chars, index int) int {
	return int(bonusAt(text, index))
}

// FoldRune exposes fzf's single-rune accent normalization to matching adapters.
func FoldRune(r rune) rune { return normalizeRune(r) }
