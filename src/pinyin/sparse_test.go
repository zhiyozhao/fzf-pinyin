package pinyin

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func TestSparseDenseParity(t *testing.T) {
	rng := rand.New(rand.NewSource(1128))
	alphabet := []rune("aAbBcC19 _-/.\t\r\n\u3000你好收货地址备忘录女宣éÜ👋𰻝")
	queries := []string{"s", "shd", "shdz", "xuan", "nihao", "dizhi", "a b", "A", "ü", "nv", "你好", "👋b", "é", "ZZ", "biang"}
	for _, scheme := range []string{"history", "default", "path"} {
		algo.WithLibraryScheme(scheme, func() {
			for trial := 0; trial < 600; trial++ {
				text := strings.Repeat("q09 _\n", 700)
				for i := 0; i < 80; i++ {
					text += string(alphabet[rng.Intn(len(alphabet))])
				}
				text += strings.Repeat("q_", rng.Intn(80)) + "收货地址宣导女你好👋𰻝 A/abc"
				if trial%3 == 0 {
					text = "\n收货地址" + text + "\n"
				}
				chars := util.ToChars([]byte(text))
				q := []rune(queries[rng.Intn(len(queries))])
				mode := Mode(rng.Intn(6))
				cs, norm, forward := rng.Intn(2) == 0, rng.Intn(2) == 0, rng.Intn(2) == 0
				allow := rng.Intn(2) == 0
				for _, positions := range []bool{false, true} {
					got, gp := matchRange(cs, norm, forward, &chars, q, positions, mode, allow, 0, chars.Length(), false)
					want, wp := denseMatchRange(cs, norm, forward, &chars, q, positions, mode, allow, 0, chars.Length(), false)
					if got != want || !reflect.DeepEqual(gp, wp) {
						t.Fatalf("scheme=%s trial=%d q=%q mode=%d cs=%t norm=%t forward=%t allow=%t positions=%t got=%v/%v want=%v/%v", scheme, trial, string(q), mode, cs, norm, forward, allow, positions, got, gp, want, wp)
					}
				}
			}
		})
	}
}

func TestSparseLongGapAndWinningInterval(t *testing.T) {
	algo.WithLibraryScheme("history", func() {
		for _, text := range []string{"收" + strings.Repeat("0", 90000) + "货地址", "收货地址" + strings.Repeat("0", 90000) + "收货地址", strings.Repeat("0", 90000) + "收货地址"} {
			chars := util.ToChars([]byte(text))
			for _, q := range []string{"shdz", "shd", "s"} {
				for _, forward := range []bool{false, true} {
					got, gp := match(false, true, forward, &chars, []rune(q), true, Fuzzy, true)
					want, wp := denseMatchRange(false, true, forward, &chars, []rune(q), true, Fuzzy, true, 0, chars.Length(), false)
					if got != want || !reflect.DeepEqual(gp, wp) {
						t.Fatalf("q=%s forward=%t got=%v/%v want=%v/%v", q, forward, got, gp, want, wp)
					}
				}
			}
		}
	})
}

func BenchmarkLongDPComponents(b *testing.B) {
	text := "收货地址\n" + strings.Repeat("synthetic line 你好 0123456789\n", 39999) + "END"
	chars := util.ToChars([]byte(text))
	for _, query := range []string{"s", "shdz", "shd"} {
		q := []rune(query)
		b.Run(query, func(b *testing.B) {
			algo.WithLibraryScheme("history", func() {
				for _, name := range []string{"native", "dense-phonetic", "dense-literal", "sparse-phonetic", "sparse-literal"} {
					b.Run(name, func(b *testing.B) {
						slab := util.MakeSlab(100000, 2048)
						b.ResetTimer()
						for i := 0; i < b.N; i++ {
							switch name {
							case "native":
								algo.FuzzyMatchV2(false, true, true, &chars, q, false, slab)
							case "dense-phonetic":
								denseMatchRange(false, true, true, &chars, q, false, Fuzzy, true, 0, chars.Length(), false)
							case "dense-literal":
								denseMatchRange(false, true, true, &chars, q, false, Fuzzy, false, 0, chars.Length(), false)
							case "sparse-phonetic":
								match(false, true, true, &chars, q, false, Fuzzy, true)
							case "sparse-literal":
								match(false, true, true, &chars, q, false, Fuzzy, false)
							}
						}
					})
				}
			})
		})
	}
}
