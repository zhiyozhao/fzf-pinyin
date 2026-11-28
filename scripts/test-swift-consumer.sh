#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
root="$PWD"
task_dir="$(mktemp -d /tmp/fzfp-swift-consumer.XXXXXX)"
trap 'rm -r "$task_dir"' EXIT
cp -R bindings/swift/Consumer "$task_dir/Consumer"
case "${1:-local}" in
  local)
    FZFP_PACKAGE_PATH="$root" FZFP_LOCAL_XCFRAMEWORK=1 \
      swift test --package-path "$task_dir/Consumer" --scratch-path "$task_dir/build"
    ;;
  remote)
    env -u FZFP_PACKAGE_PATH -u FZFP_LOCAL_XCFRAMEWORK \
      swift test --package-path "$task_dir/Consumer" --scratch-path "$task_dir/build"
    ;;
  *) echo 'usage: test-swift-consumer.sh [local|remote]' >&2; exit 2 ;;
esac
