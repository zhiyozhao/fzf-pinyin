package pinyin

import (
	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
	"slices"
	"unicode"
)

// Frozen dense recurrence before sparse-row optimization, used only as an oracle.
func denseMatchRange(cs, norm, forward bool, input *util.Chars, query []rune, withPos bool, mode Mode, allowPinyin bool, begin, stop int, fixedStart bool) (algo.Result, *[]int) {
	m, n := len(query), input.Length()
	width := (m + 1) * blockBonuses
	size := 2*width + m + 1
	work := workspaces.Get().(*workspace)
	if cap(work.cells) < size {
		work.cells = make([]state, size)
	}
	storage := work.cells[:size]
	defer func() {
		clear(storage)
		if cap(work.cells) > 11*257 {
			work.cells = nil
		}
		workspaces.Put(work)
	}()
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
	limit := 0 // No column beyond the largest consumed query prefix is reachable.
	// At most O(c*m) live states each refer to at most m original positions.
	// Reclaim dead paths in batches; this caps traceback space at O(c*m*m)
	// even for winning intervals containing very long gaps.
	traceLimit := max(65536, size*m*2)
	for i := begin; i < stop; i++ {
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
			(mode != Boundary || i == 0 || !word(input.Get(i-1))) && (!fixedStart || i == begin)
		endOK := (mode != Suffix && mode != Equal || i+1 == right) &&
			(mode != Boundary || i+1 == n || !word(input.Get(i+1)))
		// Score typed letters as a literal consecutive chunk. A new, stronger
		// boundary replaces the block bonus; a skipped source rune resets it.
		nextLimit := limit
		offer := func(j int, from state, k int) {
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
			candidate := state{score: from.score + 16*k + first + max(4, block)*(k-1),
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
			nextLimit = max(nextLimit, col)
			if col == m && endOK && better(candidate, best, forward) {
				best = candidate
			}
		}
		for j := 0; j < min(m, limit+1); j++ {
			if j == 0 && !startOK {
				continue
			}
			consume := func(k int) {
				if j == 0 {
					if startOK {
						offer(j, state{start: i, trace: -1, valid: true}, k)
					}
					return
				}
				for _, from := range prev[j*blockBonuses : (j+1)*blockBonuses] {
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
			for j := 1; j < min(m, limit+1); j++ {
				oldGap := gap[j]
				oldGap.score--
				for _, from := range prev[j*blockBonuses : (j+1)*blockBonuses] {
					if !from.valid {
						continue
					}
					from.score -= 3
					if better(from, oldGap, forward) {
						oldGap = from
					}
				}
				gap[j] = oldGap
			}
		}
		limit = nextLimit
		prev, next = next, prev
		if withPos && len(traces) > traceLimit {
			traces = compactTraces(traces, prev, gap, &best)
		}
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
