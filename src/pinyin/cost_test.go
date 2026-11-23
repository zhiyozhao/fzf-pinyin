package pinyin

import (
	"reflect"
	"testing"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func TestLinePenaltyScores(t *testing.T) {
	for _, scheme := range testSchemes(t) {
		algo.Init(scheme)
		// '-' gives B=8 in all schemes: 32+2*8+8=56.
		// One skipped '-' costs 3; pinyin costs 1 for the entire path.
		// '/' gives B=9 in default/path and B=8 in history:
		// machao has six letters, hence 96+7*B before the line penalty.
		want := map[string][]int{
			"default": {56, 55, 53, 55, 53, 159, 158},
			"path":    {56, 55, 53, 55, 53, 159, 158},
			"history": {56, 55, 53, 55, 53, 152, 151},
		}[scheme]
		for i, tc := range []struct{ text, query string }{
			{"-bj", "bj"}, {"-北京", "bj"}, {"-b-j", "bj"},
			{"-音乐", "yy"}, {"-y-y", "yy"},
			{"/machao", "machao"}, {"/马超", "machao"},
		} {
			for _, withPos := range []bool{false, true} {
				got, _ := run(FuzzyMatchV2, tc.text, tc.query, withPos)
				if got.Score != want[i] {
					t.Fatalf("%s %q / %q: %+v, want %d", scheme, tc.text, tc.query, got, want[i])
				}
			}
		}
	}
}

func TestEverySpellingPaysOnce(t *testing.T) {
	for _, scheme := range testSchemes(t) {
		algo.Init(scheme)
		for _, tc := range []struct{ text, query, literal string }{
			{"/中", "zhong", "/zhong"}, // full
			{"/中", "zh", "/zh"},       // consonant initial
			{"/音", "y", "/y"},         // first letter
			{"/音", "in", "/in"},       // zero-initial final
			{"/中", "zho", "/zho"},     // unfinished final syllable
			{"/中国", "zhongguo", "/zhongguo"},
			{"/中国", "zg", "/zg"},
			{"/先", "xian", "/xian"},
			{"/西安", "xian", "/xian"}, // segmentation count is free
			{"/中国", "中gu", "/中gu"},   // literal Han plus pinyin prefix
			{"/中Go国", "中goguo", "/中Goguo"},
		} {
			got, _ := run(FuzzyMatchV2, tc.text, tc.query, false)
			literal, _ := run(FuzzyMatchV2, tc.literal, tc.query, false)
			if got.Score != literal.Score-1 {
				t.Fatalf("%s %q / %q: %d, literal %d; want exactly one point", scheme, tc.text, tc.query, got.Score, literal.Score)
			}
		}
	}
}

func TestLiteralHanPathsAreUntaxed(t *testing.T) {
	for _, scheme := range testSchemes(t) {
		algo.Init(scheme)
		// Han transitions get B=8 in the existing DP, including literal
		// edges. Native fzf has a smaller boundary here, so simply taking
		// max(native, DP-1) would incorrectly tax these literal matches.
		for _, tc := range []struct {
			text, query       string
			start, end, score int
		}{
			{"田abc", "abc", 1, 4, 48 + 2*8 + 2*8},
			{"国中a", "中a", 1, 3, 32 + 2*8 + 8},
			{"国中a文", "中a文", 1, 4, 48 + 2*8 + 2*8},
		} {
			for _, withPos := range []bool{false, true} {
				got, _ := run(FuzzyMatchV2, tc.text, tc.query, withPos)
				if got.Start != tc.start || got.End != tc.end || got.Score != tc.score {
					t.Fatalf("%s literal %q / %q: %+v; want [%d,%d) score=%d", scheme, tc.text, tc.query, got, tc.start, tc.end, tc.score)
				}
			}
		}
		// No spelling letters: keep the upstream result, including positions.
		for _, q := range []string{"中国", "国", "中🙂"} {
			want, wp := run(algo.FuzzyMatchV2, "田中国中🙂", q, true)
			got, gp := run(FuzzyMatchV2, "田中国中🙂", q, true)
			if got != want || !reflect.DeepEqual(gp, wp) {
				t.Fatalf("literal Han query %q: %+v %v != %+v %v", q, got, gp, want, wp)
			}
		}
	}
}

func TestPenaltyDoesNotChangeNativeTie(t *testing.T) {
	// The DP literal path and taxed pinyin can tie. The native result must
	// still win a public tie, with its own position order/nil conventions.
	for _, forward := range []bool{false, true} {
		// Literal: 32+16+8=56. Pinyin: 32+20+8-3-1=56.
		c := util.ToChars([]byte("-bj 北-京"))
		want, wp := algo.FuzzyMatchV2(false, true, forward, &c, []rune("bj"), true, nil)
		got, gp := FuzzyMatchV2(false, true, forward, &c, []rune("bj"), true, nil)
		if got != want || !reflect.DeepEqual(gp, wp) {
			t.Fatalf("forward=%v: %+v %v != %+v %v", forward, got, gp, want, wp)
		}
	}
}

func TestLiteralRecoveryCompetesWithPinyin(t *testing.T) {
	for _, fn := range []algo.Algo{FuzzyMatchV1, FuzzyMatchV2, ExactMatchNaive, SuffixMatch} {
		for _, forward := range []bool{false, true} {
			// Both DP paths have raw score 16+2*8=32. In forward
			// mode the raw DP prefers 安; only that path pays one point.
			// The literal a after Han also has B=8, above native's B=0.
			c := util.ToChars([]byte("-安a"))
			got, gp := fn(false, true, forward, &c, []rune("a"), true, nil)
			if got != (algo.Result{Start: 2, End: 3, Score: 32}) || gp == nil || !reflect.DeepEqual(*gp, []int{2}) {
				t.Fatalf("forward=%v: %+v %v; want untaxed literal a at index 2, score 32", forward, got, gp)
			}
		}
	}
}
