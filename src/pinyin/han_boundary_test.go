package pinyin

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func TestHanBoundaryXdaoScores(t *testing.T) {
	for _, tc := range []struct {
		text      string
		score     int
		positions []int
	}{
		{"文化宣导员会议", 93, []int{2, 3}},
		{"小一二三四道", 91, []int{0, 5}},
		{"小一道", 94, []int{0, 2}},
		{"小一二道", 93, []int{0, 3}},
	} {
		got, pos := run(FuzzyMatchV2, tc.text, "xdao", true)
		if got.Score != tc.score || pos == nil || !reflect.DeepEqual(*pos, tc.positions) {
			t.Errorf("%q: %+v %v; want score=%d positions=%v", tc.text, got, pos, tc.score, tc.positions)
		}
	}
}

func TestHanShortGapTradeoff(t *testing.T) {
	for _, gap := range []string{"一", "一二"} {
		text := "小" + gap + "道文文化宣导员会议"
		c := util.ToChars([]byte(text))
		for _, forward := range []bool{true, false} {
			got, pos := FuzzyMatchV2(false, true, forward, &c, []rune("xdao"), true, nil)
			want, score := []int{0, 1 + len([]rune(gap))}, 94
			if gap == "一二" {
				score = 93
				if !forward {
					start := len([]rune("小" + gap + "道文文化"))
					want = []int{start, start + 1}
				}
			}
			if got.Score != score || pos == nil || !reflect.DeepEqual(*pos, want) {
				t.Fatalf("gap=%q forward=%v: %+v %v; want %d %v", gap, forward, got, pos, score, want)
			}
		}
	}
}

func TestHanScoreBoundarySchemes(t *testing.T) {
	defer algo.Init("default")
	for _, scheme := range []string{"default", "path", "history"} {
		algo.Init(scheme)
		for _, prefix := range []string{"", " ", "/", "，", "文", "a", "1"} {
			c := util.ToChars([]byte(prefix + "中"))
			index := len([]rune(prefix))
			boundary := algo.BoundaryBonus(&c, index)
			if prefix == "文" || prefix == "a" || prefix == "1" {
				boundary = 8
			}
			got, _ := FuzzyMatchV2(false, true, true, &c, []rune("zhong"), false, nil)
			if want := 16*5 + 4*4 - 1 + 2*boundary; got.Score != want {
				t.Fatalf("%s prefix=%q: %+v; want %d (B=%d)", scheme, prefix, got, want, boundary)
			}
		}
	}
}

func TestRandomHanContiguousAlignment(t *testing.T) {
	rng := rand.New(rand.NewSource(20261123))
	defer algo.Init("default")
	checks := 0
	for _, scheme := range []string{"default", "path", "history"} {
		algo.Init(scheme)
		for _, prefix := range []string{"", "文 ", "文，", "文/", "文"} {
			for trial := 0; trial < 200; trial++ {
				pair := []struct{ a, b, query string }{{"中", "国", "zguo"}, {"北", "京", "beijing"}, {"宣", "导", "xdao"}}[rng.Intn(3)]
				// Equal spelling evidence and >=3 skipped Han guarantee a
				// strict win even over default scheme's strongest start.
				// Gap=1 can win and gap=2 can tie; separate tests pin those
				// deliberate boundary/gap tradeoffs instead of hiding them.
				gap := ""
				// Fillers cannot consume any query letters; otherwise a
				// different syllable segmentation adds different evidence.
				for n := 3 + rng.Intn(14); n > 0; n-- {
					gap += []string{"文", "田"}[rng.Intn(2)]
				}
				gapped := prefix + pair.a + gap + pair.b
				lead := strings.Repeat("文", 1+rng.Intn(7))
				contiguous := lead + pair.a + pair.b
				loose, loosePos := run(FuzzyMatchV2, gapped, pair.query, true)
				tight, _ := run(FuzzyMatchV2, contiguous, pair.query, false)
				if tight.Start < 0 || loose.Start < 0 || tight.Score <= loose.Score {
					t.Fatalf("%s %q: tight=%+v loose=%+v pos=%v (%q / %q)", scheme, pair.query, tight, loose, loosePos, contiguous, gapped)
				}
				separator := strings.Repeat("文田", 12)
				// Both occurrence orders, including the case where an early
				// gapped predecessor could mask the later adjacent path.
				for _, tightFirst := range []bool{false, true} {
					text, start := gapped+separator+contiguous, len([]rune(gapped+separator+lead))
					if tightFirst {
						text, start = contiguous+separator+gapped, len([]rune(lead))
					}
					c := util.ToChars([]byte(text))
					for _, forward := range []bool{false, true} {
						got, pos := FuzzyMatchV2(false, true, forward, &c, []rune(pair.query), true, nil)
						want := []int{start, start + 1}
						if got.Score != tight.Score || got.Start != start || got.End != start+2 || pos == nil || !reflect.DeepEqual(*pos, want) {
							t.Fatalf("scheme=%s query=%q text=%q: %+v %v; want score=%d alignment=%v", scheme, pair.query, text, got, pos, tight.Score, want)
						}
						withoutPos, _ := FuzzyMatchV2(false, true, forward, &c, []rune(pair.query), false, nil)
						if withoutPos != got {
							t.Fatal("position collection changes selected alignment")
						}
						checks++
					}
				}
			}
		}
	}
	t.Logf("%d randomized same-line alignment checks across schemes, boundaries, directions and occurrence orders", checks)
}
