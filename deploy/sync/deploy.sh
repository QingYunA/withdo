#!/usr/bin/env bash
# Deploys the sync server to the production host: builds the static linux/amd64 adhd-sync from this
# checkout, backs up the live binary and compose file, installs the new ones, restarts, health-checks.
#   deploy/sync/deploy.sh [--dry-run]
# --dry-run builds locally, then prints every remote command instead of running it.
# ADHD_SYNC_HOST is the ssh host (default adhd-srv). sync.env and the Zitadel files are not touched.
# On a failed health check it prints the rollback command and exits 1; it never rolls back itself.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

dry=0
case "${1:-}" in
  "") ;;
  --dry-run) dry=1 ;;
  *) echo "usage: $0 [--dry-run]" >&2; exit 2 ;;
esac

host="${ADHD_SYNC_HOST:-adhd-srv}"
dir=/opt/adhd-todo/sync
stamp="$(date +%Y%m%d-%H%M)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

remote() {
  if [ "$dry" = 1 ]; then echo "[dry-run] ssh $host: $1"; else ssh "$host" "$1"; fi
}
copy() {
  if [ "$dry" = 1 ]; then echo "[dry-run] scp $1 $host:$2"; else scp -q "$1" "$host:$2"; fi
}

echo "building adhd-sync (linux/amd64)"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$work/adhd-sync" ./cmd/adhd-sync

tmp_bin="/tmp/adhd-sync.$stamp"
tmp_yml="/tmp/sync.compose.yml.$stamp"
copy "$work/adhd-sync" "$tmp_bin"
copy deploy/sync/sync.compose.yml "$tmp_yml"

# The binary is bind-mounted, so an unchanged compose file would keep the old one running: recreate.
up="docker compose --env-file sync.env -f sync.compose.yml up -d --force-recreate --wait sync"
rollback="ssh $host 'set -e; cd $dir; cp bin/adhd-sync.bak-$stamp bin/adhd-sync; cp sync.compose.yml.bak-$stamp sync.compose.yml; $up'"
fail() {
  echo "!!! $1. Not rolled back." >&2
  echo "!!! Roll back with:" >&2
  echo "$rollback" >&2
  exit 1
}

remote "set -e; cd $dir; cp bin/adhd-sync bin/adhd-sync.bak-$stamp; cp sync.compose.yml sync.compose.yml.bak-$stamp; install -m 755 $tmp_bin bin/adhd-sync; cp $tmp_yml sync.compose.yml; rm $tmp_bin $tmp_yml; $up" ||
  fail "DEPLOY FAILED while installing or restarting"

if [ "$dry" = 0 ]; then sleep 2; fi
remote "docker logs --since 1m adhd-sync-sync-1 2>&1 | tail -5" || true
if [ "$dry" = 1 ]; then
  echo "[dry-run] health check: expect HTTP 200 from http://127.0.0.1:18080/v1/config"
  code=200
else
  code="$(ssh "$host" "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:18080/v1/config" || true)"
fi

[ "$code" = 200 ] || fail "HEALTH CHECK FAILED: /v1/config returned '${code:-none}' (expected 200)"
[ "$dry" = 1 ] && echo "dry run, nothing was deployed. Rollback would be:" || echo "healthy (200). Roll back with:"
echo "$rollback"
