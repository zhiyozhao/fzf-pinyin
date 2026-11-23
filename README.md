[English](README.en.md)

# fzf-pinyin

给 [fzf](https://github.com/junegunn/fzf) 加上拼音搜索，命令叫 `fzfp`。

- 搜索汉字时支持全拼、首字母、声母，也能混用汉字和拼音。
- 纯英文候选的结果和排序在 260 组对拍中与 fzf 一致；10 万行基准耗时为 fzf 的 1.02–1.14 倍，见[验证与性能](docs/benchmarks.md)。
- 拼音也支持模糊搜索：可以按顺序跳过原文中的字，保留原文输出和高亮。

## 安装

macOS / Linux 可用 Homebrew，无需 Go：

```sh
brew install zhiyozhao/tap/fzfp
```

Linux 也可用安装脚本，支持 x86_64 / arm64，校验 SHA256 后安装到 `~/.local/bin`：

```sh
curl -fL https://raw.githubusercontent.com/zhiyozhao/fzf-pinyin/fzfp-v0.1.0/install-fzfp -o install-fzfp
sh install-fzfp --version 0.1.0
```

安装后确保目录在 PATH 中。`fzfp` 可与 `fzf` 并存；用法相同，把命令换成 `fzfp` 即可，已有 shell 快捷键不会自动切换。更多见[安装说明](docs/install.md)。

## 拼音搜索规则

每个字可独立选择一种写法；下面加粗的是匹配的原文字。

| 输入 | 能匹配 | 规则 |
|---|---|---|
| `beijing`、`bj` | **北京** | 全拼或首字母 |
| `zhg`、`zguo` | **中国** | 声母，或缩写与全拼混用 |
| `北jing`、`bei京` | **北京** | 汉字和拼音混用 |
| `lvyou`、`lüyou` | **旅游** | ü 可以写成 v |
| `beiji` | **北京**市 | 最后匹配的字可没打完，不必是整行最后一字 |
| `bj` | **北**小**京** | 默认允许按顺序跳字 |
| `'bj` | **北京**市 | 前置单引号要求连续，不匹配“北小京” |
| `^bj$` | **北京** | 覆盖整行，不匹配“去北京”或“北京市” |
| `chongqing`、`zhongqing` | **重庆** | 多音字接受字典中的全部读音 |

只有最后匹配的字允许未完成拼音：`zhonggu` 能匹配“中国”，`zhoguo` 不能；`hong` 也不能从“中”的 zhong 中间开始匹配。

拼音不分大小写；实际英文字母仍遵守 fzf 的大小写设置。不支持声调、双拼和拼写纠错，多音字不按词组消歧。更多写法见[拼音指南](docs/pinyin.md)，排除、字段及多条件查询见[搜索语法](docs/syntax.md)。

[开发指南](CONTRIBUTING.md) · [变更记录](CHANGELOG.fzfp.md) · [上游 fzf 文档](README.fzf.md) · [MIT](LICENSE) · [第三方许可](THIRD_PARTY_NOTICES.md)
