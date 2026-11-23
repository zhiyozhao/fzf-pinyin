package pinyin

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// User-supplied regression examples, not files read from a user's disk.
var machaoPaths = []string{
	"Codes/Agents/Devs/projects/maclip-HANDOFF-1117.md",
	"Codes/Agents/Devs/projects/maclip/CHANGELOG.md",
	"Work/pdp-bot/node_modules/side-channel-map/CHANGELOG.md",
	"Pictures/koh/super/马超-冷晖之枪.png",
	"Pictures/koh/normal/马超-幸存者.jpg",
}

func TestRandomCrossScriptRanking(t *testing.T) {
	rng := rand.New(rand.NewSource(20261124))
	words := []struct{ han, full, initials string }{
		{"北京", "beijing", "bj"}, {"马超", "machao", "mc"},
		{"中国", "zhongguo", "zg"}, {"文档", "wendang", "wd"},
		{"开发工具", "kaifagongju", "kfgj"}, {"音乐", "yinyue", "yy"},
	}
	checks := 0
	for _, scheme := range testSchemes(t) {
		algo.Init(scheme)
		for trial := 0; trial < 1000; trial++ {
			w := words[rng.Intn(len(words))]
			prefix := []string{"", " ", "/", "-", "notes/", "啊"}[rng.Intn(6)]
			for _, q := range []string{w.full, w.initials} {
				tight, _ := run(FuzzyMatchV2, prefix+w.han, q, false)
				// Equivalent literal evidence, including the inherited Han
				// boundary after another Han: '-' has the same boundary 8.
				literalPrefix := prefix
				if prefix == "啊" {
					literalPrefix = "-"
				}
				native, _ := run(algo.FuzzyMatchV2, literalPrefix+q, q, false)
				cost := native.Score - tight.Score
				hanCount := len([]rune(w.han))
				if tight.Start < 0 || cost < 1 || cost > 3*hanCount {
					t.Fatalf("literal parity: %s %q %q score=%d native=%d cost=%d", scheme, prefix+w.han, q, tight.Score, native.Score, cost)
				}
				// Insert nonquery lowercase letters within a literal word.
				// Both candidates have the same initial boundary; no new
				// boundary is manufactured by the insertion.
				cut := 1 + rng.Intn(len(q)-1)
				looseText := literalPrefix + q[:cut] + strings.Repeat("x", 1+rng.Intn(12)) + q[cut:]
				loose, _ := run(algo.FuzzyMatchV2, looseText, q, false)
				if tight.Score < loose.Score {
					t.Fatalf("cross-script gap: %s %q / %q q=%q: %d < %d", scheme, prefix+w.han, looseText, q, tight.Score, loose.Score)
				}
				checks++
			}
		}
	}
	t.Logf("%d randomized full-spelling/initials vs literal parity and gap comparisons", checks)
}

func TestSpellingEvidenceAndBlockAlignment(t *testing.T) {
	for _, tc := range []struct{ text, full, prefix, initials string }{
		{"/北京", "beijing", "beiji", "bj"},
		{"/马超", "machao", "macha", "mc"},
	} {
		full, _ := run(FuzzyMatchV2, tc.text, tc.full, false)
		prefix, _ := run(FuzzyMatchV2, tc.text, tc.prefix, false)
		initials, _ := run(FuzzyMatchV2, tc.text, tc.initials, false)
		if full.Score <= prefix.Score || prefix.Score <= initials.Score {
			t.Fatalf("typed evidence: full=%+v prefix=%+v initials=%+v", full, prefix, initials)
		}
	}
	// Literal and Han edges share the same chunk even across script changes;
	// untyped letters in a final syllable never contribute any evidence.
	for _, tc := range []struct {
		text, q, literal string
		cost             int
	}{
		{"/马超Go", "machaoGo", "/machaoGo", 2},
		{"/马Go超", "maGochao", "/maGochao", 2},
		{"/马超", "macha", "/macha", 5},
		{"/马超", "马chao", "/马chao", 1},
	} {
		got, _ := run(FuzzyMatchV2, tc.text, strings.ToLower(tc.q), false)
		literal, _ := run(algo.FuzzyMatchV2, tc.literal, strings.ToLower(tc.q), false)
		if got.Score != literal.Score-tc.cost {
			t.Fatalf("%q %q: %+v vs %+v cost=%d", tc.text, tc.q, got, literal, tc.cost)
		}
	}
	// Competing source alignments must retain their distinct block bonuses.
	for _, forward := range []bool{true, false} {
		c := util.ToChars([]byte("文马超---------/马超"))
		got, positions := FuzzyMatchV2(false, true, forward, &c, []rune("machao"), true, nil)
		if got.Score != 157 || positions == nil || !reflect.DeepEqual(*positions, []int{13, 14}) {
			t.Fatalf("best chunk forward=%v: %+v %v", forward, got, positions)
		}
	}
}

