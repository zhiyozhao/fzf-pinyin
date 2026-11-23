#!/bin/sh
set -eu
directory=$1 version=$2
(cd "$directory" && shasum -a 256 -c checksums.txt)
set -- "$directory"/fzfp-*.tar.gz
test "$#" -eq 4
for platform in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64; do
  archive=$directory/fzfp-$version-$platform.tar.gz
  listing=$(tar -tzf "$archive")
  for required in fzfp fzfp.1 LICENSE NOTICE THIRD_PARTY_NOTICES.md; do
    printf '%s\n' "$listing" | grep -Fx "$required" >/dev/null
  done
done
echo 'Four archives: checksums and required root files verified'
