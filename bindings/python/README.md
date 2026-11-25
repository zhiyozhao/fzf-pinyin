# fzfp for Python

Python 3.10+, macOS 13+ or Linux. Calls the actual Go pinyin matcher in-process.
Build a native wheel from the complete checkout (Go and a C compiler required):

```sh
python3 -m pip wheel --no-deps --wheel-dir .cache/bindings ./bindings/python
python3 -m pip install .cache/bindings/fzfp-0.1.0-*.whl
```

Run installation in a virtual environment. Wheel consumers do not need Go.

```python
from fzfp import match, filter, Options, version

m = match("中国", "zhongguo")
assert m.matched and m.positions == [0, 1]
hits = filter(["北京", "-b-j", "-bj"], "bj", Options(scheme="history"))
for hit in hits:
    print(hit.index, hit.text, hit.score, hit.positions)
print(version())
```

Positions are ordinary Python string indexes, not UTF-8 byte indexes.
`utf8_positions` and `utf16_positions` are also available. One literal fuzzy
term is accepted; CLI extended syntax is not parsed. Mixed-scheme calls are
thread safe and serialized in Go. Exceptions own no native memory.
Full contract and distribution plan: [bindings documentation](../../docs/bindings.md).
