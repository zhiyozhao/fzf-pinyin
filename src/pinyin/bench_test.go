package pinyin

import (
	"fmt"
	"strings"
	"testing"

	"github.com/junegunn/fzf/internal/fixture"
	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func BenchmarkMatch(b *testing.B) {
	for _, mixed := range []bool{false, true} {
		lines := strings.Split(strings.TrimSuffix(fixture.Corpus(1000, mixed), "\n"), "\n")
		chars := make([]util.Chars, len(lines))
		for i, line := range lines {
			chars[i] = util.ToChars([]byte(line))
		}
		for _, query := range []string{"pr", "zg", "zhongguo"} {
			for _, matcher := range []struct {
				name string
				fn   algo.Algo
			}{{"fzf", algo.FuzzyMatchV2}, {"fzfp", FuzzyMatchV2}} {
				b.Run(fmt.Sprintf("mixed=%t/%s/%s", mixed, query, matcher.name), func(b *testing.B) {
					q := []rune(query)
					slab := util.MakeSlab(100000, 2048)
					for _, c := range chars {
						matcher.fn(false, true, true, &c, q, false, slab)
					}
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						matcher.fn(false, true, true, &chars[i%len(chars)], q, false, slab)
					}
				})
			}
		}
	}
}
