package integration

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func TestMachaoPathRanking(t *testing.T) {
	bin := binary(t, "FZFP_BIN", "fzfp")
	paths := []string{
		"Codes/Agents/Devs/projects/maclip-HANDOFF-1117.md",
		"Codes/Agents/Devs/projects/maclip/CHANGELOG.md",
		"Work/pdp-bot/node_modules/side-channel-map/CHANGELOG.md",
		"Pictures/koh/super/马超-冷晖之枪.png",
		"Pictures/koh/normal/马超-幸存者.jpg",
	}
	for _, scheme := range []string{"default", "path", "history"} {
		got := filter(t, bin, strings.Join(paths, "\n")+"\n", "--scheme="+scheme, "--filter=machao")
		lines := strings.Split(strings.TrimSpace(got.stdout), "\n")
		if got.code != 0 || len(lines) != len(paths) || !strings.Contains(lines[0], "/马超-") || !strings.Contains(lines[1], "/马超-") {
			t.Fatalf("scheme=%s: %+v; want both 马超 files first", scheme, got)
		}
	}
}

func TestRankingExpectations(t *testing.T) {
	bin := binary(t, "FZFP_BIN", "fzfp")
	for _, tc := range []struct{ name, query, first, second, scheme, want string }{
		{"word start", "beijing", "notes/啊北京.txt", "notes/北京.txt", "default", "notes/北京.txt"},
		{"shorter tied line", "bj", "notes/北京-详细记录.txt", "notes/北京.txt", "default", "notes/北京.txt"},
		{"history keeps tied input order", "bj", "notes/北京-详细记录.txt", "notes/北京.txt", "history", "notes/北京-详细记录.txt"},
		{"path prefers basename on equal score", "bj", "a/北京/x", "long/北京", "path", "long/北京"},
		{"default has no basename preference", "bj", "a/北京/x", "long/北京", "default", "a/北京/x"},
		{"mixed scripts", "beijinggo", "notes/北--京Go.txt", "notes/北京Go.txt", "default", "notes/北京Go.txt"},
		{"mixed literal Han", "北jing", "notes/北--京.txt", "notes/北京.txt", "default", "notes/北京.txt"},
		{"full spelling vs initials for same query", "beijing", "notes/波额伊鸡伊年哥.txt", "notes/北京.txt", "default", "notes/北京.txt"},
		{"complete final vs prefix", "beiji", "notes/北京.txt", "notes/北极.txt", "default", "notes/北极.txt"},
		{"unweighted readings", "chongming", "重名.txt", "虫鸣.txt", "default", "重名.txt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := filter(t, bin, tc.first+"\n"+tc.second+"\n", "--scheme="+tc.scheme, "--filter="+tc.query)
			if got.code != 0 || !strings.HasPrefix(got.stdout, tc.want+"\n") {
				t.Fatalf("%+v; want %q first", got, tc.want)
			}
		})
	}
}

func TestPathAndLengthRulesAreUpstream(t *testing.T) {
	adapted := binary(t, "FZFP_BIN", "fzfp")
	native := binary(t, "FZF_REFERENCE_BIN", "fzf-reference")
	for _, tc := range []struct{ corpus, scheme, first string }{
		{"a/bj/x\nlong/bj\n", "path", "long/bj"},
		{"a/bj/x\nlong/bj\n", "default", "a/bj/x"},
		{"notes/bj-detailed-log.txt\nnotes/bj.txt\n", "default", "notes/bj.txt"},
		{"notes/bj-detailed-log.txt\nnotes/bj.txt\n", "history", "notes/bj-detailed-log.txt"},
	} {
		want := filter(t, native, tc.corpus, "--scheme="+tc.scheme, "--filter=bj")
		got := filter(t, adapted, tc.corpus, "--scheme="+tc.scheme, "--filter=bj")
		if got != want || !strings.HasPrefix(want.stdout, tc.first+"\n") {
			t.Fatalf("%s: native=%+v adapted=%+v; want %q first", tc.scheme, want, got, tc.first)
		}
	}
}

// fd-style paths, document titles and chat subjects are entirely generated.
// Check an ordered policy across the full result set, not just the top hit.
func TestSyntheticRankingAudit(t *testing.T) {
	bin := binary(t, "FZFP_BIN", "fzfp")
	rng := rand.New(rand.NewSource(20261125))
	words := []struct{ han, full, initials string }{
		{"北京", "beijing", "bj"}, {"马超", "machao", "mc"},
		{"中国", "zhongguo", "zg"}, {"文档", "wendang", "wd"},
		{"开发工具", "kaifagongju", "kfgj"}, {"音乐", "yinyue", "yy"},
	}
	var corpus strings.Builder
	for i := 0; i < 30000; i++ {
		w := words[rng.Intn(len(words))]
		kind := i % 5
		label := []string{"full", "loose-full", "initials", "loose-initials", "title"}[kind]
		prefix := fmt.Sprintf("corpus/%s/%06d/", label, i)
		switch kind {
		case 0, 2:
			fmt.Fprintf(&corpus, "%s%s-记录.md\n", prefix, w.han)
		case 1, 3:
			q := w.full
			if kind == 3 {
				q = w.initials
			}
			cut := 1 + rng.Intn(len(q)-1)
			fmt.Fprintf(&corpus, "%s%s%s%s-log.txt\n", prefix, q[:cut], strings.Repeat("x", 2+rng.Intn(10)), q[cut:])
		case 4:
			fmt.Fprintf(&corpus, "讨论 %06d / %sGo SDK 更新与文档\n", i, w.han)
		}
	}
	for _, scheme := range []string{"default", "path", "history"} {
		for _, w := range words {
			for _, query := range []string{w.full, w.initials} {
				got := filter(t, bin, corpus.String(), "--scheme="+scheme, "--filter="+query)
				if got.code != 0 {
					t.Fatal(got)
				}
				looseSeen, tightCount, looseCount := false, 0, 0
				for _, line := range strings.Split(strings.TrimSuffix(got.stdout, "\n"), "\n") {
					if strings.HasPrefix(line, "corpus/loose-") {
						looseSeen = true
						looseCount++
					}
					if strings.HasPrefix(line, "corpus/") && strings.Contains(line, "/"+w.han+"-") {
						tightCount++
						if looseSeen {
							t.Fatalf("scheme=%s query=%s: continuous Chinese ranked below fragmented literal: %s", scheme, query, line)
						}
					}
				}
				if tightCount == 0 || looseCount == 0 {
					t.Fatalf("audit has no competitors: %s %s tight=%d loose=%d", scheme, query, tightCount, looseCount)
				}
				t.Logf("%s %s: %d continuous Han before %d fragmented literal matches", scheme, query, tightCount, looseCount)
			}
		}
	}
}
