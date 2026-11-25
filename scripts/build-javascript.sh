#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
# Direct N-API C compilation: no node-gyp, npm install, or downloaded headers.
node_prefix="$(node -p 'require("path").dirname(require("path").dirname(process.execPath))')"
node_include="${FZFP_NODE_INCLUDE:-$node_prefix/include/node}"
if [ ! -f "$node_include/node_api.h" ]; then echo 'Node development headers missing; set FZFP_NODE_INCLUDE to existing headers' >&2; exit 1; fi
mkdir -p bindings/javascript/.cache .cache/bindings/npm
native_dir="$PWD/bindings/javascript/.cache"
CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=13.0 go build -trimpath -buildvcs=false -buildmode=c-archive -o "$native_dir/libfzfp.a" ./capi
if [ "$(uname -s)" = Darwin ]; then
  clang -O3 -std=c11 -mmacosx-version-min=13.0 -bundle -undefined dynamic_lookup -DNAPI_VERSION=8 -DNODE_GYP_MODULE_NAME=fzfp \
    -I "$node_include" -I capi bindings/javascript/native.c "$native_dir/libfzfp.a" \
    -framework CoreFoundation -framework Security -lresolv -lpthread -o "$native_dir/fzfp.node"
else
  cc -O3 -std=c11 -fPIC -shared -DNAPI_VERSION=8 -DNODE_GYP_MODULE_NAME=fzfp \
    -I "$node_include" -I capi bindings/javascript/native.c "$native_dir/libfzfp.a" \
    -lpthread -ldl -lm -o "$native_dir/fzfp.node"
fi
node scripts/pack-javascript.js
