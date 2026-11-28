package core

import (
	"slices"
	"testing"
)

func TestLatinFallbackPositionsAscending(t *testing.T) {
	for _, scheme := range []string{"default", "path", "history"} {
		for _, count := range []int{1, 2, 1100} {
			texts := make([]string, count)
			for i := range texts {
				texts[i] = "😀ab/ab abb"
			}
			for _, workers := range []int{1, 4} {
				x, err := NewIndex(texts, Options{Scheme: scheme}, workers)
				if err != nil {
					t.Fatal(err)
				}
				for repeat := 0; repeat < 3; repeat++ {
					for _, exact := range []bool{false, true} {
						for _, bounds := range []bool{false, true} {
							for _, limit := range []int{-1, 1, 50} {
								hits, _, err := x.FindMode("ab", limit, true, bounds, exact)
								if err != nil || len(hits) == 0 {
									t.Fatalf("%s: %v", scheme, err)
								}
								for _, h := range hits {
									if len(h.Positions) != 2 || !slices.IsSorted(h.Positions) {
										t.Fatalf("%s count=%d workers=%d exact=%t bounds=%t: %+v", scheme, count, workers, exact, bounds, h)
									}
								}
							}
						}
					}
				}
			}
		}
		for _, operation := range []string{"match", "filter"} {
			r, err := Execute(Request{Operation: operation, Text: "😀ab", Candidates: []string{"😀ab"}, Query: "ab", Options: Options{Scheme: scheme}})
			if err != nil {
				t.Fatal(err)
			}
			m := r.Match
			if operation == "filter" {
				m = &r.Results[0].Match
			}
			if !slices.Equal(m.Positions, []int{1, 2}) || !slices.Equal(m.UTF8Positions, []int{4, 5}) || !slices.Equal(m.UTF16Positions, []int{2, 3}) {
				t.Fatalf("%s %s: %+v", scheme, operation, m)
			}
		}
	}
}
