#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
root="$PWD"
mkdir -p .cache/bindings
case "${1:-all}" in all|swift|python|javascript) ;; *) echo 'usage: build-bindings.sh [all|swift|python|javascript]' >&2; exit 2 ;; esac
if [ "${1:-all}" = javascript ]; then exec bash scripts/build-javascript.sh; fi
case "${1:-all}" in
  swift|all)
    if [ "$(uname -s)" != Darwin ] && [ "${1:-all}" = swift ]; then echo 'Swift artifacts require macOS/Xcode' >&2; exit 1; fi
    if [ "$(uname -s)" = Darwin ]; then
    build_dir="$(mktemp -d /tmp/fzfp-swift.XXXXXX)"
    trap 'rm -r "$build_dir"' EXIT
    for arch in arm64 amd64; do
      CGO_ENABLED=1 GOOS=darwin GOARCH="$arch" MACOSX_DEPLOYMENT_TARGET=13.0 \
        go build -trimpath -buildvcs=false -buildmode=c-archive -o "$build_dir/$arch.a" ./capi
    done
    lipo -create "$build_dir/arm64.a" "$build_dir/amd64.a" -output "$build_dir/libfzfp.a"
    mkdir -p "$build_dir/include" bindings/swift/.cache
    cp capi/fzfp.h "$build_dir/include/"
    # module.modulemap is versioned; generated Go headers are private.
    cp capi/module.modulemap "$build_dir/include/"
    xcodebuild -create-xcframework -library "$build_dir/libfzfp.a" -headers "$build_dir/include" -output "$build_dir/CFzfp.xcframework"
    if [ -d bindings/swift/.cache/CFzfp.xcframework ]; then
      mv bindings/swift/.cache/CFzfp.xcframework "$build_dir/previous.xcframework"
    fi
    mv "$build_dir/CFzfp.xcframework" bindings/swift/.cache/
    cp LICENSE NOTICE THIRD_PARTY_NOTICES.md bindings/swift/.cache/CFzfp.xcframework/
    go_root="$(go env GOROOT)"
    if [ -f "$go_root/LICENSE" ]; then
      cp "$go_root/LICENSE" bindings/swift/.cache/CFzfp.xcframework/Go-LICENSE
    else
      cp "$go_root/../LICENSE" bindings/swift/.cache/CFzfp.xcframework/Go-LICENSE
    fi
    ditto -c -k --sequesterRsrc --keepParent bindings/swift/.cache/CFzfp.xcframework "$build_dir/CFzfp.xcframework.zip"
    mv "$build_dir/CFzfp.xcframework.zip" .cache/bindings/
    swift package compute-checksum .cache/bindings/CFzfp.xcframework.zip > .cache/bindings/CFzfp.xcframework.zip.sha256
    if [ "${1:-all}" = swift ]; then exit 0; fi
    fi
    ;;
esac
if [ "$(uname -s)" = Darwin ]; then
  for wheel_arch in arm64 x86_64; do
    python3 -m pip wheel --no-deps -C "arch=$wheel_arch" --wheel-dir "$root/.cache/bindings" "$root/bindings/python"
  done
else
  python3 -m pip wheel --no-deps --wheel-dir "$root/.cache/bindings" "$root/bindings/python"
fi
if [ "${1:-all}" = all ]; then bash scripts/build-javascript.sh; fi
