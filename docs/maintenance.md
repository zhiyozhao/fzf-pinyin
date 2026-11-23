[开发指南](../CONTRIBUTING.md)

# 维护与发布

面向更新上游、制作版本包和维护 Homebrew formula 的维护者。

## 与上游文件的边界

上游已有文件尽量保持原样；本项目文档单独新增。上游的 `ADVANCED.md`、`README-VIM.md`、`CHANGELOG.md`、`BUILD.md`、`RELEASE.md`、`SECURITY.md` 保持参考版本原文。fzfp 的构建说明在 [CONTRIBUTING](../CONTRIBUTING.md)，发布说明在本文，版本变化在 [CHANGELOG.fzfp.md](../CHANGELOG.fzfp.md)。

根目录 `Makefile` 与参考版本逐字节一致；fzfp 的构建、检查和打包入口独立放在 `Makefile.fzfp`，例如 `make -f Makefile.fzfp check`。上游目标仍构建当前 fork 源码，原版对照必须从固定上游 worktree 构建。上游差异及理由见下表。

fzfp 的安全策略位于 `.github/SECURITY.md`。GitHub 官方说明的读取顺序为 `.github/`、根目录、`docs/`，因此可保留根目录上游政策而显示本项目政策；来源见[默认社区文件的优先级](https://docs.github.com/en/communities/setting-up-your-project-for-healthy-contributions/creating-a-default-community-health-file#about-default-community-health-files)。

## 上游更新

主分支为 `main`；日常修改照常新增提交，已发布标签固定不动。当前参考提交为 [bf4869b](https://github.com/junegunn/fzf/commit/bf4869b7fa7748d973eeb414da4dde0d7fe4cd3c)，版本标记为 fzf 0.75。保留 `upstream` remote，地址为 `https://github.com/junegunn/fzf.git`。

上游更新后，从 main 创建主题分支，将审阅过的 upstream 提交 merge 进该分支：

```sh
git fetch upstream --tags
git switch -c update/upstream main
git merge --no-ff <审阅过的新上游提交>
```

在主题分支处理冲突，同步更新 `Makefile.fzfp` 和终端测试的参考提交，从新上游树刷新六份保留文档、`README.fzf.md` 和根目录 `Makefile`，并更新上游差异清单。重点审查 Pattern 匹配器分派、缓存和边界评分接口。运行行为、性质、交互及性能检查，不以修改预期值掩盖回归；检查通过后再将主题分支合入 main。已发布标签始终固定在原提交，不随上游同步移动。

保留 LICENSE 中上游版权、NOTICE 和 `THIRD_PARTY_NOTICES.md` 中的全部依赖许可全文。升级依赖时同步核对许可附录。上游集成脚本仍默认调用 `fzf`，不能把保留脚本描述为已经适配 `fzfp`。

## 版本包

CLI 标签使用 `fzfp-v<版本>`，不覆盖上游标签，也不移动已发布标签。正式版本号需与 `main.go` 一致，并同步 `Makefile.fzfp`、手册、变更记录及安装示例。

`make -f Makefile.fzfp snapshot` 在本地生成四个平台包和校验文件，不上传。发布工具固定为 GoReleaser v2.20.0，需要比 CLI 更新的 Go；当前发布工作流固定 Go 1.27.4。缓存仍留在仓库内。

包涵盖 macOS / Linux × amd64 / arm64，名称为 `fzfp-<版本>-<系统>_<架构>.tar.gz`。每包包含程序、手册及完整许可文件。`checksums.txt` 包含四个 SHA256。快照版本带 `-snapshot`，不能将其哈希填入正式 formula。

## 发布顺序

1. 更新版本、两份 README 和相关中文文档，运行检查、快照和压缩包核验。匹配或终端行为有变化时，还需完整终端测试。
2. 审阅提交和标签后，仅推送目标分支及明确的发布标签。标签触发 `.github/workflows/release.yml`。
3. 工作流验证标签格式、源码版本和提交，构建四个平台包，检查许可和二进制内容，再上传 GitHub Release。Linux amd64 包还会在 runner 上实际执行并通过独立安装脚本验证。
4. 从正式 Release 下载资产核对校验和、版本和拼音搜索。macOS 包应在对应系统实测；只有交叉编译不等于运行验证。
5. 在 [zhiyozhao/homebrew-tap](https://github.com/zhiyozhao/homebrew-tap) 的 Actions 页面手动触发 **Sync fzfp formula**，核对生成的四个平台 URL/SHA，再验证安装、升级及手册。同步不按时钟自动运行。

工作流因自定义标签前缀跳过 GoReleaser 的内置标签校验之前，必须先完成自己的版本和提交验证。当前未配置 Apple 签名或公证；不要把包描述为已公证。

## Homebrew 与独立安装器

Homebrew 打包代码以 [tap 仓库](https://github.com/zhiyozhao/homebrew-tap) 为唯一来源：`scripts/fzfp/` 内保存模板、更新器和测试，`.github/workflows/sync-fzfp.yml` 仅支持手动触发。每次同步先运行更新器测试，再从正式 Release 下载并核对四个资产的 SHA，经过 Homebrew style/audit 后只提交 `Formula/fzfp.rb`。模板的零哈希不能直接用于安装；formula 只安装 `fzfp` 和手册，不安装上游 shell 或 Vim 文件。

修改 tap 后可手动触发同步并设 `dry_run=true`：它会测试生成器、下载校验正式资产，并确认渲染结果与现有 formula 一致，不修改或提交 formula。主仓库无需复制 tap 代码，也无需持有 tap 写凭据；工作流使用 tap 自己的 `GITHUB_TOKEN`。

`install-fzfp` 根据系统和架构选择资产、核对唯一 SHA256、在目标目录暂存后替换程序。失败不能破坏已有程序；不修改 shell 配置。下载校验依赖 GitHub HTTPS 提供的清单，并非独立签名认证。

主仓库打包验证入口为 `scripts/verify-binary.sh`、`scripts/verify-archives.sh` 和 `scripts/test-install.sh`。每个二进制都进行 strings 检查；包内完整许可附录必须与源码一致，不能只检查文件名存在。普通检查包含安装器回归；生成器回归属于 tap 的同步工作流。

## 上游差异清单

下表列出参考上游已存在、仍修改或删除的文件；新增的拼音包、测试、安装器和文档不属于这张表。

| 上游文件 | 状态 | 保留差异的理由 |
|---|---|---|
| `README.md` | 修改 | 中文首页；上游原文逐字节保存在 `README.fzf.md` |
| `main.go` | 修改 | fzfp 版本、帮助命令名、项目地址和嵌入手册 |
| `src/pattern.go` | 修改 | 将搜索模式接入拼音匹配，并仅复用安全的前缀候选缓存 |
| `go.mod` | 修改 | 增加 go-pinyin 读音数据及终端测试所用 creack/pty；保留上游模块路径 |
| `go.sum` | 修改 | 固定新增依赖校验和 |
| `.gitignore` | 修改 | 忽略项目缓存、构建产物和本地配置 |
| `.goreleaser.yml` | 修改 | 生成本项目四平台包，包含手册、完整许可和校验和 |
| `.github/CODEOWNERS` | 修改 | 本 fork 的代码审阅责任由维护者承担 |
| `.github/workflows/release.yml` | 修改 | 仅处理 fzfp tag 和资产，不运行上游签名、MSI 或 winget 发布 |
| `.github/workflows/linux.yml` | 删除 | 上游任务依赖其 lint/install 和 shell 集成流程；fork 使用独立检查及隔离对照 |
| `.github/workflows/macos.yml` | 删除 | 同样依赖上游安装流程，Ruby 版本也不满足完整测试要求 |
| `.github/workflows/winget.yml` | 删除 | 面向 junegunn.fzf，本项目不发布 Windows 包 |

六份保留文档、根目录 `Makefile`、LICENSE 以及通用社区/平台配置保持上游原文。新增代码版权及许可在 NOTICE；依赖许可全文随包提供。`README.fzf.md` 与参考上游 README 逐字节一致。

同步上游时核对新增文件和上述差异，逐项审查是否仍有必要：

```sh
git diff --stat bf4869b HEAD
git diff --name-status --diff-filter=MD bf4869b HEAD
git show bf4869b:Makefile | cmp - Makefile
git show bf4869b:README.md | cmp - README.fzf.md
```