func TestRandomBoundaryTradeoffAudit(t *testing.T) {
	words := []struct{ han, full, initials string }{
		{"北京", "beijing", "bj"}, {"马超", "machao", "mc"},
		{"中国", "zhongguo", "zg"}, {"音乐", "yinyue", "yy"},
		{"开发工具", "kaifagongju", "kfgj"},
	}
	for _, scheme := range testSchemes(t) {
		algo.Init(scheme)
		rng := rand.New(rand.NewSource(271828))
		wins, nativeTradeoffs, costTradeoffs := 0, 0, 0
		for trial := 0; trial < 3000; trial++ {
			w := words[rng.Intn(len(words))]
			prefix := []string{"", "/", "-", " "}[rng.Intn(4)]
			for _, q := range []string{w.full, w.initials} {
				cut := 1 + rng.Intn(len(q)-1)
				gap := strings.Repeat([]string{"x", "-", "_", "/", " "}[rng.Intn(5)], 1+rng.Intn(8))
				tight, _ := run(FuzzyMatchV2, prefix+w.han, q, false)
				equivalent, _ := run(algo.FuzzyMatchV2, prefix+q, q, false)
				loose, _ := run(algo.FuzzyMatchV2, prefix+q[:cut]+gap+q[cut:], q, false)
				switch {
				case tight.Score >= loose.Score:
					wins++
				case equivalent.Score < loose.Score:
					// A newly introduced stronger boundary also wins in fzf.
					nativeTradeoffs++
				case equivalent.Score-tight.Score >= 1 && equivalent.Score-tight.Score <= 3*len([]rune(w.han)):
					// Preserve the documented preference for literal evidence
					// when only the small conversion cost separates them.
					costTradeoffs++
				default:
					t.Fatalf("unexplained ranking: %s %q q=%q gap=%q tight=%d equivalent=%d loose=%d", scheme, prefix+w.han, q, gap, tight.Score, equivalent.Score, loose.Score)
				}
			}
		}
		t.Logf("%s: 6000 adversarial boundary/gap comparisons: %d Han wins/ties; %d upstream boundary tradeoffs; %d conversion-cost tradeoffs", scheme, wins, nativeTradeoffs, costTradeoffs)
	}
}

func TestMachaoScores(t *testing.T) {
	want := []int{138, 137, 137, 157, 157}
	for i, text := range machaoPaths {
		got, positions := run(FuzzyMatchV2, text, "machao", true)
		t.Logf("%s: score=%d positions=%v", text, got.Score, positions)
		if got.Score != want[i] {
			t.Fatalf("%q: score=%d; want %d", text, got.Score, want[i])
		}
	}
	chinese, _ := run(FuzzyMatchV2, machaoPaths[3], "machao", false)
	loose, _ := run(algo.FuzzyMatchV2, machaoPaths[0], "machao", false)
	literal, _ := run(algo.FuzzyMatchV2, "/machao", "machao", false)
	// Six typed letters are six consecutive observations; there are two
	// full-spelling costs, rather than a discount for four internal letters.
	if chinese.Score != literal.Score-2 || chinese.Score <= loose.Score {
		t.Fatalf("full spelling=%d; literal=%d; gapped literal=%d", chinese.Score, literal.Score, loose.Score)
	}
}

func TestExplicitScoringTradeoffs(t *testing.T) {
	for _, tc := range []struct {
		han, literal, query    string
		hanScore, literalScore int
	}{
		// Real new word boundaries can outweigh small abbreviation costs.
		{"-北京", "-b-j", "bj", 52, 53},
		{"-音乐", "-y-y", "yy", 50, 53},
		// Default's line-start bonus is inherited by all three letters of
		// bei: this is also a literal fzf boundary/gap tradeoff.
		{"文北京", "北文田田文文京", "beijing", 174, 175},
	} {
		han, _ := run(FuzzyMatchV2, tc.han, tc.query, false)
		literal, _ := run(FuzzyMatchV2, tc.literal, tc.query, false)
		if han.Score != tc.hanScore || literal.Score != tc.literalScore {
			t.Fatalf("%q / %q query=%q: %d / %d; want %d / %d", tc.han, tc.literal, tc.query, han.Score, literal.Score, tc.hanScore, tc.literalScore)
		}
	}
}

func TestUnmatchedHanDoesNotInflateLiteralChunk(t *testing.T) {
	for _, fn := range []struct{ native, adapted algo.Algo }{
		{algo.FuzzyMatchV1, FuzzyMatchV1}, {algo.FuzzyMatchV2, FuzzyMatchV2},
	} {
		for _, text := range []string{"xabc /田", "file/xabc---田"} {
			want, wp := run(fn.native, text, "abc", true)
			got, gp := run(fn.adapted, text, "abc", true)
			if got != want || !reflect.DeepEqual(gp, wp) {
				t.Fatalf("unmatched Han inflated literal match %q: %+v %v vs %+v %v", text, got, gp, want, wp)
			}
		}
	}
}
