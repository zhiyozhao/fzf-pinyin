# Swift / Python / Node bindings (typed ABI 2 + JSON ABI 1, preview 0.1.0)

One Go implementation serves the CLI, Swift, Python and Node. All binding files live
in this repository so a scoring change and its shared test vectors can be
reviewed together. No existing upstream file is modified by the bindings.
Measured batch results and reproduction commands: [performance](bindings-performance.md).

## Contract

`match(text, query, options)` matches one **literal fuzzy term**, using pinyin
FuzzyMatchV2. It does not parse fzf extended-query operators, whitespace-separated
terms, fields, ANSI, smart case, or interactive CLI options. Default: case
insensitive, accent normalization on, forward matching, positions on, default
scheme. An accented query disables accent folding, following fzf's term rules.
Use `case_sensitive` explicitly; host locale and case folding are never used.

`filter(candidates, query, options)` matches and sorts a whole batch in Go,
retaining duplicate entries and their zero-based input indexes. Rank follows
fzf's default tie rules for the selected scheme: descending score, then
`default`: trimmed rune length; `path`: filename-relative match start, then
trimmed rune length; `history`: input order. The final tie is always input
order. An empty query preserves input order in all schemes, like CLI filter.
Score and tie keys saturate to 65535 as in fzf. No CLI flags such as
`--tiebreak`, `--nth`, `--tac`, or `--ansi` are implied. Empty query matches all;
empty result is an empty list in both wrappers.

Result: `matched`, `score`, `start`, `end`, `positions`, `utf8_positions`,
`utf16_positions`. Start/end and positions count original Unicode scalars
(Go runes, Python str indexes), end is exclusive. No match: score 0, start/end
-1 and empty arrays. Positions are sorted ascending, unique, and never count
expanded pinyin letters. UTF offsets point to the same scalars. Positions off
returns empty position arrays while preserving start/end and scores. The facade
uses exact traced bounds even when positions are off; this deliberately costs
more than the upstream no-position approximation and keeps path ranking stable.
Swift
offers `utf16Ranges(in:)` for NSRange; this is scalar alignment, not grapheme
cluster expansion. Always use the original unmodified text for highlighting.
Embedded NUL and newlines in strings are supported. Invalid UTF-8 is rejected
at the ABI boundary; Python surrogate-only strings raise UnicodeEncodeError.

## C ABI and memory

The current language convenience APIs use additive **typed ABI 2**, as does
`Index(candidates).find(query, limit, positions)`. Construct an Index once;
it copies UTF-8 candidates and prepares rune storage, trimmed lengths and path
metadata once. Each query prepares case/normalization once and reuses fixed
100K/2048 upstream slabs per worker, retaining upstream fallback behavior.
An ingestion-time conservative ASCII letter mask includes case/accent-folded
literals and every synthetic pinyin/IME spelling; queries only skip candidates
missing required ASCII letters. This is an order-insensitive rejection filter,
not an alternate matching algorithm. Unicode scalars without ASCII query letters
are left to the original matcher. Dictionary/Unicode coverage and randomized
differential tests guard against false negatives.
Workers keep bounded heaps with **all** tie keys for top-K. Path conservatively
traces exact start during ranking; other schemes can score without alignment.
Only returned hits are traced on `positions=true`. Legacy convenience results
keep exact start/end even when position arrays are disabled.

`find` rank hits have index/score and optional alignment (`nil`/`None`/`null`).
The host retains its original list to map indexes to values. Unlimited is
nil/None/null, zero returns empty, negative limits are invalid at the language
API. C uses -1 for unlimited. Position arrays are canonical rune coordinates;
only returned alignments are converted to UTF-8/UTF-16 by wrappers.

ABI 2 uses a UTF-8 byte arena plus count+1 uint64 offsets at construction,
validated monotonic uint64 handles, versioned fixed-width options, contiguous
typed hits and a canonical positions arena. Struct sizes, flags, reserved
fields, offset bounds and UTF-8 are validated. Limits: 256 MiB bytes/query,
10 million candidates, 64 workers. Structs use natural C alignment. Successful
find returns a C-owned `fzfp_results` even when empty; release exactly once
with `fzfp_results_free`. On error output is NULL. Status codes are documented
in `capi/fzfp.h`; nonexistent/stale handles and double destroy return 3. Destroy
unregisters a handle without invalidating a query already holding a Go reference.
Results are independent of the Index and remain valid after destroy.

