# SwiftPM and CLI 0.2.1 release process

This document prepares a release; it does not authorize creating tags or publishing.
Python/npm packages are built and tested at 0.2.1 but are not published here.

## Tags and assets

Use two tags at exactly the same reviewed commit:

- `fzfp-v0.2.1` triggers the existing CLI release workflow.
- `0.2.1` is the SemVer tag SwiftPM resolves. `fzfp-v0.2.1` is not a SwiftPM version.

Only the CLI tag gets a GitHub Release. In addition to the four CLI archives and
their `checksums.txt`, that release carries:

```
CFzfp.xcframework.zip
CFzfp.xcframework.zip.sha256
CFzfp.provenance.json
```

The static XCFramework includes macOS arm64 and x86_64, deployment target macOS
13, the C module/header, fzf/dependency notices, and the Go license. The separate
provenance file records the exact build source commit, source-input SHA256,
Go/Xcode/SDK/Clang/Python/zlib versions, architectures and zip checksum. Source
archives for both tags have the same content; there is no second distribution repo.

## Resolve the manifest/asset dependency

1. Commit core, wrapper, public C headers, versions, tests and build scripts.
   Push the preparation branch. `swift-prepare.yml` builds the XCFramework twice
   in the same CI environment, creates sorted ZIP entries with fixed timestamps
   and modes, and requires the two archives to be byte-identical. It also checks
   SwiftPM's checksum and a clean external consumer. It uploads a private review
   artifact named `swift-0.2.1`; it never publishes a release.
2. Download that successful run's artifact. Record its actual run ID, source
   commit and SHA256 in `.github/swift-release.json`. Set the root binaryTarget
   URL to the future `fzfp-v0.2.1` release and its checksum to the actual zip hash.
   Commit only the manifest, release pin, and documentation. Do not change core,
   wrapper, headers, dependencies, tests or asset-building scripts after this
   build. If any such input changes, build/review again and update the pin.
3. CI verifies the pinned artifact against the final manifest and a hash of all
   declared source inputs. The final tag commit can descend from the build commit
   with only packaging/documentation changes. This breaks the commit-hash cycle
   without pretending the core binary was built from a later commit. The
   `core_revision` value identifies a real reviewed implementation baseline;
   provenance separately identifies the exact artifact build source commit.
4. After independent gate approval, fast-forward main and wait for its CI.
   Create and push **both tags atomically** at that commit. Before publishing,
   the workflow requires both tags to resolve to HEAD. Do not push the SemVer tag
   early: consumers could resolve a manifest before its asset exists.
5. CLI CI explicitly sets `GORELEASER_CURRENT_TAG` to the triggering CLI tag,
   so automatic tag selection cannot publish to the SemVer tag instead. It
   builds/checks/publishes its four archives. Its dependent `swift` job
   downloads the pinned successful preparation run, verifies source/input/hash
   equality, and uploads those exact three reviewed files without overwriting
   existing assets. It does not rebuild a potentially different zip after tagging.
6. Finally CI resolves the remote URL in a fresh external Swift consumer and runs
   its test. This last remote check requires real assets/tags and cannot truthfully
   pass during pre-release review. Local and CI pre-release consumers select the
   local binary with `FZFP_LOCAL_XCFRAMEWORK=1`.

The two tags can be pushed in one atomic ref update. SwiftPM may briefly fail
to download while the release workflow runs; announce availability only after
the upload and clean remote consumer pass. If publication fails, inspect it;
do not move version tags or replace a checksummed asset. Review artifacts expire
after 90 days; if release is delayed beyond retention, prepare and pin a new run.

## Reproduction and limits

From the recorded source commit, with the recorded toolchain/SDK:

```sh
bash scripts/build-bindings.sh swift
python3 scripts/swift-release.py prepare
swift package compute-checksum .cache/swift-release/CFzfp.xcframework.zip
bash scripts/test-swift-consumer.sh local
```

Builds use Go `-trimpath -buildvcs=false -ldflags=-buildid=` and `ZERO_AR_DATE=1`;
input ordering, archive/ZIP timestamps and modes
are normalized. The two complete CI builds verify reproducibility on the recorded
runner. Different compiler/SDK/zlib versions may change native code or compressed
bytes; this is why publication uses the reviewed artifact rather than assuming a
different machine can recreate its checksum. Its source correspondence is checked
independently of byte reproduction. The asset has no tag commit hash embedded in
it; the external provenance describes its exact build commit.

The root manifest normally uses a release URL. For source development, build the
XCFramework and use `FZFP_LOCAL_XCFRAMEWORK=1 swift build/test` (supply a scratch
path under `.cache`). The nested Swift manifest remains a local-build entrypoint.

## Algorithm review

For an ineligible span of s source scalars, dense DP can add no query edge. Its
old gap loses s points, and a previously matched state loses `3+(s-1)`; `prev`
becomes empty. Sparse advancement applies exactly these operations. Coordinates,
anchors and boundary bonuses still refer to the original source. Eligibility is
conservative over all query scalars and all accepted spelling prefixes, so false
positives only cost time; no feasible edge is skipped. The per-call cache is bounded.

For m query scalars, each contributes 16 and a bonus at most B; only the first
bonus is doubled. Thus every DP path is bounded by `16m+B(m+1)`; gaps subtract.
B is at least the Han transition bonus (8) and the scheme's space/delimiter
bonuses. If the native literal result reaches that ceiling, neither the pinyin
path (which also loses one point) nor the pure-literal DP can beat it. The old
wrapper retains native alignment on ties, so returning it early preserves all
result fields. Cases below the bound still run the complete DP and literal floor.

The 0.2.0 release differential checks real CLI output bytes, stderr and status,
not scores/positions that the CLI does not export. Frozen dense/legacy and shared
three-language oracle tests independently compare those fields. No truncation,
line splitting or Maclip-specific search budget is used.
