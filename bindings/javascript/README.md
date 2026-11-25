# fzfp for Node.js

Shared Go pinyin core through N-API (Node 22+, macOS/Linux). Native package;
browser/WASM support is deferred. No npm packages or compilers are needed at
runtime when a matching platform binary is installed.

```js
const { match, filter, Index } = require('fzfp');
console.log(match('中国', 'zhongguo').positions); // [0, 1]
console.log(filter(['北京', 'fooBar'], 'bj'));
const titles = ['北京', '中国', 'fooBar'];
const index = new Index(titles, { scheme: 'path', workers: 4 });
try {
  for (const hit of index.find('bj', { limit: 50, positions: true })) {
    console.log(titles[hit.index], hit.score, hit.alignment.utf16Positions);
  }
} finally { index.close(); }
```

TypeScript definitions ship in the package. Index owns a copied immutable
corpus; retain it across queries. `limit: null`/omitted returns all, zero returns
none, negative limits throw. Rank-only hits have `alignment: null`. Alignment
positions and bounds count original Unicode scalars; `utf16Positions` can be
used for JavaScript string slicing. No grapheme expansion. Lone surrogates are
rejected. `match`/`filter` retain exact bounds and default to positions enabled.

Queries are literal fuzzy terms, with the same case, normalization and three
scheme tie rules as Swift/Python; extended fzf operators are not parsed.
Calls are synchronous and block the Node event loop while Go batch workers
run. Put the Index in a `worker_threads` worker for UI/server responsiveness.
There is no implicit native cancellation, async callback or streaming injector.
Close waits naturally for synchronous calls in its owning JS isolate; close is
idempotent, and a finalizer is a fallback. Index objects cannot be transferred
between isolates. Global scheme selection serializes independent requests.

From the checkout: `bash scripts/build-javascript.sh`, then
`node --test bindings/javascript/test.js`. Requires existing Go, C compiler and
Node headers (`FZFP_NODE_INCLUDE` can point to an existing installation).
Build stages main and platform npm tarballs in `.cache/bindings/npm`.
The main package resolves a platform optional dependency, as in esbuild;
there is no install-time download, build script or silent source fallback.
Names/scope ownership and the complete native platform matrix must be approved
before publication. Only the host tarball is built locally.
