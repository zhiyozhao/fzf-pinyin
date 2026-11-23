package fixture

import (
	"fmt"
	"math/rand"
	"strings"
)

// PerfCase is a reproducible whole-corpus scan, shared by library and CLI probes.
type PerfCase struct {
	Name, Corpus, Query string
	Positions           bool
}

// PerfCases never reads real paths. Seed and sizes are part of the benchmark.
func PerfCases() []PerfCase {
	rng := rand.New(rand.NewSource(20261123))
	words := []string{"北京大学", "中国文档", "开发工具", "音乐文件", "重庆银行", "方案设计", "马超", "上海数据"}
	latin := []string{"src", "project", "report", "archive", "config", "document", "notes", "main"}
	var mixed, english strings.Builder
	for i := 0; i < 100000; i++ {
		a, b := latin[rng.Intn(len(latin))], latin[rng.Intn(len(latin))]
		w := words[rng.Intn(len(words))]
		if i%2 == 0 {
			fmt.Fprintf(&mixed, "%s/%s/%s-%06d.go\n", a, w, b, i)
			fmt.Fprintf(&english, "%s/project/%s-%06d.go\n", a, b, i)
		} else {
			fmt.Fprintf(&mixed, "讨论 %06d / %s Go SDK %s 更新\n", i, w, a)
			fmt.Fprintf(&english, "discussion %06d / project Go SDK %s update\n", i, a)
		}
	}
	cases := []PerfCase{}
	for _, group := range []struct {
		name    string
		queries []string
	}{
		{"full", []string{"zhongguo", "beijing", "kaifagongju"}},
		{"initials", []string{"zg", "bj", "kfgj"}},
		{"mixed", []string{"zhongg", "beij", "kaifagj"}},
		{"latin", []string{"pr", "sdk", "config"}},
	} {
		for _, q := range group.queries {
			cases = append(cases, PerfCase{"mixed100k/" + group.name + "/" + q, mixed.String(), q, false})
		}
	}
	// Retain the historical 50%-Han fixture for the original 63/103 ms target.
	cases = append(cases, PerfCase{"historical100k", Corpus(100000, true), "zhongguo", false})
	for _, n := range []int{10000, 100000} {
		line := strings.Repeat("中国", n/2) + "\n"
		cases = append(cases, PerfCase{fmt.Sprintf("long%d", n), line, "zhongguo", false})
		cases = append(cases, PerfCase{fmt.Sprintf("long%d/positions", n), line, "zhongguo", true})
	}
	longQuery := strings.Repeat("zhongguo", 6) // 48 letters, many legal alignments.
	cases = append(cases, PerfCase{"long-query100k", mixed.String(), longQuery, false})
	cases = append(cases, PerfCase{"long-line-query", strings.Repeat("中国", 50000) + "\n", longQuery, false})
	cases = append(cases, PerfCase{"long-line-query/positions", strings.Repeat("中国", 50000) + "\n", longQuery, true})
	var poly strings.Builder
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&poly, "poly/%06d/%s.txt\n", i, strings.Repeat("重行乐长得单解曾区薄", 10))
	}
	cases = append(cases, PerfCase{"polyphonic", poly.String(), "chonghangyuezhang", false})
	for _, q := range []string{"pr", "sdk", "config"} {
		cases = append(cases, PerfCase{"english100k/" + q, english.String(), q, false})
	}
	return cases
}
