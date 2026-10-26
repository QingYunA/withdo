#!/usr/bin/env bash
# Checks before a commit or a PR.
#   scripts/check.sh [static|fast|full|race|linux|android]
#   scripts/check.sh linux [go test args]   (default ./...)
# static: gofmt, vet, i18n, the 800-line file limit, the generated catalog.
# fast (default): static + short tests. full: fast + race on the core.
# race: only the race tests of full, which CI runs as a job of its own beside fast.
# linux: go test -short in a Docker image like CI's (scripts/ci-linux.Dockerfile), to reproduce a
# failure that passes on the Mac (GOOS-dependent text, timing). The module and build caches live in
# named volumes. The sync Postgres tests skip there: the container has no database.
#   scripts/check.sh linux ./app/views -run TestX
# android: the Kotlin unit tests and lint, as CI's android job runs them. It builds the Go core
# (android/core/core.aar) first when it is missing or older than the Go sources.
# The sync tests need Postgres: ADHD_SYNC_TEST_DATABASE_URL, else the local adhd-sync-pg container.
# The design screens (TestScreens*, skipped under -short) are rendered by scripts/screens.sh and
# reviewed in the PR.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
# Without -trimpath every worktree path gets its own build cache entries and each new worktree recompiles everything.
export GOFLAGS="${GOFLAGS:--trimpath}"

mode="${1:-fast}"
case "$mode" in static | fast | full | race | linux | android) ;; *) echo "usage: $0 [static|fast|full|race|linux|android]" >&2; exit 2 ;; esac

if [ "$mode" = android ]; then
  if [ -z "${JAVA_HOME:-}" ] && command -v brew >/dev/null 2>&1; then
    JAVA_HOME="$(brew --prefix openjdk@17)"
    export JAVA_HOME
  fi
  export ANDROID_HOME="${ANDROID_HOME:-$HOME/Library/Android/sdk}"
  # Rebuild the Go core when any Go file or go.mod is newer than it, so the tests never run an old one.
  aar=android/core/core.aar
  if [ ! -f "$aar" ] || [ -n "$(find mobile internal app go.mod -newer "$aar" \( -name '*.go' ! -name '*_test.go' -o -name go.mod \) | head -1)" ]; then
    scripts/android-core.sh
  fi
  (cd android && ./gradlew --no-daemon testDebugUnitTest lintDebug)
  exit 0
fi

if [ "$mode" = linux ]; then
  shift
  [ $# -gt 0 ] || set -- ./...
  docker build -q -t adhd-ci-linux --build-arg "GO_VERSION=$(awk '/^go /{print $2}' go.mod)" \
    -f scripts/ci-linux.Dockerfile scripts >/dev/null
  exec docker run --rm -v "$PWD":/src -w /src -v adhd-ci-gomod:/go/pkg/mod -v adhd-ci-gobuild:/root/.cache/go-build \
    -e GOFLAGS adhd-ci-linux go test -short "$@"
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

# Hand-written Go files stay under 800 lines. Generated files and the pixel-art data are exempt.
long_allow="app/theme/harness_marks.go"
long="$(find app internal cmd mobile -name '*.go' ! -name '*_gen.go' 2>/dev/null | grep -vxF "$long_allow" |
  xargs wc -l | awk '$2 != "total" && $1 > 800 {print $1, $2}')"
if [ -n "$long" ]; then
  echo "Go files over 800 lines (split them, or allow-list pixel data in scripts/check.sh):" >&2
  echo "$long" >&2
  exit 1
fi

# The generated catalog must match the JSON catalogs: generating it again changes nothing.
catalog=app/locales/catalog_gen.go
before="$(shasum "$catalog")"
(cd app && go tool mygo generate >/dev/null)
if [ "$(shasum "$catalog")" != "$before" ]; then
  echo "$catalog was stale; it is regenerated now: commit it" >&2
  exit 1
fi
[ "$mode" = static ] && exit 0

go test -short ./...
[ "$mode" = fast ] && exit 0

race_tests
