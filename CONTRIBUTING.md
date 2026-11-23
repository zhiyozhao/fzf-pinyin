[首页](README.md)

# 开发指南

面向修改代码、运行测试或嵌入匹配库的开发者。只想安装使用，请看[安装说明](docs/install.md)。

## 构建与检查

需要 Go 1.23+、Git、make、sh 和 Ruby。请克隆完整仓库；检查需要历史中固定的上游参考提交。

```sh
make build
./Artifacts/fzfp --version
make check
```

`make build` 生成 `Artifacts/fzfp`，版本带开发标记。`make check` 包含格式检查、go vet、Go 测试、独立 CLI 行为检查、隔离终端交互、race、安装脚本及 Homebrew formula 生成测试。

依赖和构建缓存放在 `.cache/`，程序和报告放在 `Artifacts/`。检查会在 `.cache/upstream` 建立固定版本的只读参考 worktree，请保持其干净。构建源码不需要先安装 fzf。

## 修改匹配行为时

保持无汉字候选的分数、匹配范围、位置数组和排序与参考版本一致。拼音实现集中在 `src/pinyin/`；原版 `src/algo/algo.go` 保持不变。也要验证缓存复用和逐字输入，不能只测一次独立查询。

测试语料应使用合成或公开数据，配置、终端和临时目录应隔离。行为变化同步更新两份 README 和相关中文文档。测试的预期结果应来自已定义的规则，不能为了通过检查直接改成当前输出。

## 按任务继续阅读

- [匹配设计](docs/design.md)：评分、原文位置、缓存约束。
- [Go 库](docs/library.md)：在其他程序中调用匹配器。
- [验证方法](docs/benchmarks.md)：性质测试和性能测量。
- [终端测试](docs/e2e.md)：额外依赖、报告及已知限制。
- [维护与发布](docs/maintenance.md)：上游更新、版本包和 Homebrew。
