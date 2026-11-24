package pinyin

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func init() { algo.Init("default") }

func run(fn algo.Algo, text, query string, positions bool) (algo.Result, *[]int) {
	c := util.ToChars([]byte(text))
	return fn(false, true, true, &c, []rune(query), positions, nil)
}

func TestSpellings(t *testing.T) {
	for _, tc := range []struct{ text, query string }{
		{"中国", "zg"}, {"中国", "zhg"}, {"中国", "zhongguo"},
		{"中国", "zhongg"}, {"中国", "zguo"}, {"中国", "zhonggu"},
		{"中国", "中guo"}, {"中国", "zhong国"},
		{"方案", "fan"}, {"方案", "fangan"}, {"方案", "fa"},
		{"西安", "xian"}, {"银行", "yinhang"}, {"银行", "yinxing"},
		{"重庆", "chongqing"}, {"重庆", "zhongqing"},
		{"音乐", "yy"}, {"音乐", "yinyue"},
		{"女绿", "nvlv"}, {"女绿", "nülü"},
		{"src/中文测试.go", "zwcs.go"}, {"src/中文测试.go", "src/zhongwencs.go"},
		{"𠀀", "he"}, {"〇", "ling"},
	} {
		t.Run(tc.text+"/"+tc.query, func(t *testing.T) {
			r, pos := run(FuzzyMatchV2, tc.text, tc.query, true)
			if r.Start < 0 || pos == nil || len(*pos) == 0 {
				t.Fatalf("not recalled: %q %q: %+v %v", tc.text, tc.query, r, pos)
			}
			plain, _ := run(FuzzyMatchV2, tc.text, tc.query, false)
			if plain != r {
				t.Fatalf("withPos changes score or bounds: %+v != %+v", plain, r)
			}
		})
	}
}

func TestNoIntraSyllableFuzz(t *testing.T) {
	for _, q := range []string{"zg", "h", "hong", "zog", "zong", "zhofoo"} {
		r, _ := run(FuzzyMatchV2, "中", q, true)
		if r.Start >= 0 {
			t.Fatalf("invented spelling %q: %+v", q, r)
		}
	}
	if r, _ := run(FuzzyMatchV2, "中国", "zhguo", true); r.Start < 0 {
		t.Fatal("complete shengmu may precede next syllable")
	}
	if r, _ := run(FuzzyMatchV2, "中国", "zhoguo", true); r.Start >= 0 {
		t.Fatal("a non-final syllable cannot use arbitrary prefix zho")
	}
}

func TestOriginalRunePositions(t *testing.T) {
	r, p := run(FuzzyMatchV2, "🙂中x国", "zhonggu", true)
	if p == nil {
		t.Fatal("no positions")
	}
	sort.Ints(*p)
	if r.Start != 1 || r.End != 4 || !reflect.DeepEqual(*p, []int{1, 3}) {
		t.Fatalf("positions are original rune offsets: %+v %v", r, *p)
	}
}

func TestAnchorsAndExact(t *testing.T) {
	for _, tc := range []struct {
		fn      algo.Algo
		text, q string
		match   bool
	}{
		{ExactMatchNaive, "中x国", "zg", false},
		{ExactMatchNaive, "中国", "zhonggu", true},
		{PrefixMatch, " 中国 foo", "zg", true},
		{PrefixMatch, "x中国", "zg", false},
		{SuffixMatch, "x中国 ", "zg", true},
		{SuffixMatch, "中国x", "zg", false},
		{EqualMatch, " 中国 ", "zg", true},
		{EqualMatch, "中国x", "zg", false},
		{ExactMatchBoundary, "x中国x", "zg", false},
		{ExactMatchBoundary, "x 中国 x", "zg", true},
	} {
		r, _ := run(tc.fn, tc.text, tc.q, true)
		if (r.Start >= 0) != tc.match {
			t.Errorf("%q %q: %+v; want match=%v", tc.text, tc.q, r, tc.match)
		}
	}
}

func TestCase(t *testing.T) {
	c := util.ToChars([]byte("中国Go"))
	if r, _ := FuzzyMatchV2(true, false, true, &c, []rune("ZG"), false, nil); r.Start < 0 {
		t.Fatal("Han spelling edges have no case")
	}
	if r, _ := FuzzyMatchV2(true, false, true, &c, []rune("zgGo"), false, nil); r.Start < 0 {
		t.Fatal("mixed literal case must be respected")
	}
	if r, _ := FuzzyMatchV2(true, false, true, &c, []rune("ZGgo"), false, nil); r.Start >= 0 {
		t.Fatal("folding phonetic letters must not fold literal Latin")
	}
}

func TestScoreProperties(t *testing.T) {
	pairs := [][3]string{
		{"中国", "中--国", "zg"},
		{"中国", "中--------国", "zhongguo"},
		{"中国", "x中国", "zg"},
		{"zg", "中国", "zg"},
		{"zhongguo", "中国", "zhongguo"},
		{"中国", "中国任意后缀", "zhongguo"},
	}
	for i, p := range pairs {
		a, _ := run(FuzzyMatchV2, p[0], p[2], false)
		b, _ := run(FuzzyMatchV2, p[1], p[2], false)
		if a.Start < 0 || b.Start < 0 {
			t.Fatalf("bad property setup: %v", p)
		}
		if i == len(pairs)-1 {
			if a.Score != b.Score {
				t.Errorf("unmatched suffix changes score: %v: %d != %d", p, a.Score, b.Score)
			}
		} else if a.Score <= b.Score {
			t.Errorf("ranking invariant: %v: %d <= %d", p, a.Score, b.Score)
		}
	}
	// No score is created by untyped letters in a long syllable.
	for _, q := range []string{"z", "zh", "zho", "zhon", "zhong"} {
		r, _ := run(FuzzyMatchV2, "中", q, false)
		if r.Score > 36*len([]rune(q))+20 {
			t.Fatalf("score inflation: %q %d", q, r.Score)
		}
	}
}

func TestEveryPrefix(t *testing.T) {
	for _, text := range []string{"中国", "重庆银行", "方案", "音乐", "女绿", "中Go国"} {
		queries := map[string][]string{
			"中国":   {"zhongguo", "zhongg", "zhguo", "zguo", "中guo"},
			"重庆银行": {"chongqingyinhang", "zhongqyh", "cqyinxing"},
			"方案":   {"fangan", "fan"}, "音乐": {"yinyue", "yy"},
			"女绿": {"nvlv", "nülü"}, "中Go国": {"zhonggoguo"},
		}
		for _, q := range queries[text] {
			runes := []rune(q)
			for n := 1; n <= len(runes); n++ {
				r, _ := run(FuzzyMatchV2, text, string(runes[:n]), false)
				if r.Start < 0 {
					t.Fatalf("prefix lost: %q %q (%q)", text, q, string(runes[:n]))
				}
			}
		}
	}
}

func TestLiteralFallback(t *testing.T) {
	for _, text := range []string{"中文foobar", "中文🙂école", "𰻞abc", "完全中文"} {
		for _, q := range []string{"foobar", "e", "中文", "𰻞", ""} {
			orig, _ := run(algo.FuzzyMatchV2, text, strings.ToLower(q), false)
			got, _ := run(FuzzyMatchV2, text, strings.ToLower(q), false)
			if orig.Start >= 0 && (got.Start < 0 || got.Score < orig.Score) {
				t.Fatalf("literal recall/score regressed: %q %q: %+v %+v", text, q, orig, got)
			}
		}
	}
}
