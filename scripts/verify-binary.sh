#!/bin/sh
# Run for every GoReleaser build before it can be archived or published.
set -eu
export FZF_DEFAULT_OPTS= FZF_DEFAULT_OPTS_FILE=/dev/null
binary=$1 os=$2 arch=$3
inspection=$(mktemp)
trap 'rm -f "$inspection"' EXIT HUP INT TERM
strings "$binary" > "$inspection"
if LC_ALL=C grep -F '/Users/' "$inspection" || LC_ALL=C grep -Ei 'mioffice|xiaomi' "$inspection"; then
  echo "Private build strings found in $binary" >&2
  exit 1
fi
echo "Strings privacy check passed: $os/$arch"
case "$(uname -s):$(uname -m):$os:$arch" in
  Linux:x86_64:linux:amd64|Linux:aarch64:linux:arm64|Darwin:arm64:darwin:arm64|Darwin:x86_64:darwin:amd64)
    test "$(printf '北京\n上海\n' | "$binary" --filter=bj)" = 北京
    "$binary" --version
    echo "Native pinyin filter passed: $os/$arch"
    ;;
esac
