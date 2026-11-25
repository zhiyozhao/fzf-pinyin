# Fzfp for Swift

Swift 5.9+, macOS 13+, Xcode, Go and a C compiler for source builds.

```sh
bash scripts/build-bindings.sh swift
cd bindings/swift
swift build
swift test
```

Add `.package(path: "path/to/fzfp/bindings/swift")` and the `Fzfp` product to
a consuming Swift package after building the XCFramework. Both macOS CPUs
are included. The local binaryTarget is generated and ignored by Git.
For this local path the package identity is `swift` (the directory name), so
the consuming target uses `.product(name: "Fzfp", package: "swift")`.

```swift
import Fzfp

let match = try Fzfp.match("中国", query: "zhongguo")
assert(match.matched && match.positions == [0, 1])
let hits = try Fzfp.filter(["北京", "-b-j", "-bj"], query: "bj",
                           options: Options(scheme: .history))
for hit in hits {
    print(hit.index, hit.text, hit.score)
    print(hit.match.utf16Ranges(in: hit.text))
}
print(try Fzfp.version())

// Keep the same corpus across many queries; return only visible hits.
let titles = ["北京", "中国", "fooBar"]
let index = try Index(titles, options: Options(scheme: .path), workers: 4)
for hit in try index.find("bj", limit: 50, positions: true) {
    print(titles[hit.index], hit.score, hit.alignment!.utf16Ranges(in: titles[hit.index]))
}
try index.close() // deinit is also a fallback
```

Positions count Unicode scalars, not Swift Characters. UTF-16 ranges refer to
the original string. API calls are synchronous, throwable, thread safe, and
serialized across requests in Go; each batch can use up to four workers by
default. Index is not Sendable; own it in a background actor for UI work.
Same-index calls and close are locked; close is idempotent. Cancelling an
await does not stop a native query. Rank-only `find` returns no alignment by
default; positions opt in. `limit: nil` returns all, zero none, negative throws.
Corpus construction is a one-time cost; do not rebuild an Index per keystroke.
Full contract and release URL/checksum plan: [bindings documentation](../../docs/bindings.md).
