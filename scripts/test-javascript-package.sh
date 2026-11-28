#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
task_dir="$(mktemp -d /tmp/fzfp-npm-test.XXXXXX)"
trap 'rm -r "$task_dir"' EXIT
# Extract the exact npm pack outputs into a consumer module tree. Avoid a
# registry install: unpublished optional dependency names are intentionally
# unresolved until release. The loader exercises the platform package path.
mkdir -p "$task_dir/node_modules/fzfp" "$task_dir/node_modules/@fzfp/native-$(node -p 'process.platform+"-"+process.arch')"
tar -xzf .cache/bindings/npm/fzfp-0.2.1.tgz -C "$task_dir/node_modules/fzfp" --strip-components=1
tar -xzf ".cache/bindings/npm/fzfp-native-$(node -p 'process.platform+"-"+process.arch')-0.2.1.tgz" \
  -C "$task_dir/node_modules/@fzfp/native-$(node -p 'process.platform+"-"+process.arch')" --strip-components=1
FZFP_NPM_CONSUMER="$task_dir" node -e '
const assert=require("node:assert/strict");
const f=require(require("node:path").join(process.env.FZFP_NPM_CONSUMER,"node_modules/fzfp"));
assert.deepEqual(f.match("中国","zg").positions,[0,1]);
const x=new f.Index(["北京","中国"]); assert.equal(x.find("bj")[0].index,0); x.close();
console.log("packed npm consumer passed");'