`fzfp_typed_abi_version()` is 2. **`fzfp_abi_version()` remains 1**, preserving
existing JSON consumers and the original golden metadata. Language version
queries additionally expose typed ABI 2 (`typedABIVersion` in Swift). No Go
pointer escapes, no per-candidate FFI callback or per-query corpus transfer.
See `capi/examples/typed.c` for a compiled typed ownership/validation example.

### Retained JSON ABI 1

The hand-maintained public header is `capi/fzfp.h`; the generated cgo header is
private. ABI 1 exports only `fzfp_abi_version`, `fzfp_call`, and `fzfp_free`.
`fzfp_call` takes a UTF-8 JSON request and its byte length, returning a C-allocated
JSON buffer with explicit length. The same function does both match and batch
filter. This keeps Go internal structs out of the ABI, gives identical semantics
to all hosts, and avoids per-candidate FFI calls. JSON encoding/decoding costs
are included in the separate legacy benchmark. ABI 1 behavior is unchanged;
typed entrypoints are additive. Swift `filterJSON` is an explicit baseline.

```c
#include "fzfp.h"
#include <stdio.h>
#include <string.h>
int main(void) {
    const char *req = "{\"operation\":\"match\",\"text\":\"中国\",\"query\":\"zg\"}";
    size_t count = 0;
    int32_t status = 0;
    char *out = fzfp_call(req, strlen(req), &count, &status);
    if (!out) return 2;
    fwrite(out, 1, count, stdout);
    fzfp_free(out); /* also required for error responses */
    return status;
}
```

Request keys: `operation` (`match`, `filter`, `version`), `text`, `candidates`,
`query`, `options`. Options keys: `scheme`, `case_sensitive`, `normalize`,
`positions`. Unknown keys, malformed/trailing JSON, invalid options/UTF-8, or
requests over 256 MiB return status 1 and `{"error":"..."}`. Recovered Go panic:
status 2. Null output indicates invalid output pointers or allocation failure;
invalid arbitrary pointers remain caller bugs. Output is not NUL terminated.
The caller must free each non-null output exactly once, using `fzfp_free` only.
Input buffers are borrowed during the call, copied, never retained. No Go
pointer crosses the boundary. Swift throws; Python raises ValueError for status
1, RuntimeError for status 2, MemoryError for null.

`version` returns binding/core contract version `0.1.0`, pinned core source
revision, dictionary version, and independent integer ABI version `1`. The
revision identifies the scoring baseline; binding commits are recorded by Git
and release metadata. A scoring change requires new version metadata and golden
vectors even if the C transport remains ABI compatible. Version 0.x is preview.

## Threading and scheme

The upstream `algo.Init` mutates process globals and assumes pristine startup
state. Re-initializing path then default is unsafe. The new
`src/algo/library_scheme.go` captures all three schemes once, each from pristine
defaults; it selects/restores complete snapshots under a mutex for each library
operation. All FFI calls are safe from host threads, but matching operations
are **serialized across requests**, including whole batches. Within a typed
batch, up to min(4, GOMAXPROCS) workers run by default; explicit workers is
0 (auto) or 1..64. Corpora under 1024 rows use one worker. Each worker owns
scratch and a local bounded heap; it joins before scheme globals are restored.
Worker panics are joined and propagated to the ABI recovery boundary. Same
Index queries serialize scratch reuse; Swift/Python close locks wait for the
in-flight query. Node's synchronous API stays in its owning isolate. Concurrent
mixed-scheme calls are tested. Callers cannot mix bare Go `algo` calls with this facade concurrently
inside the same Go runtime. Swift's static archive and Python's shared library
each own their runtime; linking multiple differently-versioned Go archives into
one executable is unsupported. Normal Go callers of `src/pinyin` keep their
existing initialization contract. No asynchronous cancellation API is promised.

## Build and distribution

From a checkout with Go, C compiler, Python 3.10+, Node 22+ with development
headers, and (on macOS) Xcode:

```sh
bash scripts/build-bindings.sh all
bash scripts/test-bindings.sh
```

