# fzfp for Python

Python 3.10+, macOS 13+ or Linux. Calls the actual Go pinyin matcher in-process.
Build a native wheel from the complete checkout (Go and a C compiler required):

```sh
python3 -m pip wheel --no-deps --wheel-dir .cache/bindings ./bindings/python
python3 -m pip install .cache/bindings/fzfp-0.2.1-*.whl
```

Run installation in a virtual environment. Wheel consumers do not need Go.

```python
from fzfp import match, filter, Index, Options, version

m = match("中国", "zhongguo")
assert m.matched and m.positions == [0, 1]
hits = filter(["北京", "-b-j", "-bj"], "bj", Options(scheme="history"))
for hit in hits:
    print(hit.index, hit.text, hit.score, hit.positions)
print(version())

titles = ["北京", "中国", "fooBar"]
with Index(titles, Options(scheme="path"), workers=4) as index:
    for hit in index.find("bj", limit=50, positions=True):
        print(titles[hit.index], hit.score, hit.positions)
```

Positions are ordinary Python string indexes, not UTF-8 byte indexes.
`utf8_positions` and `utf16_positions` are also available. One literal fuzzy
term is accepted; CLI extended syntax is not parsed. Mixed-scheme calls are
thread safe and serialized in Go. Exceptions own no native memory.
Each batch uses up to four native workers by default; ctypes releases the GIL
while Go runs. Index copies the corpus once and reuses query scratch. It has
an idempotent close/context manager, synchronized with in-flight queries.
`limit=None` returns all, zero returns none, negative raises ValueError.
Rank-only hits have `alignment=None`; positions opt in to a full Match and
Unicode offsets. Use a persistent Index for repeated queries. Async callers
may use `asyncio.to_thread(index.find, ...)`; cancelling the await does not
interrupt native work, and close waits for the call to finish.
Full contract and distribution plan: [bindings documentation](../../docs/bindings.md).
