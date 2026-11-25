// Package core is the value-based library facade used by every foreign binding.
// Query is one literal fuzzy term, not fzf's extended query language.
package core

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/pinyin"
	"github.com/junegunn/fzf/src/util"
)

const Version = "0.1.0"
const CoreRevision = "093ccbbdeae6e446e1e903d1ed80971d428e524b"
const ABIVersion = 1

type Options struct {
	Scheme        string `json:"scheme"`
	CaseSensitive bool   `json:"case_sensitive"`
	Normalize     *bool  `json:"normalize,omitempty"`
	Positions     *bool  `json:"positions,omitempty"`
}

type Request struct {
	Operation  string   `json:"operation"`
	Text       string   `json:"text"`
	Candidates []string `json:"candidates"`
	Query      string   `json:"query"`
	Options    Options  `json:"options"`
}

type Match struct {
	Matched        bool  `json:"matched"`
	Score          int   `json:"score"`
	Start          int   `json:"start"`
	End            int   `json:"end"`
	Positions      []int `json:"positions"`
	UTF8Positions  []int `json:"utf8_positions"`
	UTF16Positions []int `json:"utf16_positions"`
}

type Hit struct {
	Index int `json:"index"`
	Match
}

type Response struct {
	Match   *Match `json:"match,omitempty"`
	Results []Hit  `json:"results,omitempty"`
	Error   string `json:"error,omitempty"`
}

func Execute(req Request) (Response, error) {
	if req.Operation != "match" && req.Operation != "filter" {
		return Response{}, fmt.Errorf("unknown operation: %s", req.Operation)
	}
	if !utf8.ValidString(req.Query) || !utf8.ValidString(req.Text) {
		return Response{}, fmt.Errorf("input must be UTF-8")
	}
	for _, s := range req.Candidates {
		if !utf8.ValidString(s) {
			return Response{}, fmt.Errorf("candidate must be UTF-8")
		}
	}
	if req.Options.Scheme == "" {
		req.Options.Scheme = "default"
	}
	norm := req.Options.Normalize == nil || *req.Options.Normalize
	pos := req.Options.Positions == nil || *req.Options.Positions
	query := req.Query
	if !req.Options.CaseSensitive {
		query = strings.ToLower(query)
	}
	q := []rune(query)
	forward, _ := schemeMatchingPolicy(req.Options.Scheme)
	// Like fzf, accented query literals disable accent folding.
	if norm {
		for _, r := range q {
			if algo.FoldRune(r) != r {
				norm = false
				break
			}
		}
	}
	out := Response{}
	ok := algo.WithLibraryScheme(req.Options.Scheme, func() {
		slab := util.MakeSlab(100*1024, 2048)
		match := func(text string) Match {
			chars := util.ToChars([]byte(text))
			// Upstream's no-position fast path may return an approximate start.
			// The library promises exact, option-independent bounds (path sorting
			// also depends on them), so always trace and only omit exported arrays.
			r, pp := pinyin.FuzzyMatchV2(req.Options.CaseSensitive, norm, forward, &chars, q, true, slab)
			m := Match{Matched: r.Start >= 0, Score: r.Score, Start: r.Start, End: r.End, Positions: []int{}, UTF8Positions: []int{}, UTF16Positions: []int{}}
			if !m.Matched {
				m.Score = 0
				m.Start = -1
				m.End = -1
				return m
			}
			if pos && pp != nil {
				m.Positions = append(m.Positions, (*pp)...)
				sort.Ints(m.Positions)
			}
			if len(m.Positions) == 0 {
				return m
			}
			p, ri, u16 := 0, 0, 0
			for byteIndex, r := range text {
				if p < len(m.Positions) && m.Positions[p] == ri {
					m.UTF8Positions = append(m.UTF8Positions, byteIndex)
					m.UTF16Positions = append(m.UTF16Positions, u16)
					p++
				}
				ri++
				u16++
				if r > 0xffff {
					u16++
				}
			}
			return m
		}
		if req.Operation == "match" {
			m := match(req.Text)
			out.Match = &m
			return
		}
		out.Results = []Hit{}
		type rank struct{ score, pathname, length int }
		ranks := make([]rank, len(req.Candidates))
		for i, text := range req.Candidates {
			m := match(text)
			if !m.Matched {
				continue
			}
			chars := util.ToChars([]byte(text))
			r := rank{score: min(65535, m.Score), pathname: 65535, length: int(chars.TrimLength())}
			if m.End > m.Start {
				runes := []rune(text)
				last := -1
				for j, c := range runes {
					if c == '/' || c == '\\' {
						last = j
					}
				}
				if last <= m.Start {
					r.pathname = min(65535, m.Start-last)
				}
			}
			ranks[i] = r
			out.Results = append(out.Results, Hit{Index: i, Match: m})
		}
		// fzf passes an empty query through without sorting.
		if len(q) == 0 {
			return
		}
		sort.SliceStable(out.Results, func(i, j int) bool {
			a, b := ranks[out.Results[i].Index], ranks[out.Results[j].Index]
			if a.score != b.score {
				return a.score > b.score
			}
			if req.Options.Scheme == "history" {
				return false
			}
			if req.Options.Scheme == "path" && a.pathname != b.pathname {
				return a.pathname < b.pathname
			}
			return a.length < b.length
		})
	})
	if !ok {
		return Response{}, fmt.Errorf("unknown scheme: %s", req.Options.Scheme)
	}
	return out, nil
}
