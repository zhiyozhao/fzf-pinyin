[开发指南](../CONTRIBUTING.md)

# 维护与发布

面向更新上游、制作版本包和维护 Homebrew formula 的维护者。

## 与上游文件的边界

上游已有文件尽量保持原样；本项目文档单独新增。上游的 `ADVANCED.md`、`README-VIM.md`、`CHANGELOG.md`、`BUILD.md`、`RELEASE.md`、`SECURITY.md` 保持参考版本原文。fzfp 的构建说明在 [CONTRIBUTING](../CONTRIBUTING.md)，发布说明在本文，版本变化在 [CHANGELOG.fzfp.md](../CHANGELOG.fzfp.md)。

必要差异及每个文件的保留理由见[上游文件审查](upstream-diff.md)。根目录 Makefile 保留 fzfp 的默认构建和检查入口，`Makefile.fzf` 是参考版本 Makefile 的逐字节副本。它仅作上游参考，不是独立源码树，不能保证在本 fork 上执行它会生成原版 fzf。

fzfp 的安全策略位于 `.github/SECURITY.md`。GitHub 官方说明的读取顺序为 `.github/`、根目录、`docs/`，因此可保留根目录上游政策而显示本项目政策；来源见[默认社区文件的优先级](https://docs.github.com/en/communities/setting-up-your-project-for-healthy-contributions/creating-a-default-community-health-file#about-default-community-health-files)。

## 上游更新

主分支为 `main`；上游完整历史之后保留一个汇总本项目差异的提交。当前参考提交为 [bf4869b](https://github.com/junegunn/fzf/commit/bf4869b7fa7748d973eeb414da4dde0d7fe4cd3c)，版本标记为 fzf 0.75。保留 `upstream` remote，地址为 `https://github.com/junegunn/fzf.git`。

上游更新后，在主题分支把本项目提交 rebase 到审阅过的新上游提交之上；需要补充修复时，用 amend 或 squash 将本项目差异保持为一个提交：

```sh
git fetch upstream --tags
git switch -c update/upstream main
git rebase --onto <新上游提交> <旧上游参考提交>
```

同步更新 Makefile 和终端测试的参考提交，从新上游树刷新六份保留文档、`README.fzf.md` 和 `Makefile.fzf`，并更新上游差异清单。重点审查 Pattern 匹配器分派、缓存和边界评分接口。发布前运行行为、性质、交互及性能检查，不以修改预期值掩盖回归。备份并核对远端状态后，以显式 `--force-with-lease` 更新 main；发布标签固定在原提交，不随 rebase 移动。已有本地分支应先备份，再对齐新的 main。

保留 LICENSE 中上游版权、NOTICE、第三方声明和 `docs/licenses/`。上游集成脚本仍默认调用 `fzf`，不能把保留脚本描述为已经适配 `fzfp`。

## 版本包

CLI 标签使用 `fzfp-v<版本>`，不覆盖上游标签，也不移动已发布标签。正式版本号需与 `main.go` 一致，并同步 Makefile、手册、变更记录及安装示例。

`make snapshot` 在本地生成四个平台包和校验文件，不上传。发布工具固定为 GoReleaser v2.20.0，需要比 CLI 更新的 Go；当前发布工作流固定 Go 1.27.4。缓存仍留在仓库内。

包涵盖 macOS / Linux × amd64 / arm64，名称为 `fzfp-<版本>-<系统>_<架构>.tar.gz`。每包包含程序、手册及完整许可文件。`checksums.txt` 包含四个 SHA256。快照版本带 `-snapshot`，不能将其哈希填入正式 formula。

## 发布顺序

1. 更新版本、两份 README 和相关中文文档，运行检查、快照和压缩包核验。匹配或终端行为有变化时，还需完整终端测试。
2. 审阅提交和标签后，仅推送目标分支及明确的发布标签。标签触发 `.github/workflows/release.yml`。
3. 工作流验证标签格式、源码版本和提交，构建四个平台包，检查许可和二进制内容，再上传 GitHub Release。Linux amd64 包还会在 runner 上实际执行并通过独立安装脚本验证。
4. 从正式 Release 下载资产核对校验和、版本和拼音搜索。macOS 包应在对应系统实测；只有交叉编译不等于运行验证。
5. 确认 tap 已生成有效 formula，并验证安装、升级及手册。

工作流因自定义标签前缀跳过 GoReleaser 的内置标签校验之前，必须先完成自己的版本和提交验证。当前未配置 Apple 签名或公证；不要把包描述为已公证。

## Homebrew 与独立安装器

`packaging/homebrew/fzfp.rb` 是包含零哈希的模板，不能直接发布。生成器从已下载的正式资产核对四个哈希后，才写出有效 formula。tap 同步工作流和生成器在同一目录；同步任务只更新 `Formula/fzfp.rb`。formula 只安装 `fzfp` 和手册，不安装上游 shell 或 Vim 文件。

`install-fzfp` 根据系统和架构选择资产、核对唯一 SHA256、在目标目录暂存后替换程序。失败不能破坏已有程序；不修改 shell 配置。下载校验依赖 GitHub HTTPS 提供的清单，并非独立签名认证。

打包验证入口包括 `scripts/verify-binary.sh`、`scripts/verify-archives.sh`、`scripts/test-install.sh` 和 `packaging/homebrew/test-update.rb`；普通检查已包含后两项。
