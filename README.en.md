[中文](README.md)

# fzf-pinyin

Add pinyin search to [fzf](https://github.com/junegunn/fzf), with the `fzfp` command.

- Search Chinese characters using full pinyin, first letters, or consonant initials; mix pinyin with Chinese characters.
- Candidates without Han characters retain fzf's matching and ordering behavior; see [validation and performance measurements](docs/testing.md).
- Pinyin supports fuzzy matching: skip characters in order while keeping the original output and highlights.

## Install

The current release is **0.2.0**, including the pinyin input, scoring, and performance improvements described below. Install with Homebrew on macOS or Linux, without Go:

```sh
brew install zhiyozhao/tap/fzfp
fzfp --version
# fzfp 0.2.0 (based on fzf 0.75)
```

You can also build from source with [Go 1.23+](https://go.dev/doc/install):

```sh
git clone https://github.com/zhiyozhao/fzf-pinyin.git
cd fzf-pinyin
go build -trimpath -o ./bin/fzfp .
./bin/fzfp --version
```

Place the resulting `bin/fzfp` in a directory on PATH. Language bindings still require a source build; see below.

Use `fzfp` as you would `fzf`; existing shell shortcuts do not switch automatically. See [installation](docs/install.md) for the installer and other options.

## Pinyin matching rules

Choose a spelling independently for each character. Bold text marks the source characters matched.

| Input | Matches | Rule |
|---|---|---|
| `beijing`, `bj` | **北京** | Full pinyin or first letters |
| `zhg`, `zguo` | **中国** | Consonant initials, or initials mixed with full syllables |
| `北jing`, `bei京` | **北京** | Mix Chinese characters and pinyin |
| `lvyou`, `lüyou` | **旅游** | Use v or ü |
| `nv`, `nü` | **女** | nu does not stand for nü |
| `lue`, `lve` | **略** | nue / nve likewise match 虐 |
| `ju`, `jv` | **居** | After j/q/x/y, use u or v for ü; yu / yv match 鱼 |
| `yinyue`, `yy` | **音乐** | Keep standard y/w spelling; in does not stand for yin |
| `en`, `ng` | **嗯** | e / n also work as initials; eng does not match |
| `beiji` | **北京**市 | The last matched character accepts an unfinished syllable, even within a longer line |
| `bj` | **北**小**京** | Fuzzy matching can skip source characters |
| `'bj` | **北京**市 | A leading quote requires adjacency; excludes 北小京 |
| `^bj$` | **北京** | Cover the whole line; excludes 去北京 and 北京市 |
| `chongqing`, `zhongqing` | **重庆** | Accept all dictionary readings of each character |

Only the last matched character accepts an unfinished syllable: `zhonggu` matches 中国, while `zhoguo` does not. Matching cannot start inside a syllable: `hong` does not match 中 (zhong).

Pinyin ignores case; literal Latin text follows fzf's case settings. Tones, double-pinyin schemes, and spelling correction are not supported. Readings are not selected by phrase context. See [pinyin input](docs/pinyin.md) and [search syntax](docs/syntax.md) for details, exclusions, fields, and multiple conditions.

A line matched using any pinyin loses one point in total, regardless of character count. When other scoring factors are equal, literal matches rank first.

## Use from other languages

Swift, Python, and JavaScript (Node.js) can match pinyin and return scores and positions in the original text. Use `match` for one string, `filter` for a list, or `Index` to search the same list repeatedly. Queries are single fuzzy terms; CLI operators such as quotes, `^`, and `$` are not parsed.

These packages have **not been published** as a SwiftPM release or to PyPI or npm. Build from a complete source checkout for now. Each language guide covers prerequisites and builds; see the [bindings documentation](docs/bindings.md) for the full API.

[Swift](bindings/swift/README.md): build first, then add a local package dependency to your Swift project.

```swift
import Fzfp

print(try Fzfp.match("中国", query: "zhongguo").positions) // [0, 1]
```

[Python](bindings/python/README.md): build a wheel from source and install it in a virtual environment.

```python
from fzfp import match

print(match("中国", "zhongguo").positions)  # [0, 1]
```

[JavaScript](bindings/javascript/README.md): build first, then run this example from the repository root. Requires Node.js 22+; browsers are not supported.

```js
const { match } = require('./bindings/javascript');

console.log(match('中国', 'zhongguo').positions); // [0, 1]
```

Supporting documentation is in Chinese: [development](CONTRIBUTING.md) · [changelog](CHANGELOG.fzfp.md). See also [upstream fzf documentation](README.fzf.md), [MIT](LICENSE), and [third-party licenses](THIRD_PARTY_NOTICES.md).
