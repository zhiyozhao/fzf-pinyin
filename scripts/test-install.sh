#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT HUP INT TERM
mkdir -p "$work/mock" "$work/release" "$work/source" "$work/home"
printf '#!/bin/sh\nprintf "fixture\\n"\n' > "$work/source/fzfp"
for os in linux darwin; do
  for arch in amd64 arm64; do
    tar -czf "$work/release/fzfp-0.2.1-${os}_${arch}.tar.gz" -C "$work/source" fzfp
  done
done
(cd "$work/release" && shasum -a 256 *.tar.gz > checksums.txt)
printf '#!/bin/sh\ncase "$1" in -s) echo "${TEST_OS:-Linux}" ;; -m) echo "${TEST_ARCH:-x86_64}" ;; esac\n' > "$work/mock/uname"
printf '#!/bin/sh\nwhile [ "$1" != -o ]; do url=$1; shift; done\nprintf "%%s\\n" "$url" >> "$FETCH_LOG"\ncp "$FIXTURE/${url##*/}" "$2"\n' > "$work/mock/curl"
chmod +x "$work/mock/uname" "$work/mock/curl"
export FIXTURE="$work/release" HOME="$work/home" PATH="$work/mock:$PATH" FETCH_LOG="$work/fetch.log"
sh "$root/install-fzfp" --version 0.2.1 --bin-dir "$work/installed"
test "$("$work/installed/fzfp")" = fixture
sh "$root/install-fzfp" # default installs this release to ~/.local/bin
test -x "$HOME/.local/bin/fzfp"
sh "$root/install-fzfp" --version latest --bin-dir "$work/latest"
test -x "$work/latest/fzfp"
grep -Fx 'https://github.com/zhiyozhao/fzf-pinyin/releases/download/fzfp-v0.2.1/fzfp-0.2.1-linux_amd64.tar.gz' "$FETCH_LOG" >/dev/null
for os in Linux Darwin; do
  for arch in x86_64 aarch64; do
    TEST_OS=$os TEST_ARCH=$arch sh "$root/install-fzfp" --bin-dir "$work/$os-$arch"
    test -x "$work/$os-$arch/fzfp"
  done
done
printf broken >> "$work/release/fzfp-0.2.1-linux_amd64.tar.gz"
if sh "$root/install-fzfp" --bin-dir "$work/installed"; then exit 1; fi
test "$("$work/installed/fzfp")" = fixture
if TEST_ARCH=armv7l sh "$root/install-fzfp" --bin-dir "$work/unsupported"; then exit 1; fi
test ! -e "$work/unsupported"
if sh "$root/install-fzfp" --version ../bad; then exit 1; fi
cp "$work/release/checksums.txt" "$work/manifest"
head -n 1 "$work/manifest" > "$work/release/checksums.txt"
if sh "$root/install-fzfp" --bin-dir "$work/missing"; then exit 1; fi
test ! -e "$work/missing"
echo 'Installer: four platforms, latest pinning, checksum rejection, safe replacement and invalid inputs passed'
