[English](README.en.md)

# fzf-pinyin

给 [fzf](https://github.com/junegunn/fzf) 加上拼音搜索，命令叫 `fzfp`。

- 搜索汉字时支持全拼、首字母、声母，也能混用汉字和拼音。
- 不含汉字的候选保留 fzf 的匹配和排序行为，见[验证与性能测量](docs/testing.md)。
- 拼音也支持模糊搜索，可以按顺序跳字，输出和高亮仍是原文。

## 安装

当前正式版为 **0.2.1**，包含下面的拼音输入、评分和性能改进。macOS / Linux 可用 Homebrew 安装，无需 Go：

```sh
brew install zhiyozhao/tap/fzfp
fzfp --version
# fzfp 0.2.1 (based on fzf 0.75)
```

也可从源码构建，需要 [Go 1.23+](https://go.dev/doc/install)：

```sh
git clone https://github.com/zhiyozhao/fzf-pinyin.git
cd fzf-pinyin
go build -trimpath -o ./bin/fzfp .
./bin/fzfp --version
```

把生成的 `bin/fzfp` 放到 PATH 中的目录即可使用。语言绑定仍需要从源码构建，见下文。

用法和 fzf 相同，把命令换成 `fzfp` 即可；已有 shell 快捷键不会自动切换。安装脚本和其他方式见[安装说明](docs/install.md)。

## 拼音搜索规则

每个字可独立选择一种写法；下面加粗的是匹配的原文字。

| 输入 | 能匹配 | 规则 |
|---|---|---|
| `beijing`、`bj` | **北京** | 全拼或首字母 |
| `zhg`、`zguo` | **中国** | 声母，或缩写与全拼混用 |
| `北jing`、`bei京` | **北京** | 汉字和拼音混用 |
| `lvyou`、`lüyou` | **旅游** | ü 可以写成 v |
| `nv`、`nü` | **女** | 不用 nu 代替 nü |
| `lue`、`lve` | **略** | 同样接受 nue / nve 匹配“虐” |
| `ju`、`jv` | **居** | j/q/x/y 后的 ü 可写成 u 或 v，如 yu / yv 匹配“鱼” |
| `yinyue`、`yy` | **音乐** | 按正常 y/w 拼写，不用 in 代替 yin |
| `en`、`ng` | **嗯** | 也可用首字母 e / n，不接受 eng |
| `beiji` | **北京**市 | 最后匹配的字可没打完，不必是整行最后一字 |
| `bj` | **北**小**京** | 默认允许按顺序跳字 |
| `'bj` | **北京**市 | 前置单引号要求连续，不匹配“北小京” |
| `^bj$` | **北京** | 覆盖整行，不匹配“去北京”或“北京市” |
| `chongqing`、`zhongqing` | **重庆** | 多音字接受字典中的全部读音 |

只有最后匹配的字允许未完成拼音：`zhonggu` 能匹配“中国”，`zhoguo` 不能；`hong` 也不能从“中”的 zhong 中间开始匹配。

拼音不分大小写；实际英文字母仍遵守 fzf 的大小写设置。不支持声调、双拼和拼写纠错，多音字不按词组消歧。更多写法见[拼音指南](docs/pinyin.md)，排除、字段及多条件查询见[搜索语法](docs/syntax.md)。

一行里只要用了拼音，总共扣 1 分，不随字数增加；其余评分条件相同时，直接匹配原文的结果排在前面。

## 在其他语言里使用

Swift、Python 和 JavaScript（Node.js）都能匹配拼音，并返回分数和原文中的匹配位置。支持单条匹配、列表筛选，以及重复搜索同一列表的 `Index`。查询是一个模糊搜索词，不解析 CLI 的引号、`^`、`$` 等语法。

Swift 0.2.1 支持通过 SwiftPM 直接引用，发布资产和 tag 就绪后无需 Go；[发布流程](docs/swift-release.md)说明发布前的本地验证方式。Python 和 npm 包本轮不发布，仍从完整仓库源码构建。完整 API 见[绑定文档](docs/bindings.md)。

[Swift](bindings/swift/README.md)：`.package(url: "https://github.com/zhiyozhao/fzf-pinyin", exact: "0.2.1")`，产品名为 `Fzfp`。`Index.find(exact: true)` 支持连续拼音匹配。

```swift
import Fzfp

print(try Fzfp.match("中国", query: "zhongguo").positions) // [0, 1]
```

[Python](bindings/python/README.md)：从源码构建 wheel，并安装到虚拟环境。

```python
from fzfp import match

print(match("中国", "zhongguo").positions)  # [0, 1]
```

[JavaScript](bindings/javascript/README.md)：构建后，在仓库根目录运行此例；需要 Node.js 22+，不支持浏览器。

```js
const { match } = require('./bindings/javascript');

console.log(match('中国', 'zhongguo').positions); // [0, 1]
```

[开发指南](CONTRIBUTING.md) · [变更记录](CHANGELOG.fzfp.md) · [上游 fzf 文档](README.fzf.md) · [MIT](LICENSE) · [第三方许可](THIRD_PARTY_NOTICES.md)
