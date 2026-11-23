// Package pinyin provides fzf-compatible matching with Mandarin spelling edges.
// Candidates without Han characters call the supplied upstream algorithm
// unchanged, including its score, offsets, position order and nil conventions.
//
// Call algo.Init once before concurrent use. Functions accept an already folded
// query, following algo.Algo's contract; fzf's Pattern layer handles smart case,
// normalization and extended search syntax. Returned offsets count original
// runes, never UTF-8 bytes or expanded pinyin letters.
// Synthetic Han spelling edges ignore case; literal Latin edges obey the
// caller's caseSensitive flag. This does not alter non-Han matching.
package pinyin

import (
	"strings"
	"sync"
	"unicode"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
	data "github.com/mozillazg/go-pinyin"
)

// Mode controls adjacency and anchors in original-text coordinates.
type Mode uint8

const (
	Fuzzy    Mode = iota // Original runes may be skipped, with an affine gap cost.
	Exact                // No skipped runes inside the match.
	Prefix               // Exact, anchored after optional leading whitespace.
	Suffix               // Exact, anchored before optional trailing whitespace.
	Equal                // Both prefix and suffix anchors.
	Boundary             // Exact, with word boundaries on both sides.
)

// Wrap adapts any upstream matcher, using it verbatim on non-Han input.
// The literal upstream match also remains a score/recall floor on Han input.
func Wrap(fallback algo.Algo, mode Mode) algo.Algo {
	return func(cs, norm, forward bool, input *util.Chars, query []rune, withPos bool, slab *util.Slab) (algo.Result, *[]int) {
		native, nativePos := fallback(cs, norm, forward, input, query, withPos, slab)
		if len(query) == 0 || !HasHan(input) {
			return native, nativePos
		}
		// A query with no spelling letters can only have literal edges.
		phonetic := false
		for _, r := range query {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == 'ü' || r == 'Ü' {
				phonetic = true
				break
			}
		}
		if !phonetic {
			return native, nativePos
		}
		result, positions := match(cs, norm, forward, input, query, withPos, mode, true)
		if result.Start >= 0 {
			// max(all paths - 1, literal paths) charges exactly once for
			// using pinyin, without adding a "used pinyin" DP dimension.
			// Native recall covers literal paths, but its score need not:
			// the DP also has Han/Han and Han/Latin boundary bonuses.
			rawScore := result.Score
			result.Score = max(1, result.Score-1)
			if native.Start >= 0 && rawScore > native.Score {
				literal, literalPos := match(cs, norm, forward, input, query, withPos, mode, false)
				if literal.Start >= 0 && (literal.Score > result.Score || literal.Score == result.Score &&
					(literal.Start != result.Start && (literal.Start < result.Start) == forward ||
						literal.Start == result.Start && literal.End != result.End && (literal.End < result.End) == forward)) {
					result, positions = literal, literalPos
				}
			}
		}
		if result.Start >= 0 && (native.Start < 0 || result.Score > native.Score) {
			return result, positions
		}
		return native, nativePos
	}
}

// These functions implement algo.Algo. V1 and V2 retain their distinct native
// behavior; on Han input both use the same optimal phonetic dynamic program.
var (
	FuzzyMatchV2       = Wrap(algo.FuzzyMatchV2, Fuzzy)
	FuzzyMatchV1       = Wrap(algo.FuzzyMatchV1, Fuzzy)
	ExactMatchNaive    = Wrap(algo.ExactMatchNaive, Exact)
	ExactMatchBoundary = Wrap(algo.ExactMatchBoundary, Boundary)
	PrefixMatch        = Wrap(algo.PrefixMatch, Prefix)
	SuffixMatch        = Wrap(algo.SuffixMatch, Suffix)
	EqualMatch         = Wrap(algo.EqualMatch, Equal)
)

// HasHan recognizes Unicode's Han script, including extensions and 〇.
func HasHan(input *util.Chars) bool {
	if input.IsBytes() {
		return false
	}
	for i := 0; i < input.Length(); i++ {
		if unicode.Is(unicode.Han, input.Get(i)) {
			return true
		}
	}
	return false
}

type spelling struct {
	text string
}

// Only per-rune readings are cached: input strings and queries are never kept.
// The cache is bounded by the dictionary's finite codepoint set.
var readings sync.Map // rune -> []spelling

func spellings(r rune) []spelling {
	if cached, ok := readings.Load(r); ok {
		return cached.([]spelling)
	}
	if !unicode.Is(unicode.Han, r) {
		return nil
	}
	if _, ok := data.PinyinDict[int(r)]; !ok {
		return nil
	}
	args := data.Args{Style: data.Normal, Heteronym: true}
	normal := data.SinglePinyin(r, args)
	forms := []spelling{}
	add := func(s string) {
		if s == "" {
			return
		}
		for _, f := range forms {
			if f.text == s {
				return
			}
		}
		forms = append(forms, spelling{s})
	}
	for _, s := range normal {
		// Normal style uses keyboard v for ü. Non-ASCII rare syllables remain
		// searchable literally; they are not silently stripped or invented.
		if strings.IndexFunc(s, func(r rune) bool { return r < 'a' || r > 'z' }) >= 0 {
			continue
		}
		add(s)
		add(s[:1])
		initial := ""
		for _, c := range []string{"zh", "ch", "sh", "b", "p", "m", "f", "d", "t", "n", "l", "g", "k", "h", "j", "q", "x", "r", "z", "c", "s"} {
			if strings.HasPrefix(s, c) {
				initial = c
				break
			}
		}
		if initial != "" {
			add(initial)
		} else {
			// y/w are spelling letters, not phonological initials. Accept
			// the zero-initial final as well as the normal spelling.
			add(zeroFinal(s))
		}
	}
	actual, _ := readings.LoadOrStore(r, forms)
	return actual.([]spelling)
}

func zeroFinal(s string) string {
	switch {
	case strings.HasPrefix(s, "yu"):
		return "v" + s[2:]
	case strings.HasPrefix(s, "yi"):
		return s[1:]
	case strings.HasPrefix(s, "y"):
		return "i" + s[1:]
	case strings.HasPrefix(s, "wu"):
		return s[1:]
	case strings.HasPrefix(s, "w"):
		return "u" + s[1:]
	default:
		return s
	}
}
