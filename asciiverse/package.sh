#!/bin/sh
# Packs dist/termsuji-<target>.tar.gz (or .zip for Windows) for av: the
# termsuji and gnugo binaries at the top, with termsuji's LICENSE,
# README.md and PATCHES.md, and GNU Go's licence (asciiverse/gnugo.sh puts
# gnugo and GNUGO-COPYING in stage/).
#
#   asciiverse/package.sh x86_64-unknown-linux-musl path/to/termsuji
#   asciiverse/package.sh x86_64-pc-windows-gnu path/to/termsuji.exe
set -eu
cd "$(dirname "$0")/.."
target="$1"
binary="$2"
case "$target" in *windows*) exe=.exe ;; *) exe= ;; esac
cp "$binary" "stage/termsuji$exe"
cp LICENSE README.md PATCHES.md stage/
mkdir -p dist
if [ -n "$exe" ]; then
  (cd stage && zip -qr "../dist/termsuji-$target.zip" .)
else
  tar -czf "dist/termsuji-$target.tar.gz" -C stage .
fi
ls dist
