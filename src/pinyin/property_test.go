package pinyin

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func TestNonHanSchemeDifferential(t *testing.T) {
	for _, scheme := range testSchemes(t) {
		algo.Init(scheme)
		TestNonHanDifferential(t)
	}
}

func TestNonHanDifferential(t *testing.T) {
	rng := rand.New(rand.NewSource(20261120))
	alphabet := []rune("abcXYZ123_ /:éÉößΩωЖж한글かな🙂💡")
	pairs := []struct {
		name            string
		native, adapted algo.Algo
	}{
		{"v1", algo.FuzzyMatchV1, FuzzyMatchV1}, {"v2", algo.FuzzyMatchV2, FuzzyMatchV2},
		{"exact", algo.ExactMatchNaive, ExactMatchNaive},
		{"boundary", algo.ExactMatchBoundary, ExactMatchBoundary},
		{"prefix", algo.PrefixMatch, PrefixMatch}, {"suffix", algo.SuffixMatch, SuffixMatch},
		{"equal", algo.EqualMatch, EqualMatch},
	}
	for trial := 0; trial < 6000; trial++ {
		text := make([]rune, rng.Intn(90))
		for i := range text {
			text[i] = alphabet[rng.Intn(len(alphabet))]
		}
		q := []rune{}
		for _, r := range text {
			if rng.Intn(7) == 0 {
				q = append(q, r)
			}
		}
		if trial%4 == 0 {
			q = append(q, '中')
		} // Query alone must not trigger the adapter.
		cs, norm, forward := rng.Intn(2) == 0, rng.Intn(2) == 0, rng.Intn(2) == 0
		if !cs {
			q = []rune(strings.ToLower(string(q)))
		}
		c := util.RunesToChars(text)
		var slab *util.Slab
		if trial%2 == 0 {
			slab = util.MakeSlab(32, 32)
		}
		for _, pair := range pairs {
			for _, withPos := range []bool{false, true} {
				want, wp := pair.native(cs, norm, forward, &c, q, withPos, slab)
				got, gp := pair.adapted(cs, norm, forward, &c, q, withPos, slab)
				if got != want || !reflect.DeepEqual(gp, wp) {
					t.Fatalf("%s trial=%d text=%q q=%q: %+v %v != %+v %v", pair.name, trial, text, q, got, gp, want, wp)
				}
			}
		}
	}
	t.Log("84,000 identical result/position comparisons, both directions and folding modes")
}

// This independent fixture records well-established toneless spellings, not
// forms produced by the implementation or its data dependency.
var recallFixture = []struct {
	han   rune
	forms []string
}{
	{'中', []string{"zhong", "zh", "z"}},
	{'国', []string{"guo", "g"}},
	{'安', []string{"an", "a"}},
	{'音', []string{"yin", "in", "y"}},
	{'重', []string{"zhong", "chong", "zh", "ch", "z", "c"}},
	{'行', []string{"xing", "hang", "x", "h"}},
}

func TestRandomRecallAndPrefixClosure(t *testing.T) {
	rng := rand.New(rand.NewSource(70279))
	comparisons := 0
	for trial := 0; trial < 2500; trial++ {
		text, q := "", ""
		for j, count := 0, 1+rng.Intn(8); j < count; j++ {
			fixture := recallFixture[rng.Intn(len(recallFixture))]
			if rng.Intn(3) == 0 {
				text += "🙂-x"
			} // Whole source runes may be skipped.
			text += string(fixture.han)
			if rng.Intn(5) == 0 {
				q += string(fixture.han)
			} else {
				q += fixture.forms[rng.Intn(len(fixture.forms))]
			}
			if rng.Intn(5) == 0 {
				text += "Go"
				q += "go"
			}
		}
		qr := []rune(q)
		for end := 1; end <= len(qr); end++ {
			got, p := run(FuzzyMatchV2, text, string(qr[:end]), end%2 == 0)
			comparisons++
			if got.Start < 0 {
				t.Fatalf("trial %d: lost %q / %q (full=%q)", trial, text, string(qr[:end]), q)
			}
			if p != nil {
				last := -1
				for _, index := range *p {
					if index < 0 || index >= len([]rune(text)) || index == last {
						t.Fatalf("invalid highlight: %v", *p)
					}
					last = index
				}
			}
		}
	}
	t.Logf("%d generated query-prefix recall assertions", comparisons)
}

func TestRandomRankingInvariants(t *testing.T) {
	rng := rand.New(rand.NewSource(8128))
	for trial := 0; trial < 1500; trial++ {
		length := 2 + rng.Intn(12)
		base := "中" + strings.Repeat("国", length-1)
		q := "z" + strings.Repeat("g", length-1)
		contiguous, _ := run(FuzzyMatchV2, base, q, false)
		last := contiguous.Score
		for gap := 1; gap <= 15; gap++ {
			// A separator may introduce a boundary bonus, but must never
			// compensate for breaking adjacent source-character evidence.
			sep := []string{"-", " ", "🙂", "x"}[trial%4]
			gapped := "中" + strings.Repeat(sep, gap) + strings.Repeat("国", length-1)
			r, _ := run(FuzzyMatchV2, gapped, q, false)
			if r.Start < 0 || r.Score > last || r.Score >= contiguous.Score {
				t.Fatalf("gap monotonicity: %q %q: %d > %d; contiguous %d", gapped, q, r.Score, last, contiguous.Score)
			}
			last = r.Score
		}
		// Irrelevant suffixes don't add matching evidence.
		extended, _ := run(FuzzyMatchV2, base+"/not-a-match", q, false)
		if extended.Score != contiguous.Score {
			t.Fatal("unmatched suffix changed score")
		}
	}
}

func FuzzPrefixRecall(f *testing.F) {
	f.Add([]byte{0, 1, 4, 5, 2, 3}, uint8(7))
	f.Add([]byte{3, 3, 0}, uint8(1))
	f.Fuzz(func(t *testing.T, choices []byte, trim uint8) {
		if len(choices) == 0 || len(choices) > 40 {
			return
		}
		text, q := "", ""
		for _, b := range choices {
			x := recallFixture[int(b)%len(recallFixture)]
			text += string(x.han)
			q += x.forms[int(b/6)%len(x.forms)]
		}
		runes := []rune(q)
		prefix := string(runes[:1+int(trim)%len(runes)])
		if r, _ := run(FuzzyMatchV2, text, prefix, false); r.Start < 0 {
			t.Fatalf("lost prefix: %q %q", text, prefix)
		}
	})
}

func TestConcurrentUse(t *testing.T) {
	for i := 0; i < 16; i++ {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			t.Parallel()
			for j := 0; j < 50; j++ {
				if r, _ := run(FuzzyMatchV2, "重庆银行🙂中国", "cqyhzguo", true); r.Start < 0 {
					t.Fatal("concurrent recall")
				}
			}
		})
	}
}
