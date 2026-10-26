#!/usr/bin/env bash
# Re-render the design screenshots into docs/design/screens.
#   scripts/screens.sh [test-name-regex]
#
# Every TestScreens* runs in a process of its own. The renderer keeps state for the life of the
# process (caches that depend on what was drawn before), so the edge pixels of a few screens
# (a dragged row's tilt) depend on which tests ran earlier in the same process: a full run and a
# run of one test gave different PNGs. One process per test makes a screen the same however many
# tests are asked for, and lets them run side by side.
set -euo pipefail
root="$(git rev-parse --show-toplevel)"
cd "$root"
# Without -trimpath every worktree path gets its own build cache entries and each new worktree recompiles everything.
export GOFLAGS="${GOFLAGS:--trimpath}"

out="$root/docs/design/screens"
pattern="${1:-TestScreens}"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
export SCREENS_OUT="$out" BIN="$work/views.test" LOGS="$work"

go test -c -o "$BIN" ./app/views
cd app/views # the tests read files beside them
tests="$("$BIN" -test.list "$pattern")"
[ -n "$tests" ] || { echo "no test matches $pattern" >&2; exit 1; }

jobs="$(sysctl -n hw.ncpu 2>/dev/null || nproc)"
echo "$tests" | xargs -P "$jobs" -I{} \
  sh -c '"$BIN" -test.run "^{}\$" -test.count=1 >"$LOGS/{}.log" 2>&1 || { echo "FAIL {}"; cat "$LOGS/{}.log"; exit 255; }'

echo "rendered $(echo "$tests" | wc -l | tr -d ' ') screen tests into docs/design/screens"
echo "changed PNGs (git status --short docs/design/screens):"
git -C "$root" status --short docs/design/screens
