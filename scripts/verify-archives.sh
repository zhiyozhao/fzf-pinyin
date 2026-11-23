#!/bin/sh
set -eu
directory=$1 version=$2
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
(cd "$directory" && shasum -a 256 -c checksums.txt)
set -- "$directory"/fzfp-*.tar.gz
test "$#" -eq 4
for platform in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64; do
  archive=$directory/fzfp-$version-$platform.tar.gz
  listing=$(tar -tzf "$archive")
  for required in fzfp fzfp.1 LICENSE NOTICE THIRD_PARTY_NOTICES.md; do
    printf '%s\n' "$listing" | grep -Fx "$required" >/dev/null
  done
  # Compare the complete appendix, not merely the presence of its filename.
  tar -xOzf "$archive" THIRD_PARTY_NOTICES.md | cmp - "$root/THIRD_PARTY_NOTICES.md"
done
echo 'Four archives: checksums, root files and complete dependency licenses verified'
