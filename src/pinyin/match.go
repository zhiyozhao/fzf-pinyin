package pinyin

import (
	"slices"
	"unicode"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// The DP has one row per original rune, one column per query rune. prev
// ends at the preceding source rune; gap has skipped at least one rune.
// An edge consumes one source rune and one or more query runes. Keeping the
// best score per cell is exact because all future costs are additive and
// depend only on query offset and whether there is a gap.
type state struct {
	score, start, end, trace int
	valid                    bool
}

type traceNode struct{ position, previous int }

func better(a, b state, forward bool) bool {
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

func word(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) }

func match(cs, norm, forward bool, input *util.Chars, query []rune, withPos bool, mode Mode) (algo.Result, *[]int) {
	m, n := len(query), input.Length()
	storage := make([]state, 3*(m+1))
	prev, next, gap := storage[:m+1], storage[m+1:2*(m+1)], storage[2*(m+1):]
	var traces []traceNode
	var best state
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
		forms := spellings(r)
		bonus := algo.BoundaryBonus(input, i)
		// Any Han character may start a Chinese word. This is a scoring
		// bonus only; Boundary mode still requires actual delimiters.
		if i > 0 && unicode.Is(unicode.Han, r) && unicode.Is(unicode.Han, input.Get(i-1)) {
			bonus = max(bonus, 8)
		}
		// Keep the existing Han/Latin or Han/number transition bonus.
		if i > 0 && word(r) && word(input.Get(i-1)) && unicode.Is(unicode.Han, r) != unicode.Is(unicode.Han, input.Get(i-1)) {
			bonus = max(bonus, 8)
		}
		startOK := (mode != Prefix && mode != Equal || i == left) &&
			(mode != Boundary || i == 0 || !word(input.Get(i-1)))
		endOK := (mode != Suffix && mode != Equal || i+1 == right) &&
			(mode != Boundary || i+1 == n || !word(input.Get(i+1)))
		// All edges out of one predecessor share this scoring formula.
		offer := func(j int, from state, k, cost int) {
			candidate := state{score: from.score + 16*k + 4*(k-1) - cost,
				start: from.start, end: i + 1, trace: from.trace, valid: true}
			if j == 0 {
				candidate.score += 2 * bonus
			} else if from.end == i {
				candidate.score += max(8, bonus)
			} else {
				candidate.score += bonus
			}
			col := j + k
			if !better(candidate, next[col], forward) {
				return
			}
			if withPos {
				candidate.trace = len(traces)
				traces = append(traces, traceNode{i, from.trace})
			}
			next[col] = candidate
			if col == m && endOK && better(candidate, best, forward) {
				best = candidate
			}
		}
		for j := 0; j < m; j++ {
			from := prev[j]
			if mode == Fuzzy {
				adjacent, skipped := from, gap[j]
				adjacent.score += max(8, bonus)
				skipped.score += bonus
				if better(skipped, adjacent, forward) {
					from = gap[j]
				}
			}
			if j == 0 && startOK {
				from = state{start: i, trace: -1, valid: true}
			}
			if !from.valid {
				continue
			}
			if literal == query[j] {
				offer(j, from, 1, 0)
			}
			for _, f := range forms {
				k := min(len(f.text), m-j)
				// Only the last query token may use an arbitrary prefix.
				cost := f.cost
				if k < len(f.text) {
					cost = max(4, cost)
				}
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
					offer(j, from, k, cost)
				}
			}
		}
		if mode == Fuzzy {
			for j := 1; j < m; j++ {
				oldGap, newGap := gap[j], prev[j]
				oldGap.score--
				newGap.score -= 3
				if better(newGap, oldGap, forward) {
					gap[j] = newGap
				} else {
					gap[j] = oldGap
				}
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
