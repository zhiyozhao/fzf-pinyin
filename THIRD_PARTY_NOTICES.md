# Third-party notices

The original fzf code is Copyright (c) 2013–2026 Junegunn Choi, under MIT.
The original notice is retained unchanged in [LICENSE](LICENSE). The extensions' copyright and MIT licensing declaration are in [NOTICE](NOTICE).
Original documentation and build rules remain in `README.fzf.md` and `Makefile.fzf`.

Pinyin code and the generated dictionary are from **mozillazg/go-pinyin v0.21.0**, MIT, Copyright (c) 2016 mozillazg.
The dictionary's upstream source is **mozillazg/pinyin-data**, also MIT, Copyright (c) 2016 mozillazg.
The pinned artifact is go-pinyin's generated snapshot; no unpinned data is downloaded at runtime.
The two upstream projects publish the same MIT notice; its full text is included in `docs/licenses/github.com_mozillazg_go-pinyin.LICENSE`.
The zero-initial y/w transformation follows the same spelling conventions documented by go-pinyin.

Additional Go modules and their license texts (versions are pinned in `go.mod` / `go.sum`):

| Module | Version | License |
|---|---|---|
| github.com/charlievieth/fastwalk | v1.0.14 | MIT |
| github.com/creack/pty | v1.1.24 | MIT; test only |
| github.com/gdamore/tcell/v2 | v2.9.0 | Apache-2.0; platform / build-tag dependent |
| github.com/gdamore/encoding | v1.0.1 | Apache-2.0; platform / build-tag dependent |
| github.com/junegunn/go-shellwords | 2aa3b3277741 | MIT |
| github.com/lucasb-eyer/go-colorful | v1.2.0 | MIT |
| github.com/mattn/go-isatty | v0.0.24 | MIT |
| github.com/mattn/go-runewidth | v0.0.16 | MIT |
| github.com/mozillazg/go-pinyin | v0.21.0 | MIT |
| github.com/rivo/uniseg | v0.4.7 | MIT |
| golang.org/x/sys | v0.35.0 | BSD-3-Clause |
| golang.org/x/term | v0.34.0 | BSD-3-Clause |
| golang.org/x/text | v0.28.0 | BSD-3-Clause |

Unmodified full license texts are in [docs/licenses](docs/licenses/), copied from the pinned modules.
No Apache-licensed dependency files were modified. Rime configuration was consulted as research on common abbreviation behavior; no Rime code or dictionary is incorporated.
The local gitleaks development tool is not linked into or distributed with fzfp.
The optional upstream E2E target uses minitest 5.25.4 (MIT, Copyright (c) Ryan Davis, seattle.rb),
installed only in `.cache/gems`; its original notice remains in the gem's `README.rdoc`.
It is not linked into or distributed with fzfp.
