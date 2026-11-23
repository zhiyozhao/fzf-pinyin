package integration

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/junegunn/fzf/internal/fixture"
)

func binary(t *testing.T, name, fallback string) string {
	t.Helper()
	path := os.Getenv(name)
	if path == "" {
		path = filepath.Join("..", "Artifacts", fallback)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("build binaries with make check first: %v", err)
	}
	return path
}

func cleanEnv() []string {
	env := []string{}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "FZF_") || strings.HasPrefix(e, "FZFP_") || strings.HasPrefix(e, "SHELL=") {
			continue
		}
		env = append(env, e)
	}
	return append(env, "SHELL=/bin/sh")
}

type output struct {
	stdout, stderr string
	code           int
}

func filter(t *testing.T, bin, corpus string, args ...string) output {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = cleanEnv()
	cmd.Stdin = strings.NewReader(corpus)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	if ctx.Err() != nil {
		t.Fatal("CLI timed out")
	}
	return output{stdout.String(), stderr.String(), code}
}

func TestNonHanCLIParity(t *testing.T) {
	adapted := binary(t, "FZFP_BIN", "fzfp")
	native := binary(t, "FZF_REFERENCE_BIN", "fzf-reference")
	corpus := fixture.Corpus(1200, false) + "Ωmega/한글/かな🙂\nABC abc\nfoo bar\nfoo_bar\nproject archive\n\x1b[31mproject\x1b[0m/archive.go\n"
	queries := []string{"", "pr", "src", "'report", "^src", ".go$", "!archive", "!'pr", "src | notes", "^src !report", "'foo'", "^ABC$", "foo\\ bar", "中文", "Ω", "한", "🙂", "^src$ | 'config", "!^src", "!.go$"}
	options := [][]string{
		{}, {"--algo=v1"}, {"--exact"}, {"+x"}, {"-i"}, {"+i", "--literal"},
		{"--scheme=path"}, {"--scheme=history", "--tiebreak=index"},
		{"--tiebreak=begin,end,length"}, {"--tac"}, {"--no-sort"},
		{"--delimiter=/", "--nth=2..", "--with-nth=2,1"},
		{"--ansi", "--read0", "--print0", "--print-query"},
	}
	for i, opts := range options {
		for _, q := range queries {
			input := corpus
			if i == len(options)-1 {
				input = strings.ReplaceAll(input, "\n", "\x00")
			}
			args := append(append([]string{}, opts...), "--filter", q)
			want, got := filter(t, native, input, args...), filter(t, adapted, input, args...)
			if got != want {
				t.Fatalf("args=%q\nwant=%+v\ngot=%+v", args, want, got)
			}
		}
	}
	t.Logf("%d byte-for-byte CLI comparisons: output order, stderr, exit status", len(options)*len(queries))
}

func TestPinyinSearchSyntax(t *testing.T) {
	bin := binary(t, "FZFP_BIN", "fzfp")
	corpus := "中国\n中国工具\n工具中国\n中x国\n北京\n"
	for _, tc := range []struct {
		q        string
		expected []string
	}{
		{"zg", []string{"中国", "中国工具", "工具中国", "中x国"}},
		{"'zg", []string{"中国", "中国工具", "工具中国"}},
		{"^zg", []string{"中国", "中国工具"}},
		{"zg$", []string{"中国", "工具中国"}},
		{"^zg$", []string{"中国"}},
		{"'zg'", []string{"中国"}},
		{"!zg", []string{"中x国", "北京"}},
		{"!'zg", []string{"北京"}},
		{"^zg | bj", []string{"中国", "中国工具", "北京"}},
		{"zg !gj", []string{"中国", "中x国"}},
	} {
		t.Run(tc.q, func(t *testing.T) {
			got := filter(t, bin, corpus, "--no-sort", "--filter", tc.q)
			want := strings.Join(tc.expected, "\n") + "\n"
			if got.code != 0 || got.stdout != want {
				t.Fatalf("got %+v; want %q", got, want)
			}
		})
	}
	for _, args := range [][]string{
		{"--exact", "--filter=zg"}, {"+x", "--filter=zg"}, {"--algo=v1", "--filter=zhongguo"},
		{"--delimiter=:", "--nth=2", "--filter=zg"},
	} {
		input := "id:中国\nother:北京\n"
		got := filter(t, bin, input, args...)
		if got.code != 0 || got.stdout != "id:中国\n" {
			t.Fatalf("args=%v got=%+v", args, got)
		}
	}
}

func TestMixedRanking(t *testing.T) {
	bin := binary(t, "FZFP_BIN", "fzfp")
	for _, q := range []string{"zg", "zhongguo"} {
		corpus := fmt.Sprintf("中--------国\nx中国\n中国\n%s\n", q)
		got := filter(t, bin, corpus, "--filter", q)
		want := q + "\n中国\nx中国\n中--------国\n"
		if got.stdout != want {
			t.Fatalf("q=%q got %q, want %q", q, got.stdout, want)
		}
	}
}
