package core

import (
	"reflect"
	"strings"
	"testing"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/pinyin"
	"github.com/junegunn/fzf/src/util"
)

func TestSingleCandidateTraceParity(t *testing.T) {
	for _, scheme := range []string{"history", "default", "path"} {
		for _, text := range []string{"👋备忘录/abc", "收货地址\n" + strings.Repeat("synthetic line 你好 0123456789\n", 39999) + "END", "a B----C / abc"} {
			single, _ := NewIndex([]string{text}, Options{Scheme: scheme}, 1)
			selected, _ := NewIndex([]string{text, "\x00"}, Options{Scheme: scheme}, 1)
			for _, q := range []string{"s", "shd", "shdz", "bwl", "abc", "👋b"} {
				for _, contiguous := range []bool{false, true} {
					for _, positions := range []bool{false, true} {
						got, _, err := single.FindMode(q, -1, positions, true, contiguous)
						want, _, we := selected.FindMode(q, -1, positions, true, contiguous)
						// Only exported result fields cross the binding boundary.
						// History's unused private pathname scratch can depend on
						// whether bounds were known during ranking or tracing.
						equal := len(got) == len(want)
						if equal {
							for i, h := range got {
								w := want[i]
								if h.Index != w.Index || h.Score != w.Score || h.Start != w.Start || h.End != w.End || !reflect.DeepEqual(h.Positions, w.Positions) {
									equal = false
									break
								}
							}
						}
						if err != nil || we != nil || !equal {
							t.Fatalf("scheme=%s query=%q contiguous=%t positions=%t got=%+v want=%+v err=%v/%v", scheme, q, contiguous, positions, got, want, err, we)
						}
					}
				}
			}
		}
	}
}

func TestFindModeNativeExact(t *testing.T) {
	texts := []string{"备忘录", "备 忘录", "abc def 苹果", "a B----C / abc", "👋地址"}
	x, err := NewIndex(texts, Options{Scheme: "history"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"bwl", "abcpg", "abc", "dizhi", "dezhi"} {
		for _, positions := range []bool{false, true} {
			hits, _, err := x.FindMode(query, -1, positions, true, true)
			if err != nil {
				t.Fatal(err)
			}
			got := map[int]RankHit{}
			for _, hit := range hits {
				got[hit.Index] = hit
			}
			algo.WithLibraryScheme("history", func() {
				for i, text := range texts {
					chars := util.ToChars([]byte(text))
					r, _ := pinyin.ExactMatchNaive(false, true, true, &chars, []rune(query), true, util.MakeSlab(100*1024, 2048))
					h, ok := got[i]
					if ok != (r.Start >= 0) || ok && (h.Score != r.Score || h.Start != r.Start || h.End != r.End) {
						t.Errorf("query=%q record=%d got=%+v expected=%+v", query, i, h, r)
					}
					if ok && positions && len(h.Positions) == 0 {
						t.Errorf("exact positions missing: query=%q record=%d", query, i)
					}
				}
			})
		}
	}
}
