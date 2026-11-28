package core

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/pinyin"
	"github.com/junegunn/fzf/src/util"
)

// Index owns immutable candidates and serializes queries to reuse worker scratch.
// Schemes are still serialized across indexes by WithLibraryScheme.
type Index struct {
	mu      sync.Mutex
	options Options
	items   []indexedText
	workers []indexWorker
}
type indexedText struct {
	text              string
	chars             util.Chars
	length, lastSlash int
	letters           uint32
}
type indexWorker struct {
	slab *util.Slab
	rows []RankHit
}
type RankHit struct {
	Index, Score int
	// Start/End are -1 unless exact bounds were requested (or needed to rank).
	Start, End       int
	Positions        []int
	pathname, length int
}
type Timings struct{ Prepare, Scan, Select, Trace float64 }

func NewIndex(texts []string, options Options, workers int) (*Index, error) {
	if options.Scheme == "" {
		options.Scheme = "default"
	}
	if options.Scheme != "default" && options.Scheme != "path" && options.Scheme != "history" {
		return nil, fmt.Errorf("unknown scheme: %s", options.Scheme)
	}
	if workers < 0 || workers > 64 {
		return nil, fmt.Errorf("workers must be 0..64")
	}
	if workers == 0 {
		workers = min(4, runtime.GOMAXPROCS(0))
	}
	if len(texts) < 1024 {
		workers = 1
	}
	x := &Index{options: options, items: make([]indexedText, len(texts)), workers: make([]indexWorker, workers)}
	// Snapshot pointer-valued options; callers may mutate their own bools later.
	norm := options.Normalize == nil || *options.Normalize
	x.options.Normalize = &norm
	for i, text := range texts {
		if !utf8.ValidString(text) {
			return nil, fmt.Errorf("candidate must be UTF-8")
		}
		text = strings.Clone(text)
		chars := util.ToChars([]byte(text))
		last := -1
		for j := 0; j < chars.Length(); j++ {
			if c := chars.Get(j); c == '/' || c == '\\' {
				last = j
			}
		}
		x.items[i] = indexedText{text: text, chars: chars, length: int(chars.TrimLength()), lastSlash: last, letters: pinyin.LibraryLetterMask(&chars)}
	}
	for i := range x.workers {
		x.workers[i].slab = util.MakeSlab(100*1024, 2048)
	}
	return x, nil
}

// Find takes -1 for unlimited results. Exact requests bounds even without
// positions, for the legacy convenience API. All positions are rune indexes.
func (x *Index) Find(query string, limit int, positions, exact bool) ([]RankHit, Timings, error) {
	return x.FindMode(query, limit, positions, exact, false)
}

