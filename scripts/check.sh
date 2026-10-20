#!/usr/bin/env bash
# Checks before a commit or a PR.
#   scripts/check.sh [static|fast|full|race]
# static: gofmt, vet, i18n. fast (default): static + short tests. full: fast + race on the core.
# race: only the race tests of full, which CI runs as a job of its own beside fast.
# The design screens (TestScreens*, skipped under -short) are rendered by scripts/screens.sh and
# reviewed in the PR.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

mode="${1:-fast}"
case "$mode" in static | fast | full | race) ;; *) echo "usage: $0 [static|fast|full|race]" >&2; exit 2 ;; esac

race_tests() { go test -race ./internal/... ./app/data/...; }
if [ "$mode" = race ]; then
  race_tests
  exit 0
fi

# The toolchain's own gofmt, as in CI: an older gofmt on PATH formats some code differently.
unformatted="$("$(go env GOROOT)/bin/gofmt" -l app internal cmd)"
if [ -n "$unformatted" ]; then
  echo "gofmt needed (run gofmt -w):" >&2
  echo "$unformatted" >&2
  exit 1
fi
go vet ./...
(cd app && go tool mygo i18n check)
[ "$mode" = static ] && exit 0

go test -short ./...
[ "$mode" = fast ] && exit 0

race_tests
