#!/usr/bin/env bash
# Render a mockup to a PNG with headless Chrome at 1x.
#   docs/design/tools/shot.sh <html> <out.png> <width> <height> [#hash]
set -euo pipefail

[ $# -ge 4 ] || { echo "usage: $0 <html> <out.png> <width> <height> [#hash]" >&2; exit 2; }
html="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
out="$2"
case "$out" in /*) ;; *) out="$PWD/$out" ;; esac

"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new --disable-gpu \
  --hide-scrollbars --force-device-scale-factor=1 --window-size="$3,$4" \
  --virtual-time-budget=12000 --screenshot="$out" "file://$html${5:-}"
