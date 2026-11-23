// Frozen 76553e1d implementation; private symbols renamed only.
package pinyin

import (
	"slices"
	"unicode"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func legacyWrap(fallback algo.Algo, mode Mode) algo.Algo {
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
		result, positions := legacyMatch(cs, norm, forward, input, query, withPos, mode, true)
		if result.Start >= 0 {
			// max(all paths - 1, literal paths) charges exactly once for
			// using pinyin, without adding a "used pinyin" DP dimension.
			// Native recall covers literal paths, but its score need not:
			// the DP also has Han/Han and Han/Latin boundary bonuses.
			rawScore := result.Score
			result.Score = max(1, result.Score-1)
			if native.Start >= 0 && rawScore > native.Score {
				literal, literalPos := legacyMatch(cs, norm, forward, input, query, withPos, mode, false)
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

var legacyFuzzy = legacyWrap(algo.FuzzyMatchV2, Fuzzy)

// The DP has one row per original rune, one column per query rune. prev
// ends at the preceding source rune; gap has skipped at least one rune.
// An edge consumes one source rune and one or more query runes. Each cell
// keeps separate consecutive-block bonuses: a currently lower score with a
// stronger block may overtake another alignment when more letters arrive.
// All upstream schemes use raw block bonuses 0, 7, 8, 9 or 10.
const legacyBlockBonuses = 5

type legacyState struct {
	score, start, end, trace, block int
	valid                           bool
}

type legacyTraceNode struct{ position, previous int }

func legacyBetter(a, b legacyState, forward bool) bool {
	if !a.valid {
		return false
	}
	if !b.valid || a.score != b.score {
		return !b.valid || a.score > b.score
	}
	if a.start != b.start {
		return (a.start < b.start) == forward
	}
	return (a.end < b.end) == forward
}

func legacyWord(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) }

// allowPinyin=false reuses the same legacyState slots to recover untaxed literal
// paths when the native matcher alone cannot cover their boundary scores.
func legacyMatch(cs, norm, forward bool, input *util.Chars, query []rune, withPos bool, mode Mode, allowPinyin bool) (algo.Result, *[]int) {
	m, n := len(query), input.Length()
	width := (m + 1) * legacyBlockBonuses
	storage := make([]legacyState, 2*width+m+1)
	prev, next, gap := storage[:width], storage[width:2*width], storage[2*width:]
	var traces []legacyTraceNode
	var best legacyState
	left, right := 0, n
	if mode == Prefix || mode == Equal {
		if !unicode.IsSpace(query[0]) {
			left = input.LeadingWhitespaces()
		}
	}
	if mode == Suffix || mode == Equal {
		if !unicode.IsSpace(query[m-1]) {
			right -= input.TrailingWhitespaces()
		}
	}
	for i := 0; i < n; i++ {
		clear(next)
		r := input.Get(i)
		literal := r
		if !cs {
			literal = unicode.ToLower(literal)
		}
		if norm {
			literal = algo.FoldRune(literal)
		}
		var forms []spelling
		if allowPinyin {
			forms = spellings(r)
		}
		bonus := algo.BoundaryBonus(input, i)
		// Any Han character may start a Chinese legacyWord. This is a scoring
		// bonus only; Boundary mode still requires actual delimiters.
		if i > 0 && unicode.Is(unicode.Han, r) && unicode.Is(unicode.Han, input.Get(i-1)) {
			bonus = max(bonus, 8)
		}
		// Keep the existing Han/Latin or Han/number transition bonus.
		if i > 0 && legacyWord(r) && legacyWord(input.Get(i-1)) && unicode.Is(unicode.Han, r) != unicode.Is(unicode.Han, input.Get(i-1)) {
			bonus = max(bonus, 8)
		}
		startOK := (mode != Prefix && mode != Equal || i == left) &&
			(mode != Boundary || i == 0 || !legacyWord(input.Get(i-1)))
		endOK := (mode != Suffix && mode != Equal || i+1 == right) &&
			(mode != Boundary || i+1 == n || !legacyWord(input.Get(i+1)))
		// Score typed letters as a literal consecutive chunk. A new, stronger
		// boundary replaces the block bonus; a skipped source rune resets it.
		offer := func(j int, from legacyState, k int) {
			block, first := bonus, bonus
			if j == 0 {
				first *= 2
			} else if from.end == i {
				block = from.block
				if bonus >= 8 && bonus > block {
					block = bonus
				}
				first = max(4, bonus, block)
			}
			candidate := legacyState{score: from.score + 16*k + first + max(4, block)*(k-1),
				start: from.start, end: i + 1, trace: from.trace, block: block, valid: true}
			col := j + k
			slot := 0
			if block != 0 {
				slot = block - 6
			}
			cell := col*legacyBlockBonuses + slot
			if !legacyBetter(candidate, next[cell], forward) {
				return
			}
			if withPos {
				candidate.trace = len(traces)
				traces = append(traces, legacyTraceNode{i, from.trace})
			}
			next[cell] = candidate
			if col == m && endOK && legacyBetter(candidate, best, forward) {
				best = candidate
			}
		}
		for j := 0; j < m; j++ {
			if j == 0 && !startOK {
				continue
			}
			if j > 0 {
				reachable := mode == Fuzzy && gap[j].valid
				for _, from := range prev[j*legacyBlockBonuses : (j+1)*legacyBlockBonuses] {
					reachable = reachable || from.valid
				}
				if !reachable {
					continue
				}
			}
			consume := func(k int) {
				if j == 0 {
					if startOK {
						offer(j, legacyState{start: i, trace: -1, valid: true}, k)
					}
					return
				}
				for _, from := range prev[j*legacyBlockBonuses : (j+1)*legacyBlockBonuses] {
					if from.valid {
						offer(j, from, k)
					}
				}
				if mode == Fuzzy && gap[j].valid {
					offer(j, gap[j], k)
				}
			}
			if literal == query[j] {
				consume(1)
			}
			for _, f := range forms {
				k := min(len(f.text), m-j)
				// Only the last query token may use an arbitrary prefix.
				ok := true
				for a := 0; a < k; a++ {
					q := query[j+a]
					// Han has no case. Fold only synthetic spelling edges;
					// literal Latin edges still honor the supplied case mode.
					if q >= 'A' && q <= 'Z' {
						q += 'a' - 'A'
					}
					if q == 'Ü' {
						q = 'ü'
					}
					if q == 'ü' {
						q = 'v'
					}
					if q != rune(f.text[a]) {
						ok = false
						break
					}
				}
				if ok {
					consume(k)
				}
			}
		}
		if mode == Fuzzy {
			for j := 1; j < m; j++ {
				oldGap := gap[j]
				oldGap.score--
				for _, from := range prev[j*legacyBlockBonuses : (j+1)*legacyBlockBonuses] {
					from.score -= 3
					if legacyBetter(from, oldGap, forward) {
						oldGap = from
					}
				}
				gap[j] = oldGap
			}
		}
		prev, next = next, prev
	}
	if !best.valid {
		return algo.Result{Start: -1, End: -1}, nil
	}
	result := algo.Result{Start: best.start, End: best.end, Score: max(1, best.score)}
	if !withPos {
		return result, nil
	}
	positions := []int{}
	for node := best.trace; node >= 0; node = traces[node].previous {
		positions = append(positions, traces[node].position)
	}
	slices.Reverse(positions)
	return result, &positions
}
