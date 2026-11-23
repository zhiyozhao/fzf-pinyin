[开发指南](../CONTRIBUTING.md)

# 完整终端测试

面向修改终端行为或准备发布的开发者。这是额外套件，不包含在 `make check` 中。

## 环境与执行

需要 Ruby 3.4+、tmux，以及 checkout 内 `.cache/gems` 中的 minitest 5.25.4。floating-pane 测试需要 tmux 3.7+。

通过 `E2E_RUBY` 和 `E2E_TMUX` 选择解释器与 tmux。Ruby 默认从 PATH 选择；tmux 默认取 Homebrew 在 Apple Silicon 上的安装位置，其他环境需显式设置 `E2E_TMUX`。`make e2e` 先测试固定上游，再测试本程序。macOS 自带 Ruby 版本通常过旧，需要选择兼容版本。完整运行可能需要数分钟。

如果使用 Ruby 4，将 `E2E_RUBY` 设为对应可执行文件并导出，然后在仓库根目录安装测试依赖：

```sh
GEM_HOME="$PWD/.cache/gems" GEM_PATH="$PWD/.cache/gems" \
  "$E2E_RUBY" -S gem install --no-document --force minitest -v 5.25.4
make e2e
```

`--force` 绕过 minitest 5.25.4 声明的 Ruby <4.0 限制，不代表依赖官方支持 Ruby 4；需要检查实际运行结果。使用 Ruby 3.4 时不需要该标记。依赖仅安装到 checkout，不修改全局 gems。

套件覆盖 core、exec、filter、layout、preview、raw、server 和 tmux，不包括 shell 集成或 Vim。fixture 中的程序名虽然是 `fzf`，两轮分别指向实际被测程序，不改版本输出来通过断言。

## 隔离与报告

测试使用独立 tmux socket、空配置、临时 HOME 和目录、独占端口。终端默认为 80×24。环境适配在临时上游副本中进行，不修改断言。

报告保存在 `Artifacts/e2e/`，可用 `E2E_OUTPUT` 改目录。每轮包含逐用例 JSONL、终端日志、测试注册表和汇总；`comparison.json` 记录状态差异。新增失败、跳过或缺失用例使比较失败。

两轮都有的错误仍需查看具体内容，不能仅凭错误总数相同判断兼容。

## 已知限制

在记录过的 80×24 环境中，上游布局组合矩阵的一个预览布局只剩一行列表，断言同时访问两个条目时产生 NoMethodError。加高终端后两个条目均可见；两轮出现同一错误。不能据此声称整个布局矩阵已通过。

保留原始断言，区分共同环境错误和新增回归。`E2E_LONGTEST=0` 可以关闭长矩阵，但必须报告为缩小了覆盖范围。
