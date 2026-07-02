#!/usr/bin/env bash
# Builds cross-platform release archives + checksums into dist/.
# Mirrors the goreleaser asset naming that install.sh / install.ps1 /
# `pigment upgrade` expect: pigment_<version>_<os>_<arch>.{tar.gz,zip}
set -euo pipefail

VERSION="${1:-0.1.0}"
LD="-s -w -X github.com/developerAkX/pigment/internal/version.Version=${VERSION}"

rm -rf dist
mkdir -p dist

PLATFORMS=(darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64)

for p in "${PLATFORMS[@]}"; do
  os="${p%/*}"
  arch="${p#*/}"
  echo "building ${os}/${arch}..."
  stage="dist/stage_${os}_${arch}"
  mkdir -p "$stage"
  bin="pigment"
  [ "$os" = "windows" ] && bin="pigment.exe"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -trimpath -ldflags "$LD" -o "$stage/$bin" ./cmd/pigment
  cp README.md LICENSE "$stage/" 2>/dev/null || true
  if [ "$os" = "windows" ]; then
    (cd "$stage" && zip -q -r "../pigment_${VERSION}_${os}_${arch}.zip" .)
  else
    tar -czf "dist/pigment_${VERSION}_${os}_${arch}.tar.gz" -C "$stage" .
  fi
  rm -rf "$stage"
done

cd dist
if command -v sha256sum >/dev/null 2>&1; then
  SHA="sha256sum"
else
  SHA="shasum -a 256"
fi
# Emit "<hash>  <filename>" with bare filenames (no ./ prefix).
: > checksums.txt
for f in pigment_"${VERSION}"_*; do
  $SHA "$f" | awk '{print $1"  "$2}' >> checksums.txt
done

echo "=== dist ==="
ls -la
echo "=== checksums.txt ==="
cat checksums.txt