Outputs are ignored: `bindings/swift/.cache/CFzfp.xcframework` and
`.cache/bindings/*.whl`, plus an XCFramework zip and its SwiftPM checksum.
Node main/platform npm tarballs are in `.cache/bindings/npm`; a direct C
N-API compile needs no node-gyp or install-time downloads. The Node README
documents optionalDependencies platform resolution and worker_threads use.
Node blocks its event loop during find; Go workers do not make the host call
asynchronous. Browser/WASM, native cancellation, prefix incremental caches,
streaming injection and mutation are intentionally deferred in this phase.
SwiftPM uses the local binaryTarget after this explicit
build; SwiftPM cannot by itself build the Go source. There is no build plugin
that invokes installers or requests permissions. Swift distribution should use
a tagged XCFramework zip and `swift package compute-checksum`, switching the
binaryTarget to a release URL only after review. The in-repository subpackage
currently supports local path dependencies. SwiftPM URL dependencies require
a root Package.swift or a distribution repository; decide that at publication.
No URL or checksum is fabricated in this preview.

macOS 13+ arm64 and x86_64 are combined into one static universal XCFramework.
Python wheels use ctypes and `py3-none-<platform>` tags, not CPython extension
ABI tags. Direct pip wheel builds use the native macOS CPU; `build-bindings.sh`
builds both macOS CPU wheels. The CI matrix builds
both macOS CPUs and Linux x86_64/aarch64, repairs Linux wheels with auditwheel,
and installs/tests the wheel. Plain `linux_*` local wheels are not manylinux
release artifacts. The dependency-free PEP 517 backend only bundles this narrow
shared library; source distributions include Go sources and require Go + C
compiler to rebuild. cibuildwheel is a good standard alternative when expanding
Python interpreter/OS coverage; with a py3/none ABI-independent wheel there is
no need to multiply initial builds by every CPython version. Linux manylinux
containers and native CPU runners avoid cross-C-toolchain assumptions.
On macOS, `pip wheel -C arch=x86_64` (or `arm64`) builds the other CPU wheel
using Xcode's existing C cross compiler. That verifies compilation only;
native-runner wheel installation and tests are still required before release.

Linux serves Python automation/server deployments; native macOS serves Swift
desktop consumers and Python users. Windows, musl, iOS and other Apple platforms
are deferred. The workflow has no release, upload-to-PyPI, signing, or deployment
steps. GitHub Actions artifacts are for review only. A release must additionally
verify deployment target, wheel compatibility, licenses, and a consumer outside
the checkout. The Go runtime is included; distribution size must be reviewed.

## Research sources and choice

Accessed 2026-11-25:

- [Go build modes](https://pkg.go.dev/cmd/go#hdr-Build_modes) and
  [cgo pointer rules](https://pkg.go.dev/cmd/cgo#hdr-Passing_pointers): supported
  c-archive/c-shared modes and the ownership boundary.
- [wireguard-apple Package.swift](https://github.com/wireguard/wireguard-apple/blob/master/Package.swift)
  and [README](https://github.com/wireguard/wireguard-apple): production Go static
  bridge, explicit external build integration; it is not itself an automatic
  SwiftPM Go compiler.
- [wg-go-apple Package.swift](https://github.com/partout-io/wg-go-apple/blob/master/Package.swift):
  XCFramework release zip with binaryTarget URL/checksum.
- [Apple binary Swift packages](https://developer.apple.com/documentation/xcode/distributing-binary-frameworks-as-swift-packages).
- [gopy README](https://github.com/go-python/gopy): generated CPython extensions
  are viable but add generation/CPython integration we do not need for two calls.
- [ctypes](https://docs.python.org/3/library/ctypes.html),
  [platform tags](https://packaging.python.org/en/latest/specifications/platform-compatibility-tags/),
  [cibuildwheel](https://cibuildwheel.pypa.io/): shared-library loading, wheel
  compatibility, and mature multi-platform build/test/repair orchestration.
- [frizbee](https://github.com/Saghen/frizbee): one core with in-tree C, Python,
  and WASM bindings. Its algorithm and benchmark claims are not fzfp results.

Separate binding repositories let language maintainers release independently
but add core-version skew, duplicated tests, and cross-repo coordination. Keep
`capi/`, `bindings/core`, `bindings/swift`, `bindings/python`, and shared
`bindings/testdata` here for atomic review; consider a thin Swift distribution
repository only if needed for URL-based SwiftPM releases. Avoid gopy, a second
implementation, or subprocess execution in the in-process binding hot path.
