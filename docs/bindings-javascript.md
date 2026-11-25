# JavaScript implementation choice

Decision date: 2026-11-25. First version targets **Node 22+ with a C N-API
addon statically linking the same Go C ABI**, not a new JS matching algorithm.
TypeScript declarations ship alongside CommonJS JS. Browser/WASM is deferred.

| Route | Performance/threading | Size | Browser | npm distribution | Maintenance |
|---|---|---|---|---|---|
| Node N-API + Go C archive (chosen) | same typed ABI, native batch workers, blocking host call; use worker_threads | measured host addon ~4.6 MB raw/~1.6 MB tgz, main ~10.5 KB tgz | no | main + OS/CPU optionalDependencies, no install scripts/downloads | one C ABI, small C addon, native CI per OS/CPU |
| Standard Go js/wasm | host crossings/memory and JS event loop constraints; native multicore worker strategy cannot be assumed to carry over | Go runtime + dictionary + wasm_exec.js required; not measured here | yes with JS glue/async loading | potentially one portable wasm asset plus loader | separate handle/transport adapter because cgo C ABI is unavailable; extra browser lifecycle/thread tests |
| Both | can provide native Node plus browser entrypoint | two asset families | yes | conditional exports + native packages + WASM assets | two build/test/runtime paths; premature for the first fixed-corpus API |

N-API's ABI stability avoids compiling separately for each supported Node major;
it does **not** remove CPU/OS/libc compatibility constraints. NAPI_VERSION=8 is
used; Node >=22 is the current tested host contract (also uses String.isWellFormed).
There is no node-gyp or downloaded compiler/header requirement for this local
build: existing Node development headers, clang/cc and Go are sufficient.
Ordinary wheel/npm consumers need no build toolchain.

The platform package names are placeholders pending registry/scope approval:
`@fzfp/native-darwin-arm64`, `@fzfp/native-darwin-x64`,
`@fzfp/native-linux-arm64`, `@fzfp/native-linux-x64`. Linux packages declare
glibc, not musl. `scripts/build-javascript.sh` builds the host only and stages
two tarballs using `npm pack --ignore-scripts`, without publishing. The main
package resolves an exact-version optional platform dependency, following
esbuild's packaging shape. Unlike esbuild's full installer, there is currently
no postinstall network fallback: missing/disabled optional dependencies fail
explicitly. CI prepares macOS native runners and Linux manylinux 2.28 native
containers; only macOS arm64 was executed locally. A release cannot claim
Linux/x64 runtime coverage from this local host run.

`scripts/test-javascript-package.sh` extracts the actual packed tarballs into
an external temporary consumer module tree and verifies package resolution.
It deliberately does not run registry installation against unpublished names.
Release acceptance still requires real registry install on the full matrix.

Sources (Node/Go docs and esbuild manifest checked this turn; npm reference links):

- [Node-API ABI stability](https://nodejs.org/api/n-api.html#implications-of-abi-stability)
- [Node addons and Node-API](https://nodejs.org/api/addons.html#node-api)
- [Go WebAssembly](https://go.dev/wiki/WebAssembly): standard js/wasm build and required JS runtime glue.
- [cgo](https://pkg.go.dev/cmd/cgo): native C bridge/build constraints.
- [esbuild npm manifest](https://github.com/evanw/esbuild/blob/main/npm/esbuild/package.json): exact-version OS/CPU optional dependency packages.
- [npm optionalDependencies](https://docs.npmjs.com/cli/v11/configuring-npm/package-json#optionaldependencies)
- [npm os/cpu/libc](https://docs.npmjs.com/cli/v11/configuring-npm/package-json#libc)

The JS API does not hide blocking work behind a Promise. `Index.find` owns no
background scheduler; an application can create the Index in worker_threads
and send query/result messages. Results use typed C rows internally, then JS
objects with index/score and optional alignment. No JSON in the typed hot path,
no per-candidate callback, no cross-isolate transferable Go handle.
