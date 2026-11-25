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
```

Positions count Unicode scalars, not Swift Characters. UTF-16 ranges refer to
the original string. API calls are synchronous, throwable, thread safe, and
serialized in Go. Use application scheduling for background work.
Full contract and release URL/checksum plan: [bindings documentation](../../docs/bindings.md).
