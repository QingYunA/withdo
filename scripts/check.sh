#!/usr/bin/env bash
# Checks before a commit or a PR.
#   scripts/check.sh [static|fast|full|race|android]
# static: gofmt, vet, i18n. fast (default): static + short tests. full: fast + race on the core.
# race: only the race tests of full, which CI runs as a job of its own beside fast.
# android: the Kotlin unit tests and lint, as CI's android job runs them. It builds the Go core
# (android/core/core.aar) first when missing; delete the .aar after changing mobile/core.
# The sync tests need Postgres: ADHD_SYNC_TEST_DATABASE_URL, else the local adhd-sync-pg container.
# The design screens (TestScreens*, skipped under -short) are rendered by scripts/screens.sh and
# reviewed in the PR.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

mode="${1:-fast}"
case "$mode" in static | fast | full | race | android) ;; *) echo "usage: $0 [static|fast|full|race|android]" >&2; exit 2 ;; esac

if [ "$mode" = android ]; then
  if [ -z "${JAVA_HOME:-}" ] && command -v brew >/dev/null 2>&1; then
    JAVA_HOME="$(brew --prefix openjdk@17)"
    export JAVA_HOME
  fi
  export ANDROID_HOME="${ANDROID_HOME:-$HOME/Library/Android/sdk}"
  [ -f android/core/core.aar ] || scripts/android-core.sh
  (cd android && ./gradlew --no-daemon testDebugUnitTest lintDebug)
  exit 0
fi

# The sync tests skip without a database. CI sets the URL; locally it is the adhd-sync-pg container.
sync_pg_cmd="docker run -d --name adhd-sync-pg -e POSTGRES_PASSWORD=dev -e POSTGRES_DB=adhd_sync_test -p 127.0.0.1:55432:5432 postgres:17-alpine"
if [ "$mode" != static ] && [ -z "${ADHD_SYNC_TEST_DATABASE_URL:-}" ]; then
  if (exec 3<>/dev/tcp/127.0.0.1/55432) 2>/dev/null; then
    export ADHD_SYNC_TEST_DATABASE_URL="postgres://postgres:dev@127.0.0.1:55432/adhd_sync_test?sslmode=disable"
  elif [ "$mode" = fast ]; then
    echo "sync Postgres tests skipped: no database on 127.0.0.1:55432; start one with: $sync_pg_cmd" >&2
  else
    echo "the sync tests need Postgres on 127.0.0.1:55432; start it with: $sync_pg_cmd" >&2
    exit 1
  fi
fi

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
