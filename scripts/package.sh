#!/bin/sh
# Build release tarballs and checksums.
#
#   ./scripts/package.sh v0.2.0
#   VERSION=0.2.0 ARCHES="amd64 arm64" ./scripts/package.sh
#
# The resulting files are written to dist/release/.
set -eu
cd "$(dirname "$0")/.."

VERSION="${1:-${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}}"
VERSION="${VERSION#v}"
ARCHES="${ARCHES:-amd64 arm64}"
OUTDIR="${OUTDIR:-dist/release}"
STAGE_ROOT="$OUTDIR/.stage"

rm -rf "$OUTDIR"
mkdir -p "$OUTDIR" "$STAGE_ROOT"

for arch in $ARCHES; do
  echo "== building linux/$arch"
  ARCH="$arch" OUT="dist/alt-shift-ime-$arch" VERSION="$VERSION" ./scripts/build.sh

  name="alt-shift-ime_${VERSION}_linux_${arch}"
  stage="$STAGE_ROOT/$name"
  mkdir -p "$stage/docs"

  cp "dist/alt-shift-ime-$arch" "$stage/alt-shift-ime"
  cp config/alt-shift-ime.conf "$stage/alt-shift-ime.conf"
  cp init/alt-shift-ime.conf "$stage/alt-shift-ime.upstart.conf"
  cp scripts/setup-alt-shift-ime.sh "$stage/"
  cp scripts/restore-alt-shift-ime.sh "$stage/"
  cp scripts/uninstall-alt-shift-ime.sh "$stage/"
  cp README.md README.ru.md CHANGELOG.md LICENSE "$stage/"
  cp docs/*.md "$stage/docs/"

  chmod 0755 "$stage/alt-shift-ime" "$stage"/*.sh
  chmod 0644 "$stage"/*.conf "$stage"/*.md "$stage"/docs/*.md

  tar -C "$STAGE_ROOT" -czf "$OUTDIR/$name.tar.gz" "$name"
  echo "== packaged $OUTDIR/$name.tar.gz"
done

rm -rf "$STAGE_ROOT"
(
  cd "$OUTDIR"
  sha256sum ./*.tar.gz > checksums.txt
)

echo "== release artifacts"
ls -lh "$OUTDIR"
echo "== checksums"
cat "$OUTDIR/checksums.txt"
