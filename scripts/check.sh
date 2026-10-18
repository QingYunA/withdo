#!/usr/bin/env bash
# Checks before a commit or a PR.
#   scripts/check.sh [static|fast|full]
# static: gofmt, vet, i18n. fast (default): static + short tests. full: fast + all tests + race.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

mode="${1:-fast}"
case "$mode" in static | fast | full) ;; *) echo "usage: $0 [static|fast|full]" >&2; exit 2 ;; esac

unformatted="$(gofmt -l app internal cmd)"
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

go test ./...
go test -race ./internal/... ./app/data/...
