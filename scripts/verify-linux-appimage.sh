#!/usr/bin/env bash
set -euo pipefail

appimage=${1:?usage: verify-linux-appimage.sh APPIMAGE ARCH}
arch=${2:?usage: verify-linux-appimage.sh APPIMAGE ARCH}
case "$arch" in
  amd64) machine=62 ;;
  arm64) machine=183 ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac
test -s "$appimage"

# An AppImage is an ELF runtime with a type marker at offset 8: "AI" and the type
# byte 2, the squashfs flavour. Offset 18 is the ELF machine, little-endian.
test "$(dd if="$appimage" bs=1 count=4 2>/dev/null | od -An -tx1 | tr -d ' \n')" = 7f454c46
test "$(dd if="$appimage" bs=1 skip=8 count=3 2>/dev/null | od -An -tx1 | tr -d ' \n')" = 414902
test "$(dd if="$appimage" bs=1 skip=18 count=1 2>/dev/null | od -An -tu1 | tr -d ' \n')" = "$machine"

stage=$(mktemp -d)
trap 'rm -rf -- "$stage"' EXIT
cp "$appimage" "$stage/app"
chmod +x "$stage/app"
# Extracting reads the image itself, so it needs neither FUSE nor a display.
(cd "$stage" && ./app --appimage-extract >/dev/null)
root=$stage/squashfs-root
host=$(find "$root" -type f -path '*/bin/reasonix-studio-host' ! -path '*app.asar*' | head -1)
page=$(find "$root" -type f -path '*/frontend-next/dist/index.html' ! -path '*app.asar*' | head -1)
test -n "$host" -a -x "$host"
test -n "$page"
test "$(dd if="$host" bs=1 skip=18 count=1 2>/dev/null | od -An -tu1 | tr -d ' \n')" = "$machine"
test -e "$root/AppRun"
echo "AppImage ok: kernel ${host#"$root"/}, page ${page#"$root"/}"
