[中文](README.md)

# fzf-pinyin

Add pinyin search to [fzf](https://github.com/junegunn/fzf), with the `fzfp` command.

- Search Chinese characters using full pinyin, first letters, or consonant initials; mix pinyin with Chinese characters.
- Non-Han results and ordering match fzf in 260 differential checks; measured runtime on 100,000 lines was 1.02–1.14× fzf's. See [validation and performance](docs/benchmarks.md).
- Pinyin supports fuzzy matching too: skip source characters in order, while keeping the original output and highlights.

## Install

Homebrew on macOS or Linux; Go is not required:

```sh
brew install zhiyozhao/tap/fzfp
```

On Linux, the installer supports x86_64 / arm64 and verifies SHA256 before installing to `~/.local/bin`:

```sh
curl -fL https://raw.githubusercontent.com/zhiyozhao/fzf-pinyin/fzfp-v0.1.0/install-fzfp -o install-fzfp
sh install-fzfp --version 0.1.0
```

Ensure the destination is on PATH. Use `fzfp` as you would `fzf`; both can stay installed, and existing shell shortcuts do not switch automatically. See [installation](docs/install.md) for details.

## Pinyin matching rules

Choose a spelling independently for each character. Bold text marks the source characters matched.

| Input | Matches | Rule |
|---|---|---|
| `beijing`, `bj` | **北京** | Full pinyin or first letters |
| `zhg`, `zguo` | **中国** | Consonant initials, or initials mixed with full syllables |
| `北jing`, `bei京` | **北京** | Mix Chinese characters and pinyin |
| `lvyou`, `lüyou` | **旅游** | Use v or ü |
| `beiji` | **北京**市 | The last matched character accepts an unfinished syllable, even within a longer line |
| `bj` | **北**小**京** | Fuzzy matching can skip source characters |
| `'bj` | **北京**市 | A leading quote requires adjacency; excludes 北小京 |
| `^bj$` | **北京** | Cover the whole line; excludes 去北京 and 北京市 |
| `chongqing`, `zhongqing` | **重庆** | Accept all dictionary readings of each character |

Only the last matched character accepts an unfinished syllable: `zhonggu` matches 中国, while `zhoguo` does not. Matching cannot start inside a syllable: `hong` does not match 中 (zhong).

Pinyin ignores case; literal Latin text follows fzf's case settings. Tones, double-pinyin schemes, and spelling correction are not supported. Readings are not selected by phrase context. See [pinyin input](docs/pinyin.md) and [search syntax](docs/syntax.md) for details, exclusions, fields, and multiple conditions.

Supporting documentation is in Chinese: [development](CONTRIBUTING.md) · [changelog](CHANGELOG.fzfp.md). See also [upstream fzf documentation](README.fzf.md), [MIT](LICENSE), and [third-party licenses](THIRD_PARTY_NOTICES.md).
