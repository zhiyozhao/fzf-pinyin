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
		// A literal match at the scheme's score ceiling cannot be beaten:
		// every query rune contributes 16, the first bonus is doubled, all
		// remaining bonuses are at most maxBonus, and gaps only subtract.
		// Pinyin additionally loses one point. Avoid all DP passes in this
		// common case (e.g. a word at a boundary in a very long document).
		if native.Start >= 0 {
			if native.Score >= scoreCeiling(len(query)) {
				return native, nativePos
			}
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

// Observe the active scheme's whitespace and delimiter bonuses without
// depending on or mutating upstream algo's private scheme state.
var scoreCeilingProbe = util.ToChars([]byte(" a/a"))

func scoreCeiling(queryRunes int) int {
	maxBonus := max(8, algo.BoundaryBonus(&scoreCeilingProbe, 1), algo.BoundaryBonus(&scoreCeilingProbe, 3))
	return 16*queryRunes + maxBonus*(queryRunes+1)
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

// Keyboard spellings, rather than phonological finals. Restrict u/v aliases
// to these legal syllables: lu/nu/gu and other u readings stay distinct.
var syllableAliases = map[string]string{
	"lve": "lue", "nve": "nue",
	"ju": "jv", "jue": "jve", "juan": "jvan", "jun": "jvn",
	"qu": "qv", "que": "qve", "quan": "qvan", "qun": "qvn",
	"xu": "xv", "xue": "xve", "xuan": "xvan", "xun": "xvn",
	"yu": "yv", "yue": "yve", "yuan": "yvan", "yun": "yvn",
}

// Character-specific IME codes missing from go-pinyin Normal. Do not turn
// every standalone n/ng/m or non-ASCII dictionary reading into en/eng/mu/e.
// Evidence: Sogou FAQ 2003929 (嗯); rime-luna-pinyin 84685a7 (呣/呒/嘸);
// rime-ice 8105.dict.yaml at 927e1cf (欸/呒). These are coding facts only;
// no input-method dictionary or spelling-algebra implementation is copied.
var runeAdditions = map[rune][]string{
	'嗯': {"en"},
	'呣': {"mu"},
	'欸': {"e"},
	'呒': {"fu", "mu"},
	'嘸': {"mu"},
}

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
		// Normal already uses keyboard v for ü. Residual non-ASCII readings
		// need a per-character decision, not generic accent stripping.
		if strings.IndexFunc(s, func(r rune) bool { return r < 'a' || r > 'z' }) >= 0 {
			continue
		}
		switch s {
		case "bong", "len", "hng":
			// Unverified IME full codes. Other readings remain available.
			continue
		}
		add(s)
		add(syllableAliases[s])
	}
	for _, s := range runeAdditions[r] {
		add(s)
	}
	// Derive abbreviations from the final full-code set, including additions
	// (en -> e for 嗯). y/w stay intact; there is no zero-initial-final alias.
	full := len(forms)
	for i := 0; i < full; i++ {
		s := forms[i].text
		add(s[:1])
		if strings.HasPrefix(s, "zh") || strings.HasPrefix(s, "ch") || strings.HasPrefix(s, "sh") {
			add(s[:2])
		}
	}
	actual, _ := readings.LoadOrStore(r, forms)
	return actual.([]spelling)
}
