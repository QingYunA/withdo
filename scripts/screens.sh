#!/usr/bin/env bash
# Re-render the design screenshots into docs/design/screens.
#   scripts/screens.sh [test-name-regex]
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

SCREENS_OUT="$PWD/docs/design/screens" go test ./app/views -run "${1:-TestScreens}" -count=1
