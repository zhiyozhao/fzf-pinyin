[首页](README.md)

# 变更记录

这里只记录 fzfp 的发布变化。上游历史见 [fzf 变更记录](https://github.com/junegunn/fzf/blob/bf4869b7fa7748d973eeb414da4dde0d7fe4cd3c/CHANGELOG.md)。

## 0.2.0

### 中文

- 拼音输入改用常见输入法拼写，补充 ü 的 u/v 写法和“嗯”等字的编码；不再接受省略 y/w 的旧别名，具体规则见[拼音指南](docs/pinyin.md)。
- 拼音匹配整行只扣 1 分，不按字数或拼写方式累加；改善中英混排的排序，部分结果顺序会变化。
- 提升大列表的拼音搜索速度，降低长行搜索的内存占用。
- 新增 Swift、Python 和 JavaScript（Node.js）源码接口，支持原文位置、列表筛选与重复搜索。CLI 为 0.2.0；绑定包版本保持 0.1.0，尚未通过 SwiftPM release、PyPI 或 npm 发布，需要从源码构建。
- 更新输入规则、源码安装和语言绑定说明。继续提供 macOS/Linux 的 amd64/arm64 预编译包、手册及 SHA256 清单。

### English

- Follow familiar pinyin keyboard spellings, including u/v forms for ü and additional codes for characters such as 嗯. Old aliases that omit initial y/w are no longer accepted; see the [pinyin guide](docs/pinyin.md).
- Apply one pinyin penalty per matched line, rather than per character or spelling type. Improve mixed Chinese/Latin ranking; some result orders change.
- Speed up pinyin searches over large lists and reduce memory use for long lines.
- Add Swift, Python, and JavaScript (Node.js) source interfaces for original-text positions, list filtering, and repeated searches. The CLI is 0.2.0; binding package versions remain 0.1.0 and have not been published as a SwiftPM release or to PyPI or npm. Build them from source.
- Update input, source installation, and language binding guides. Continue shipping macOS/Linux amd64/arm64 archives, manuals, and SHA256 checksums.

## 0.1.0

- 首个正式版本，基于 fzf 0.75（上游参考提交 `bf4869b`）。
- 提供独立 `fzfp` 命令和 Go 拼音匹配库。
- 支持全拼、首字母、声母、混合输入和末字未完成拼音。
- 拼音不分大小写，多音字接受字典中的全部读音。
- 搜索语法、字段和高亮按原文工作；保留无汉字候选的原版匹配行为。
- 提供 macOS 和 Linux 的 amd64、arm64 预编译包、手册、许可证及 SHA256 清单。
- 提供无需 sudo、校验 SHA256 的独立安装脚本，Homebrew 直接安装预编译包。
- 包含拼音用法、搜索语法、评分设计、Go 库、开发维护和上游差异文档。
