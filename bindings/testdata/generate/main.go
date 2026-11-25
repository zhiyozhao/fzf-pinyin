// Run from the repository root: go run ./bindings/testdata/generate > ...
// JSON output is deterministic: regenerate explicitly after core changes.
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/junegunn/fzf/bindings/core"
)

func main() {
	type vector struct {
		Name     string        `json:"name"`
		Request  core.Request  `json:"request"`
		Expected core.Response `json:"expected"`
	}
	data := struct {
		FormatVersion     int      `json:"format_version"`
		CoreRevision      string   `json:"core_revision"`
		ABIVersion        int      `json:"abi_version"`
		CoreVersion       string   `json:"core_version"`
		DictionaryVersion string   `json:"dictionary_version"`
		Vectors           []vector `json:"vectors"`
	}{FormatVersion: 1, CoreRevision: core.CoreRevision, ABIVersion: core.ABIVersion, CoreVersion: core.Version, DictionaryVersion: "go-pinyin v0.21.0"}
	pairs := [][2]string{
		{"中国", "zhongguo"}, {"北京", "bj"}, {"长沙", "chsh"}, {"中Go国", "zhonggo国"},
		{"中国", "中g"}, {"中国", "zhonggu"}, {"音乐", "yy"}, {"重庆", "chongqing"},
		{"绿茶", "lvcha"}, {"绿茶", "lücha"}, {"略过", "lueguo"}, {"略过", "lveguo"},
		{"巨人", "jvren"}, {"嗯好", "enhao"}, {"女儿", "nver"}, {"呣啊", "mua"},
		{"中国", "guozhong"}, {"绿茶", "lucha"}, {"嗯好", "enghao"},
		{"fooBar", "fb"}, {"FooBar", "FB"}, {"café", "cafe"}, {"cafe", "café"},
		{"😀中国e\u0301", "zg"}, {"𠀀中国", "zg"}, {"a\u0000北京\r\n", "bj"},
		{"😀", "😀"}, {"e\u0301北京", "\u0301bj"},
		{"", ""}, {"中国", ""}, {"", "x"}, {"abc", "!a"},
		{"/machao", "machao"}, {"/马超", "machao"}, {"-北京", "bj"},
		{"中中国", "zg"}, {"hello hello", "hl"},
		{strings.Repeat("中", 33000) + "国", "zg"},
	}
	add := func(name string, req core.Request) {
		expected, err := core.Execute(req)
		if err != nil {
			panic(err)
		}
		data.Vectors = append(data.Vectors, vector{name, req, expected})
	}
	yes, no := true, false
	for _, scheme := range []string{"default", "path", "history"} {
		opts := core.Options{Scheme: scheme, Normalize: &yes, Positions: &yes}
		for _, pair := range pairs {
			add(scheme+":"+pair[0]+"/"+pair[1], core.Request{Operation: "match", Text: pair[0], Query: pair[1], Options: opts})
		}
		for _, cs := range []bool{false, true} {
			for _, norm := range []*bool{&yes, &no} {
				for _, positions := range []*bool{&yes, &no} {
					o := core.Options{Scheme: scheme, CaseSensitive: cs, Normalize: norm, Positions: positions}
					for _, pair := range [][2]string{{"FooBar", "FB"}, {"café北京", "cafebj"}} {
						add(scheme+":options", core.Request{Operation: "match", Text: pair[0], Query: pair[1], Options: o})
					}
				}
			}
		}
		candidates := []string{"xx/北京.md", "北京/xx.md", " 北京 ", "-b-j", "-北京", "-bj", "北京", "北京", "😀北京", "no match"}
		for _, query := range []string{"bj", "beijing", "", "nomatch!"} {
			add(scheme+":filter:"+query, core.Request{Operation: "filter", Candidates: candidates, Query: query, Options: opts})
		}
		add(scheme+":tie-heavy-regression", core.Request{Operation: "filter", Candidates: []string{"case-test_qux", "test-cherry.index", "test_cherry-world", "test_finder-match", "grape-peach test", "score-lemon-filter"}, Query: "st", Options: opts})
	}
	// Shared acceptance cases include enough candidates to exercise batch
	// parallelism, random tie keys, and upstream's fixed-slab V1 fallback.
	rng := rand.New(rand.NewSource(20261125))
	parts := []string{"北京", "中国", "绿茶", "fooBar", "a-b", "aab", "😀", "é", "e\u0301", "/", "\\", " ", "嗯略巨"}
	candidates := []string{"", "a\x00北京\r\n", strings.Repeat("ab ", 2200) + "北京", strings.Repeat("a", 3400) + "b"}
	for i := 0; i < 1100; i++ {
		s := ""
		for j := 0; j < 2+rng.Intn(5); j++ {
			s += parts[rng.Intn(len(parts))]
		}
		candidates = append(candidates, s)
	}
	for _, scheme := range []string{"default", "path", "history"} {
		for _, query := range []string{"bj", "ab", strings.Repeat("a", 40) + "b"} {
			add(fmt.Sprintf("%s:acceptance:%s", scheme, query), core.Request{Operation: "filter", Candidates: candidates, Query: query, Options: core.Options{Scheme: scheme, Normalize: &yes, Positions: &yes}})
		}
		noPositions := core.Options{Scheme: scheme, Normalize: &yes, Positions: &no}
		add(scheme+":saturated-rank", core.Request{Operation: "filter", Candidates: []string{strings.Repeat("a", 3400), "/" + strings.Repeat("a", 3400), strings.Repeat("a", 3401), " " + strings.Repeat("a", 3400) + " "}, Query: strings.Repeat("a", 3400), Options: noPositions})
	}
	e := json.NewEncoder(os.Stdout)
	e.SetEscapeHTML(false)
	e.SetIndent("", "  ")
	if err := e.Encode(data); err != nil {
		panic(err)
	}
}
