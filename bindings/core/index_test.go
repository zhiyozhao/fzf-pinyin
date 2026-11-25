package core

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestIndexDifferential(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	parts := []string{"北京", "中国", "fooBar", "a-b", "aab", "😀", "é", "e\u0301", "/", "\\", " ", "绿茶", "呣嗯", "略女巨"}
	texts := []string{"", strings.Repeat("ab ", 2200) + "北京", strings.Repeat("a", 3400) + "b", strings.Repeat("中", 33000) + "国"}
	for i := 0; i < 2100; i++ {
		s := ""
		for j := 0; j < 2+rng.Intn(7); j++ {
			s += parts[rng.Intn(len(parts))]
		}
		texts = append(texts, s)
	}
	for _, scheme := range []string{"default", "path", "history"} {
		for _, query := range []string{"", "b", "bj", "zg", "fb", "ab", "cafe", "lue", "en", "😀", strings.Repeat("a", 40) + "b"} {
			options := Options{Scheme: scheme}
			oracle, err := Execute(Request{Operation: "filter", Candidates: texts, Query: query, Options: options})
			if err != nil {
				t.Fatal(err)
			}
			for _, workers := range []int{1, 4} {
				x, err := NewIndex(texts, options, workers)
				if err != nil {
					t.Fatal(err)
				}
				for _, k := range []int{-1, 0, 1, 7, 50, 99999} {
					for _, positions := range []bool{false, true} {
						got, _, err := x.Find(query, k, positions, false)
						if err != nil {
							t.Fatal(err)
						}
						want := oracle.Results
						if k >= 0 && len(want) > k {
							want = want[:k]
						}
						if len(got) != len(want) {
							t.Fatalf("count %s %q %d: %d != %d", scheme, query, k, len(got), len(want))
						}
						for i, h := range got {
							w := want[i]
							if h.Index != w.Index || h.Score != w.Score || positions && (h.Start != w.Start || h.End != w.End || !equalInts(h.Positions, w.Positions)) {
								t.Fatalf("%s %q workers=%d k=%d pos=%v row %d: %+v != %+v", scheme, query, workers, k, positions, i, h, w)
							}
						}
					}
				}
			}
		}
	}
}
func equalInts(a, b []int) bool { return fmt.Sprint(a) == fmt.Sprint(b) }

func TestWorkerPanicJoins(t *testing.T) {
	var finished [4]bool
	defer func() {
		if recover() == nil {
			t.Error("panic not propagated")
		}
		for _, done := range finished {
			if !done {
				t.Error("worker not joined")
			}
		}
	}()
	parallel(4, func(w int) {
		finished[w] = true
		if w == 2 {
			panic("test")
		}
	})
}

func TestIndexLifecycleAndOptions(t *testing.T) {
	for _, n := range []int{-1, 65} {
		if _, err := NewIndex(nil, Options{}, n); err == nil {
			t.Fatal("workers accepted")
		}
	}
	if _, err := NewIndex([]string{string([]byte{255})}, Options{}, 1); err == nil {
		t.Fatal("invalid UTF-8")
	}
	x, _ := NewIndex([]string{"北京", "a\x00北京\r\n"}, Options{}, 4)
	for _, query := range []string{string([]byte{255})} {
		if _, _, err := x.Find(query, 1, false, false); err == nil {
			t.Fatal("query accepted")
		}
	}
	if _, _, err := x.Find("b", -2, false, false); err == nil {
		t.Fatal("negative limit")
	}
	want, _, _ := x.Find("bj", -1, true, false)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, _, _ := x.Find("bj", -1, true, false)
			if !reflect.DeepEqual(got, want) {
				t.Error("concurrent find")
			}
		}()
	}
	wg.Wait()
}

func TestIndexCaseNormalizationOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(19))
	letters := []rune("ABabſKȺɑᴀéÉＲｒ😀北京绿嗯巨女略\u0301/\\ ")
	texts := []string{"Ʀ", "ᴀ", "café", "cafe", "绿茶", "女儿", "巨人", "略过", "嗯好", "呣啊", "长沙"}
	for i := 0; i < 1100; i++ {
		var s []rune
		for j := 0; j < 12; j++ {
			s = append(s, letters[rng.Intn(len(letters))])
		}
		texts = append(texts, string(s))
	}
	for _, scheme := range []string{"default", "path", "history"} {
		for _, cs := range []bool{false, true} {
			for _, norm := range []bool{false, true} {
				o := Options{Scheme: scheme, CaseSensitive: cs, Normalize: &norm}
				x, _ := NewIndex(texts, o, 4)
				for _, q := range []string{"ab", "AB", "r", "k", "a", "é", "café", "lv", "lü", "lue", "jv", "nv", "en", "mu", "chsh"} {
					oracle, err := Execute(Request{Operation: "filter", Candidates: texts, Query: q, Options: o})
					if err != nil {
						t.Fatal(err)
					}
					got, _, err := x.Find(q, -1, true, false)
					if err != nil {
						t.Fatal(err)
					}
					if len(got) != len(oracle.Results) {
						t.Fatalf("mask false negative %s cs=%v norm=%v q=%s", scheme, cs, norm, q)
					}
					for i, h := range got {
						w := oracle.Results[i]
						if h.Index != w.Index || h.Score != w.Score || h.Start != w.Start || h.End != w.End || !equalInts(h.Positions, w.Positions) {
							t.Fatalf("oracle mismatch %s %s %+v %+v", scheme, q, h, w)
						}
					}
				}
			}
		}
	}
}
