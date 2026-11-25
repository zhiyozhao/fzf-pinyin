#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
root="$PWD"
go test -race ./bindings/core ./capi ./src/algo
go build -trimpath -buildvcs=false -o .cache/bindings/fzfp .
test_dir="$(mktemp -d /tmp/fzfp-bindings-test.XXXXXX)"
trap 'rm -r "$test_dir"' EXIT
python3 -m venv "$test_dir/venv"
python3 -m pip wheel --no-deps --wheel-dir "$test_dir/wheels" ./bindings/python
"$test_dir/venv/bin/python" -m pip install --no-deps "$test_dir"/wheels/*.whl
FZFP_CLI="$root/.cache/bindings/fzfp" "$test_dir/venv/bin/python" -m unittest discover -s bindings/python/tests -v
node --test bindings/javascript/test.js
bash scripts/test-javascript-package.sh
if [ "$(uname -s)" = Darwin ]; then
  clang -mmacosx-version-min=13.0 -I capi capi/examples/match.c \
    bindings/swift/.cache/CFzfp.xcframework/macos-arm64_x86_64/libfzfp.a \
    -framework CoreFoundation -framework Security -lresolv -lpthread -o "$test_dir/c-example"
  "$test_dir/c-example"
  clang -mmacosx-version-min=13.0 -I capi capi/examples/typed.c \
    bindings/swift/.cache/CFzfp.xcframework/macos-arm64_x86_64/libfzfp.a \
    -framework CoreFoundation -framework Security -lresolv -lpthread -o "$test_dir/c-typed"
  "$test_dir/c-typed"
  swift build --package-path bindings/swift
  swift test --package-path bindings/swift
fi
