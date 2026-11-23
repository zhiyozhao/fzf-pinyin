package pinyin

import (
	"slices"
	"unicode"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// The DP has one row per original rune, one column per query rune. prev
// ends at the preceding source rune; gap has skipped at least one rune.
// An edge consumes one source rune and one or more query runes. Each cell
// keeps separate consecutive-block bonuses: a currently lower score with a
// stronger block may overtake another alignment when more letters arrive.
// All upstream schemes use raw block bonuses 0, 7, 8, 9 or 10.
const blockBonuses = 5

type state struct {
	score, start, end, trace, block int
	valid                           bool
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
	width := (m + 1) * blockBonuses
	storage := make([]state, 2*width+m+1)
	prev, next, gap := storage[:width], storage[width:2*width], storage[2*width:]
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
		// Score typed letters as a literal consecutive chunk. A new, stronger
		// boundary replaces the block bonus; a skipped source rune resets it.
		offer := func(j int, from state, k, cost int) {
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
			candidate := state{score: from.score + 16*k + first + max(4, block)*(k-1) - cost,
				start: from.start, end: i + 1, trace: from.trace, block: block, valid: true}
			col := j + k
			slot := 0
			if block != 0 {
				slot = block - 6
			}
			cell := col*blockBonuses + slot
			if !better(candidate, next[cell], forward) {
				return
			}
			if withPos {
				candidate.trace = len(traces)
				traces = append(traces, traceNode{i, from.trace})
			}
			next[cell] = candidate
			if col == m && endOK && better(candidate, best, forward) {
				best = candidate
			}
		}
		for j := 0; j < m; j++ {
			if j == 0 && !startOK {
				continue
			}
			if j > 0 {
				reachable := mode == Fuzzy && gap[j].valid
				for _, from := range prev[j*blockBonuses : (j+1)*blockBonuses] {
					reachable = reachable || from.valid
				}
				if !reachable {
					continue
				}
			}
			consume := func(k, cost int) {
				if j == 0 {
					if startOK {
						offer(j, state{start: i, trace: -1, valid: true}, k, cost)
					}
					return
				}
				for _, from := range prev[j*blockBonuses : (j+1)*blockBonuses] {
					if from.valid {
						offer(j, from, k, cost)
					}
				}
				if mode == Fuzzy && gap[j].valid {
					offer(j, gap[j], k, cost)
				}
			}
			if literal == query[j] {
				consume(1, 0)
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
					consume(k, cost)
				}
			}
		}
		if mode == Fuzzy {
			for j := 1; j < m; j++ {
				oldGap := gap[j]
				oldGap.score--
				for _, from := range prev[j*blockBonuses : (j+1)*blockBonuses] {
					from.score -= 3
					if better(from, oldGap, forward) {
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
