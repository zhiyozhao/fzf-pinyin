// Package fixture generates synthetic inputs without accessing user files.
package fixture

import (
	"fmt"
	"math/rand"
	"strings"
)

// Corpus returns deterministic path-like data. Mixed puts Han in every other
// line, with some literal Latin, emoji and accented Latin in both datasets.
func Corpus(lines int, mixed bool) string {
	rng := rand.New(rand.NewSource(20261120))
	latin := []string{"src", "project", "report", "archive", "main", "config", "document", "file", "notes", "école"}
	han := []string{"中国", "中文测试", "重庆银行", "项目文档", "方案设计", "音乐文件", "北京大学", "上海数据", "开发工具", "女绿"}
	var out strings.Builder
	for i := 0; i < lines; i++ {
		a, b := latin[rng.Intn(len(latin))], latin[rng.Intn(len(latin))]
		if mixed && i%2 == 0 {
			b = han[rng.Intn(len(han))]
		}
		fmt.Fprintf(&out, "%s/%s/%s-%06d.go\n", a, b, latin[rng.Intn(len(latin))], i)
	}
	return out.String()
}
