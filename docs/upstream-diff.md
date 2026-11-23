[维护与发布](maintenance.md)

# 上游文件审查

参考提交为 `bf4869b7fa7748d973eeb414da4dde0d7fe4cd3c`。下表仅列出上游已经存在、在本 fork 中仍修改或删除的文件；新增拼音包、测试、安装器和文档不属于这张表。

| 上游文件 | 状态 | 保留差异的理由 |
|---|---|---|
| `README.md` | 修改 | 本项目中文首页；上游原文逐字节保存在 `README.fzf.md` |
| `main.go` | 修改 | 独立版本标识、fzfp 帮助中的命令名和项目地址，以及嵌入 fzfp 手册；不只有版本号变化 |
| `src/pattern.go` | 修改 | 将各搜索模式接到拼音匹配器，并使用只复用前缀的候选缓存，避免拼音后缀缓存漏召回 |
| `go.mod` | 修改 | 新增 go-pinyin 读音数据依赖和仅用于终端测试的 creack/pty；保留上游模块路径 |
| `go.sum` | 修改 | 固定上述两个新增依赖的校验和 |
| `Makefile` | 修改 | 保留已使用的默认构建、检查及发布工作流入口，生成 Artifacts/fzfp，固定上游参考并运行拼音、缓存、终端和打包检查；上游原文另存 Makefile.fzf |
| `.gitignore` | 修改 | 忽略本项目缓存、构建产物及本地配置，避免它们进入提交 |
| `.goreleaser.yml` | 修改 | 生成 fzfp 的 macOS/Linux 四平台包、手册、许可和校验文件，发布到 fork，避免使用上游的签名和发布目标 |
| `.github/CODEOWNERS` | 修改 | 代码审阅责任属于本 fork 维护者，不向上游作者请求审阅 |
| `.github/workflows/release.yml` | 修改 | 仅处理 fzfp 标签，验证并发布本项目资产，不执行上游签名、Windows MSI 或 winget 流程 |
| `.github/workflows/linux.yml` | 删除 | 上游任务执行旧 Makefile 的 lint/install 和上游 shell 集成安装；本 fork 使用新增 check.yml，终端对照另有隔离入口 |
| `.github/workflows/macos.yml` | 删除 | 同样依赖上游 install 与 shell 集成流程，且其 Ruby 版本不满足当前上游测试要求；保留原文件会提供不能正确执行的手动入口 |
| `.github/workflows/winget.yml` | 删除 | 目标是 junegunn.fzf，且本项目不发布 Windows 包；避免留下指向上游产品的发布入口 |

## 已恢复的文件

六份上游文档（ADVANCED、README-VIM、CHANGELOG、BUILD、RELEASE、SECURITY）、LICENSE、资助配置、通用 issue/PR 模板，以及 CodeQL、依赖审查、标签和拼写检查工作流，均恢复为参考提交的原文。LICENSE 中本 fork 曾追加的版权行移除；新增代码的版权及 MIT 声明保留在 NOTICE，第三方声明同步指向它。

`Makefile.fzf` 移除了额外说明注释，现在与参考 Makefile 逐字节一致。`README.fzf.md` 也与参考 README 逐字节一致。本项目的文档不会替换这些上游参考内容。

## 复核方式

用 Git 比较参考提交和当前 HEAD，包含已删除文件：

```sh
git diff --stat bf4869b HEAD
git diff --name-status --diff-filter=MD bf4869b HEAD
```

第一条包括新增文件；第二条仅列出仍然修改或删除的上游文件。后续上游更新时同步这张表，逐项审阅差异的必要性。