// FindMode selects the native contiguous matcher without changing fuzzy scoring.
// exactBounds is independent of matching mode (ABI 2's original flag bit 1).
func (x *Index) FindMode(query string, limit int, positions, exactBounds, contiguous bool) ([]RankHit, Timings, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	var times Timings
	if limit < -1 {
		return nil, times, fmt.Errorf("limit must be nonnegative or -1 (unlimited)")
	}
	if !utf8.ValidString(query) {
		return nil, times, fmt.Errorf("query must be UTF-8")
	}
	start := time.Now()
	if !x.options.CaseSensitive {
		query = strings.ToLower(query)
	}
	q := []rune(query)
	match := pinyin.FuzzyMatchV2
	if contiguous {
		match = pinyin.ExactMatchNaive
	}
	forward, rankPositions := schemeMatchingPolicy(x.options.Scheme)
	// A one-candidate alignment request has nothing to select. Materialize it
	// during the scan instead of rerunning the complete matcher for trace.
	singleTrace := len(x.items) == 1 && (positions || exactBounds)
	traceDuringScan := rankPositions || singleTrace
	setPositions := func(h *RankHit, r algo.Result, pp *[]int) {
		if !positions {
			return
		}
		if pp != nil {
			h.Positions = slices.Clone(*pp)
			slices.Sort(h.Positions)
		} else if contiguous {
			for p := r.Start; p < r.End; p++ {
				h.Positions = append(h.Positions, p)
			}
		}
	}
	var required uint32
	for _, r := range q {
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		if r >= 'a' && r <= 'z' {
			required |= 1 << (r - 'a')
		}
	}
	norm := *x.options.Normalize
	if norm {
		for _, r := range q {
			if algo.FoldRune(r) != r {
				norm = false
				break
			}
		}
	}
	times.Prepare = elapsed(start)
	better := func(a, b RankHit) bool {
		if len(q) != 0 {
			if sa, sb := min(65535, a.Score), min(65535, b.Score); sa != sb {
				return sa > sb
			}
			if x.options.Scheme != "history" {
				if x.options.Scheme == "path" && a.pathname != b.pathname {
					return a.pathname < b.pathname
				}
				if a.length != b.length {
					return a.length < b.length
				}
			}
		}
		return a.Index < b.Index
	}
	var result []RankHit
	algo.WithLibraryScheme(x.options.Scheme, func() {
		if limit == 0 {
			result = []RankHit{}
			return
		}
		start = time.Now()
		nw := len(x.workers)
		if len(x.items) < 1024 {
			nw = 1
		}
		// Each worker has a worst-first bounded heap, or a full compact rank list.
		run := func(w int) {
			worker := &x.workers[w]
			clear(worker.rows)
			worker.rows = worker.rows[:0]
			for i := len(x.items) * w / nw; i < len(x.items)*(w+1)/nw; i++ {
				item := &x.items[i]
				if item.letters&required != required {
					continue
				}
				trace := traceDuringScan
				r, pp := match(x.options.CaseSensitive, norm, forward, &item.chars, q, trace, worker.slab)
				if r.Start < 0 {
					continue
				}
				h := RankHit{Index: i, Score: r.Score, Start: -1, End: -1, pathname: 65535, length: item.length}
				if trace {
					h.Start, h.End = r.Start, r.End
					if singleTrace {
						setPositions(&h, r, pp)
					}
				}
				if r.End > r.Start && item.lastSlash <= r.Start {
					h.pathname = min(65535, r.Start-item.lastSlash)
				}
				if limit == -1 {
					worker.rows = append(worker.rows, h)
				} else {
					worker.rows = keepBest(worker.rows, h, limit, better)
				}
			}
		}
		parallel(nw, run)
		times.Scan = elapsed(start)
		start = time.Now()
		for w := 0; w < nw; w++ {
			if limit == -1 {
				result = append(result, x.workers[w].rows...)
			} else {
				for _, h := range x.workers[w].rows {
					result = keepBest(result, h, limit, better)
				}
			}
		}
		slices.SortFunc(result, func(a, b RankHit) int {
			if better(a, b) {
				return -1
			}
			if better(b, a) {
				return 1
			}
			return 0
		})
		times.Select = elapsed(start)
		if (positions || exactBounds) && !singleTrace {
			start = time.Now()
			// Returned hits are usually small. Parallelize full exact materialization.
			align := func(w int) {
				for j := len(result) * w / nw; j < len(result)*(w+1)/nw; j++ {
					h := &result[j]
					item := &x.items[h.Index]
					r, pp := match(x.options.CaseSensitive, norm, forward, &item.chars, q, true, x.workers[w].slab)
					h.Start, h.End = r.Start, r.End
					setPositions(h, r, pp)
				}
			}
			parallel(nw, align)
			times.Trace = elapsed(start)
		}
	})
	if result == nil {
		result = []RankHit{}
	}
	return result, times, nil
}

// Join every worker before propagating a panic to the C ABI recovery boundary.
// A panicking batch must never release the global scheme lock while siblings
// are still reading scheme globals.
func parallel(n int, fn func(int)) {
	if n == 1 {
		fn(0)
		return
	}
	panics := make([]any, n)
	var wg sync.WaitGroup
	for w := 0; w < n; w++ {
		wg.Add(1)
		go func(w int) { defer wg.Done(); defer func() { panics[w] = recover() }(); fn(w) }(w)
	}
	wg.Wait()
	for _, p := range panics {
		if p != nil {
			panic(p)
		}
	}
}

func elapsed(start time.Time) float64 { return float64(time.Since(start)) / float64(time.Millisecond) }

// keepBest maintains a heap whose root is the worst retained row. Tie keys
// participate in admission as well as final sorting, including original index.
func keepBest(rows []RankHit, h RankHit, k int, better func(RankHit, RankHit) bool) []RankHit {
	if len(rows) < k {
		rows = append(rows, h)
		for i := len(rows) - 1; i > 0; {
			p := (i - 1) / 2
			if !better(rows[p], rows[i]) {
				break
			}
			rows[p], rows[i] = rows[i], rows[p]
			i = p
		}
		return rows
	}
	if !better(h, rows[0]) {
		return rows
	}
	rows[0] = h
	for i := 0; ; {
		c := 2*i + 1
		if c >= len(rows) {
			break
		}
		if c+1 < len(rows) && better(rows[c], rows[c+1]) {
			c++
		}
		if !better(rows[i], rows[c]) {
			break
		}
		rows[i], rows[c] = rows[c], rows[i]
		i = c
	}
	return rows
}
