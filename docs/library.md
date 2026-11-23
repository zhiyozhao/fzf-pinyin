[开发指南](../CONTRIBUTING.md)

# 在 Go 中使用匹配库

`src/pinyin` 提供与 fzf `algo.Algo` 相同的单条候选匹配接口，不负责命令行查询语法、字段、列表排序或终端界面。

## 引入仓库

这个 fork 保留 `github.com/junegunn/fzf` 模块路径；`fzfp-v*` 是 CLI 发布标签，不是 Go 模块语义版本。不要用指向 fork 的 `go install …@latest` 安装 CLI。

在调用方模块旁放置经过审阅的源码 checkout，并在调用方的 `go.mod` 中明确替换整个模块：

```go
require github.com/junegunn/fzf v0.0.0
replace github.com/junegunn/fzf => ../fzf-pinyin
```

相对路径指向该 checkout；这也会影响依赖图中其他导入 fzf 的包。移除替换后使用的是上游，不包含此拼音包。

## 最小示例

```go
package main

import (
    "fmt"

    "github.com/junegunn/fzf/src/algo"
    "github.com/junegunn/fzf/src/pinyin"
    "github.com/junegunn/fzf/src/util"
)

func main() {
    algo.Init("default")
    text := util.ToChars([]byte("中国"))
    result, positions := pinyin.FuzzyMatchV2(
        false, true, true, &text, []rune("zhonggu"), true, nil,
    )
    fmt.Println(result.Start, result.End, len(*positions))
    // 0 2 2
}
```

启动并发调用前初始化一次 scheme。查询需遵守 `algo.Algo` 的预处理约定：按大小写模式折叠，并在需要时归一化。CLI 的 Pattern 层已经负责这些处理，直接调用库时由调用方负责。

位置计数是原文 rune，结束位置不包含在范围内；位置数组每个原字最多出现一次，不要假设数组按升序排列。无匹配时 `Start` 为负数，不应解引用位置指针。

其他入口包括 `FuzzyMatchV1`、`ExactMatchNaive`、`ExactMatchBoundary`、`PrefixMatch`、`SuffixMatch` 和 `EqualMatch`。`Wrap(fallback, mode)` 可以适配原版匹配器。无 Han 候选直接使用 fallback；含 Han 时它仍是字面匹配的保底结果。评分、复杂度和并发约束见[匹配设计](design.md)。
