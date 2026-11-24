package integration

import "testing"

func TestIMEKeyboardCLI(t *testing.T) {
	bin := binary(t, "FZFP_BIN", "fzfp")
	for _, tc := range []struct {
		query, corpus, want string
	}{
		{"^lv$", "虑\n路\n", "虑\n"}, {"^lu$", "虑\n路\n", "路\n"},
		{"^nv$", "女\n怒\n", "女\n"}, {"^nu$", "女\n怒\n", "怒\n"},
		{"^lue$", "略\n", "略\n"}, {"^lve$", "略\n", "略\n"},
		{"^nue$", "虐\n", "虐\n"}, {"^nve$", "虐\n", "虐\n"},
		{"^ju$", "居\n", "居\n"}, {"^jv$", "居\n", "居\n"},
		{"^yu$", "鱼\n", "鱼\n"}, {"^yv$", "鱼\n", "鱼\n"}, {"^v$", "鱼\n", ""},
		{"^yin$", "音\n", "音\n"}, {"^in$", "音\n", ""},
		{"^wan$", "万\n", "万\n"}, {"^uan$", "万\n", ""},
		{"^en$", "嗯\n", "嗯\n"}, {"^ng$", "嗯\n", "嗯\n"}, {"^eng$", "嗯\n", ""},
		{"^env$", "嗯女\n", "嗯女\n"}, // e abbreviation is valid before another syllable
		{"^heng$", "哼\n", "哼\n"}, {"^hng$", "哼\n", ""},
		{"^mu$", "呣\n呒\n嘸\n", "呣\n呒\n嘸\n"},
		{"^mou$", "呣\n", "呣\n"}, {"^m$", "呣\n呒\n嘸\n", "呣\n呒\n嘸\n"},
		{"^hm$", "噷\n", "噷\n"}, {"^hen$", "噷\n", "噷\n"}, {"^xin$", "噷\n", "噷\n"},
		{"^ezhong$", "欸中\n", "欸中\n"}, {"^ehzhong$", "欸中\n誒中\n诶中\n", ""},
		{"^eizhong$", "欸中\n誒中\n诶中\n", "欸中\n誒中\n诶中\n"},
		{"^fu$", "呒\n嘸\n", "呒\n嘸\n"}, {"^wu$", "呒\n嘸\n", "呒\n嘸\n"},
	} {
		got := filter(t, bin, tc.corpus, "--no-sort", "--filter", tc.query)
		code := 0
		if tc.want == "" {
			code = 1
		}
		if got.stdout != tc.want || got.stderr != "" || got.code != code {
			t.Fatalf("%q / %q: %+v, want %q exit=%d", tc.corpus, tc.query, got, tc.want, code)
		}
	}
}
