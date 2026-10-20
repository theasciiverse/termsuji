#!/bin/sh
# Builds GNU Go 3.9.1, the engine termsuji plays against, into stage/: the
# binary (gnugo, or gnugo.exe) and its licence as GNUGO-COPYING. The source
# is Debian's tarball of the upstream tag rel-3-9-1 (GNU Go made no
# tarball of 3.9.1), checked against its sha256; dist/ gets a copy, attached
# to the release so the GPL source travels with the binary (PATCHES.md).
#
#   asciiverse/gnugo.sh                                 # Linux (static) or macOS
#   CROSS=x86_64-w64-mingw32 asciiverse/gnugo.sh        # Windows, from Linux
#
# GNU Go's build compiles small generators and runs them to write its
# pattern databases as C. A cross build cannot run its own generators, so
# for Windows a native build comes first and its generators are put where
# the cross build runs them (./mkpat and so on, without .exe); the
# databases they write are the same on every platform.
set -eu
cd "$(dirname "$0")/.."
URL=https://deb.debian.org/debian/pool/main/g/gnugo/gnugo_3.9.1.orig.tar.gz
SHA256=268e7e46653c00f27ae4f46523a2a76f56d26b3612fe41a68cb852cc10de80fb
# GNU Go is C from before GCC 10 and 14: variables defined in headers, and
# a few old declarations in its Windows code
CF="-O2 -fcommon -Wno-error=implicit-int -Wno-error=implicit-function-declaration -Wno-error=incompatible-pointer-types -Wno-error=int-conversion"
case "$(uname -s)" in
  Linux) LDSTATIC=-static ;;
  *) LDSTATIC= ;;
esac

# stage/ is what goes in the archive: start it empty
rm -rf stage
mkdir -p build stage dist
tarball=dist/gnugo-3.9.1-source.tar.gz
[ -f "$tarball" ] || curl -sSL -o "$tarball" "$URL"
echo "$SHA256  $tarball" | sha256sum -c - 2>/dev/null || echo "$SHA256  $tarball" | shasum -a 256 -c -

# a source tree in build/<name>, fresh
tree() {
  rm -rf "build/$1"
  mkdir -p "build/$1"
  tar -xzf "$tarball" -C "build/$1"
  echo "build/$1/gnugo-3.9.1"
}

native=$(tree native)
(cd "$native" && ./configure --without-readline --disable-color CFLAGS="$CF" LDFLAGS="$LDSTATIC" >/dev/null &&
  make -j4 >/dev/null)

if [ -z "${CROSS:-}" ]; then
  cp "$native/interface/gnugo" stage/gnugo
  strip stage/gnugo
else
  cross=$(tree cross)
  (cd "$cross" && ./configure --host="$CROSS" --without-readline --disable-color \
      CFLAGS="$CF" LDFLAGS="-static" AR="$CROSS-ar" RANLIB="$CROSS-ranlib" >/dev/null &&
    for gen in mkpat joseki mkeyes uncompress_fuseki mkmcpat; do
      cp "../../native/gnugo-3.9.1/patterns/$gen" patterns/
    done &&
    make -j4 AR="$CROSS-ar" >/dev/null)
  cp "$cross/interface/gnugo.exe" stage/gnugo.exe
  "$CROSS-strip" stage/gnugo.exe
fi
cp "$native/COPYING" stage/GNUGO-COPYING
echo "built GNU Go 3.9.1${CROSS:+ for $CROSS}"
