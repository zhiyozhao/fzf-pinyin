[首页](../README.md)

# 安装与常见问题

## 选择安装方式

Homebrew 和安装脚本目前提供 **0.1.0**，落后于 `main`：不包含新的输入法拼写、整行只扣 1 分的评分规则、性能改进和语言绑定。首页及[拼音指南](pinyin.md)描述的是 `main`，要使用这些功能请从源码构建。

Homebrew 支持 macOS 和 Linux，安装 `fzfp` 及其手册；独立安装脚本支持两种系统，但只安装可执行文件。两种方式都提供 x86_64（amd64）和 arm64（aarch64）版本，无需安装 Go。

Linux 预编译包是静态程序，不依赖 glibc 或 musl。尚无 32 位 ARM 包。macOS 预编译包未做 Apple 公证。

## 从源码安装当前版本

需要 [Go 1.23+](https://go.dev/doc/install)：

```sh
git clone https://github.com/zhiyozhao/fzf-pinyin.git
cd fzf-pinyin
go build -trimpath -o ./bin/fzfp .
./bin/fzfp --version
```

把 `bin/fzfp` 放到 PATH 中的目录。版本输出中的 `0.1.0` 尚未递增，请同时以源码提交号确认构建版本。后续更新可在没有本地修改的 checkout 中拉取 `main`，然后重新构建：

```sh
git pull --ff-only origin main
go build -trimpath -o ./bin/fzfp .
```

这里只构建命令行程序；Swift / Python / JavaScript 的构建入口见[绑定文档](bindings.md)。

## Homebrew

```sh
brew install zhiyozhao/tap/fzfp
fzfp --version
```

升级：

```sh
brew upgrade zhiyozhao/tap/fzfp
```

版本输出会包含 `fzfp` 自身版本和所基于的 fzf 版本。当前正式版为 0.1.0，基于 fzf 0.75。

## 安装脚本

在可写目录下载脚本，可以先查看内容再执行：

```sh
curl -fL https://raw.githubusercontent.com/zhiyozhao/fzf-pinyin/fzfp-v0.1.0/install-fzfp -o install-fzfp
sh install-fzfp --version 0.1.0
```

默认目录是 `~/.local/bin`。在当前终端启用这个目录：

```sh
export PATH="$HOME/.local/bin:$PATH"
fzfp --version
```

要永久生效，把这行 PATH 设置加入自己所用 shell 的启动配置。脚本不会替你修改配置。

自选目录，例如当前目录下的 `bin`：

```sh
sh install-fzfp --version 0.1.0 --bin-dir ./bin
./bin/fzfp --version
```

省略版本参数会安装 0.1.0；要安装最新正式版，请显式选择 `latest`，重复运行可更新已有程序：

```sh
sh install-fzfp --version latest --bin-dir ./bin
```

需要 curl、tar，以及 sha256sum 或 shasum。下载或校验失败时，原有程序保留。卸载时删除安装目录中的 `fzfp` 即可；Homebrew 安装请用 Homebrew 卸载。

## 手动下载

从 [Releases](https://github.com/zhiyozhao/fzf-pinyin/releases) 下载与系统、CPU 对应的压缩包及同版 `checksums.txt`，核对 SHA256 后解压，把 `fzfp` 放到 PATH 中的目录。包内附有 `fzfp.1` 和许可证文件。

## 装好了却找不到，或没有拼音搜索

先运行 `fzfp --version`，确认调用的是 `fzfp`。直接运行 `fzf` 或已有的 shell 快捷键，不会自动切换到这个程序。

安装脚本默认目录必须在 PATH 中。上述源码构建生成的是 `bin/fzfp`；仓库里的上游 `install` 脚本安装的是 `fzf`，请勿混用。
