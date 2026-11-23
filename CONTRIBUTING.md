[首页](README.md)

# 开发指南

面向修改代码、运行测试或嵌入匹配库的开发者。只想安装使用，请看[安装说明](docs/install.md)。

## 构建与检查

需要 Go 1.23+、Git、make 和 sh。请克隆完整仓库；检查需要历史中固定的上游参考提交。Ruby/tmux 仅完整终端套件需要。

```sh
make -f Makefile.fzfp build
./Artifacts/fzfp --version
make -f Makefile.fzfp check
```

`make -f Makefile.fzfp build` 生成 `Artifacts/fzfp`，版本带开发标记。`make -f Makefile.fzfp check` 包含格式检查、go vet、Go 测试、独立 CLI 行为检查、隔离终端交互、race 和安装脚本测试。Homebrew 的生成器及测试由 tap 仓库维护。

fzfp 的 `test`、`e2e`、`bench`、`fuzz`、`package`、`snapshot` 等目标均在 `Makefile.fzfp`，调用时同样加 `-f Makefile.fzfp`。根目录 `Makefile` 保持上游原文；直接 `make` / `make all` 使用上游目标及 `target/fzf-*` 路径，而不是 fzfp 的开发入口。如果当前提交没有可追溯的版本标签，上游 Makefile 需要显式版本参数，例如 `make all FZF_VERSION=0.75`。这些目标仍构建当前 fork 源码；原版对照程序由 `make -f Makefile.fzfp reference` 从固定上游 worktree 单独构建。

依赖和构建缓存放在 `.cache/`，程序和报告放在 `Artifacts/`。检查会在 `.cache/upstream` 建立固定版本的只读参考 worktree，请保持其干净。构建源码不需要先安装 fzf。

## 修改匹配行为时

保持无汉字候选的分数、匹配范围、位置数组和排序与参考版本一致。拼音实现集中在 `src/pinyin/`；原版 `src/algo/algo.go` 保持不变。也要验证缓存复用和逐字输入，不能只测一次独立查询。

测试语料应使用合成或公开数据，配置、终端和临时目录应隔离。行为变化同步更新两份 README 和相关中文文档。测试的预期结果应来自已定义的规则，不能为了通过检查直接改成当前输出。

## 按任务继续阅读

- [匹配设计](docs/design.md)：评分、原文位置、缓存约束。
- [Go 库](docs/library.md)：在其他程序中调用匹配器。
- [测试与性能](docs/testing.md)：性质测试、性能测量和完整终端对照。
- [维护与发布](docs/maintenance.md)：上游更新、版本包和 Homebrew。
