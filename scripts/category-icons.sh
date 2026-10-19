#!/usr/bin/env bash
# Regenerate app/theme/category_icons_gen.go from the pixelarticons release and
# docs/design/category-icons.txt (the curated list, with Chinese keywords).
#   scripts/category-icons.sh [max-error]
# max-error (default 16) is the most wrong cells of 576 an icon may have after it is reduced to
# 12 x 12 and still be searchable; the generator lists what it drops.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

version=2.4.2
sha256=1c12c5d08b837bf5684d1f0398a19eb8d8793549dcd4884c1267d6f2464cb500
url="https://registry.npmjs.org/pixelarticons/-/pixelarticons-$version.tgz"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
curl -fsSL -o "$work/p.tgz" "$url"
echo "$sha256  $work/p.tgz" | shasum -a 256 -c - >/dev/null || { echo "pixelarticons $version: checksum mismatch" >&2; exit 1; }
mkdir "$work/pkg"
tar -xzf "$work/p.tgz" -C "$work/pkg"

go run ./internal/pixmap/cmd/pxicons \
  -svg "$work/pkg/package/svg" \
  -curated docs/design/category-icons.txt \
  -max-error "${1:-16}" \
  -source "pixelarticons $version" \
  -o "$work/gen.go"
gofmt "$work/gen.go" > app/theme/category_icons_gen.go
