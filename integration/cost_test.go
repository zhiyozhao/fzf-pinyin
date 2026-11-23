package integration

import "testing"

func TestLinePenaltyRanking(t *testing.T) {
	bin := binary(t, "FZFP_BIN", "fzfp")
	for _, scheme := range []string{"default", "path", "history"} {
		for _, tc := range []struct{ query, corpus, want string }{
			{"bj", "-b-j\n-北京\n-bj\n", "-bj\n-北京\n-b-j\n"},
			{"yy", "-y-y\n-音乐\n", "-音乐\n-y-y\n"},
			{"machao", "/马超\n/machao\n", "/machao\n/马超\n"},
		} {
			got := filter(t, bin, tc.corpus, "--scheme="+scheme, "--filter="+tc.query)
			if got.code != 0 || got.stderr != "" || got.stdout != tc.want {
				t.Fatalf("%s %q: %+v; want %q", scheme, tc.query, got, tc.want)
			}
			t.Logf("%s %s: %q", scheme, tc.query, got.stdout)
		}
	}
}
