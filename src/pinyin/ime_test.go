package pinyin

import (
	"strings"
	"testing"

	"github.com/junegunn/fzf/src/algo"
)

// Independent, user-visible full-code expectations. Append another syllable
// so an incomplete final token cannot masquerade as a complete IME code.
func TestIMEFullCodes(t *testing.T) {
	for _, tc := range []struct {
		han, accept, reject string
	}{
		{"虑", "lv l", "lu"}, {"女", "nv n", "nu"},
		{"略", "lve lue l", "lvv luei"}, {"虐", "nve nue n", "nvv nuei"},
		{"居", "ju jv j", "jvu jiu"}, {"觉", "jue jve j", "jveu"},
		{"卷", "juan jvan j", "jvanv"}, {"军", "jun jvn j", "juen"},
		{"去", "qu qv q", "qvu"}, {"却", "que qve q", "qveu"},
		{"全", "quan qvan q", "qvanv"}, {"群", "qun qvn q", "quen"},
		{"需", "xu xv x", "xvu"}, {"学", "xue xve x", "xveu"},
		{"宣", "xuan xvan x", "xvanv"}, {"寻", "xun xvn x", "xuen"},
		{"鱼", "yu yv y", "v u"}, {"月", "yue yve y", "ve"},
		{"元", "yuan yvan y", "van"}, {"云", "yun yvn y", "vn yuen"},
		{"路", "lu l", "lv"}, {"怒", "nu n", "nv"}, {"古", "gu g", "gv"},
		{"嗯", "en ng e n", "eng mu"}, {"哼", "heng h", "hng eng"},
		{"呣", "m mou mu", "en ng"}, {"噷", "hm hen xin h x", "heng hng mu"},
		{"欸", "ai ei xie e a x", "eh"}, {"诶", "ei e", "ai eh xi yi"},
		{"誒", "ei xi yi e x y", "eh"},
		{"呒", "fu mu m wu f w", "en ng"}, {"嘸", "fu wu m mu f w", "en ng"},
		{"音", "yin y", "in i"}, {"万", "wan w", "uan u"},
		{"一", "yi y", "i"}, {"五", "wu w", "u"}, {"要", "yao y", "iao i"},
		{"𰻝", "biang b", "bang"}, {"𤭢", "cei c", "ce"},
		{"𥥈", "wong w", "ong veng"}, {"鞥", "eng e", "en"},
		{"欻", "chua", "chva"}, {"扽", "den", "deng"}, {"𨈖", "din", "ding"},
		{"覅", "fiao", "fao"}, {"剋", "kei", "kuei"}, {"㖸", "nia", "niang"},
		{"黁", "nun", "nuen"}, {"挼", "rua", "rva"}, {"忒", "tei", "tuei"}, {"这", "zhei", "zei"},
		{"啉", "lin lan l", "len"}, {"𱿅", "", "bong b beng bang"},
		{"㕶", "n ng", "en eng mu"}, {"唔", "wu n ng m w", "en eng mu"},
		{"哽", "geng ying n ng g y", "en eng mu"},
		{"滚", "gun g", "guen gvn"}, {"归", "gui g", "guei gvi"},
		{"九", "jiu j", "jiou jiv"},
		{"知", "zhi zh z", "zi"}, {"吃", "chi ch c", "ci"}, {"诗", "shi sh s", "si"},
		{"南", "nan n", "lan"}, {"来", "lai l", "nai"},
		{"儿", "er e", "err er0 er5"}, {"的", "de d", "de0 de5"},
		{"北", "bei b", "bie"},
	} {
		t.Run(tc.han, func(t *testing.T) {
			for _, set := range []struct {
				codes string
				want  bool
			}{{tc.accept, true}, {tc.reject, false}} {
				for _, code := range strings.Fields(set.codes) {
					for _, fn := range []algo.Algo{EqualMatch, FuzzyMatchV1, FuzzyMatchV2} {
						got, _ := run(fn, tc.han+"中", code+"zhong", true)
						if (got.Start >= 0) != set.want {
							t.Fatalf("%q / %q: %+v, want recall=%t", tc.han+"中", code+"zhong", got, set.want)
						}
					}
				}
			}
		})
	}
}

func TestIMEFinalPrefixesAndRemovedFinals(t *testing.T) {
	for _, tc := range []struct {
		han, q string
		want   bool
	}{
		{"虑", "lv", true}, {"虑", "lu", false}, {"女", "nv", true}, {"女", "nu", false},
		{"略", "lue", true}, {"略", "lve", true}, {"居", "ju", true}, {"居", "jv", true},
		{"鱼", "yu", true}, {"鱼", "yv", true}, {"鱼", "v", false},
		{"音", "yin", true}, {"音", "in", false}, {"万", "wan", true}, {"万", "uan", false},
		{"一", "i", false}, {"五", "u", false}, {"要", "iao", false},
		{"嗯", "en", true}, {"嗯", "ng", true}, {"嗯", "eng", false},
		{"宣中", "xvanzho", true}, {"宣中", "xvzhong", false},
		{"诶", "e", true}, {"噷", "he", true},
		// 儿 also has dictionary ren, so r remains a legal abbreviation.
		{"花儿", "huar", true}, {"玩儿", "war", false},
	} {
		for _, fn := range []algo.Algo{FuzzyMatchV1, FuzzyMatchV2, ExactMatchNaive, PrefixMatch, SuffixMatch, EqualMatch, ExactMatchBoundary} {
			got, _ := run(fn, tc.han, tc.q, true)
			if (got.Start >= 0) != tc.want {
				t.Fatalf("%q / %q: %+v, want recall=%t", tc.han, tc.q, got, tc.want)
			}
		}
	}
}
