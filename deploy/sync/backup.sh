#!/usr/bin/env bash
# Backs up the sync server's data into backups/<UTC date-time>/ next to this file, and deletes the
# backups older than BACKUP_KEEP_DAYS (14):
#   postgres.sql.gz             pg_dump of the tasks database
#   pocketbase-data.db.gz       PocketBase's data.db (accounts), copied with SQLite's backup API
#   pocketbase-storage.tar.gz   PocketBase's uploaded files
# Runs on the host as root, daily from withdo-backup.timer; by hand: /opt/adhd-todo/sync/backup.sh
# Restoring: deploy/sync/README.md (Backups).
set -euo pipefail
cd "$(dirname "$0")"

keep_days="${BACKUP_KEEP_DAYS:-14}"
stamp="$(date -u +%Y%m%d-%H%M)"
mkdir -p backups
chmod 700 backups
part="backups/.partial-$stamp"
mkdir "$part"
trap 'rm -rf "$part"' EXIT

docker compose --env-file sync.env -f sync.compose.yml exec -T postgres \
  pg_dump -U adhd_sync -d adhd_sync --no-owner | gzip > "$part/postgres.sql.gz"
# The dump of a working database is never a few bytes; an empty one means pg_dump said nothing.
[ "$(gzip -dc "$part/postgres.sql.gz" | wc -c)" -gt 1000 ] || { echo "the Postgres dump is empty" >&2; exit 1; }

# SQLite's backup API reads a consistent copy while PocketBase keeps writing (WAL mode).
python3 - pb_data/data.db "$part/pocketbase-data.db" <<'PY'
import sqlite3, sys
src = sqlite3.connect(f"file:{sys.argv[1]}?mode=ro", uri=True)
dst = sqlite3.connect(sys.argv[2])
src.backup(dst)
dst.close()
src.close()
PY
gzip "$part/pocketbase-data.db"
tar -czf "$part/pocketbase-storage.tar.gz" -C pb_data storage

mv "$part" "backups/$stamp"
trap - EXIT
find backups -mindepth 1 -maxdepth 1 -type d -name '20*' -mtime +"$keep_days" -exec rm -rf {} +
echo "backups/$stamp $(du -sh "backups/$stamp" | cut -f1)"
