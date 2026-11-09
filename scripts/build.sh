#!/bin/sh
# Build a static binary. Run this inside Crostini; the OpenFyde host does not
# need a Go toolchain.
#
#   ./scripts/build.sh
#   ARCH=arm64 OUT=dist/alt-shift-ime-arm64 ./scripts/build.sh
set -eu
cd "$(dirname "$0")/.."

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
ARCH="${ARCH:-amd64}"
OUT="${OUT:-dist/alt-shift-ime}"
mkdir -p "$(dirname "$OUT")"

CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" \
  go build -trimpath -buildvcs=false \
  -ldflags "-s -w -X main.version=${VERSION}" \
  -o "$OUT" ./cmd/alt-shift-ime

echo "built $OUT (${VERSION}, linux/${ARCH})"
file "$OUT" 2>/dev/null || true
sha256sum "$OUT"
