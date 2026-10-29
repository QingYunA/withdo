# Sync server deployment (M11 S4b)

Production runs on one Debian 12 host that already serves other sites through nginx on 80/443.
Everything here is added beside them; nothing else on the host is changed.

| Name | What | Upstream |
|---|---|---|
| `todo.sri-ai.xyz` | `adhd-sync` (this repo's `cmd/adhd-sync`), with PocketBase embedded as the account system (M13) | `127.0.0.1:18080` |

The DNS record is an A record, **DNS only** (not proxied by Cloudflare): certbot answers the HTTP
challenge itself, and the sync server's SSE stream would be cut by the proxy's timeout.

## Layout on the host

```text
/opt/adhd-todo/                 mode 700, root
└── sync/
    ├── sync.compose.yml        adhd-sync + its own Postgres
    ├── pb_data/                PocketBase's data (users, password hashes, its settings), owned by uid 65532
    ├── sync.env                SYNC_DB_PASSWORD and the voice variables below, mode 600
    ├── bin/adhd-sync           static linux/amd64 build
    ├── backup.sh               the daily backup (Backups, below)
    └── backups/                one folder per backup, mode 700, kept 14 days
/etc/systemd/system/withdo-backup.{service,timer}   run backup.sh every day at 03:17 UTC
/etc/nginx/sites-available/adhd-todo   nginx-adhd-todo.conf, linked from sites-enabled
/etc/letsencrypt/live/adhd-todo/       the certificate for todo.sri-ai.xyz; certbot.timer renews it
```

## From scratch

1. **DNS**: an A record for `todo.sri-ai.xyz` to the host, DNS only.
2. **Certificate**: install the port-80 server block of `nginx-adhd-todo.conf` alone, `nginx -t`,
   reload, then
   `certbot certonly --webroot -w /var/www/certbot --cert-name adhd-todo -d todo.sri-ai.xyz --deploy-hook "systemctl reload nginx"`.
3. **nginx**: install the whole `nginx-adhd-todo.conf`, `nginx -t`, reload.
4. **adhd-sync**: build `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/adhd-sync` into
   `sync/bin/`, write `sync.env`, create `sync/pb_data` (see below) and
   `docker compose --env-file sync.env -f sync.compose.yml up -d --wait`.

Clients use `server https://todo.sri-ai.xyz`; `GET /v1/config` answers `{"auth":"pocketbase"}` and
they sign in through PocketBase's routes.

## PocketBase accounts (M13)

`adhd-sync -pb-dir=/pb_data` embeds PocketBase (pinned in `go.mod`) in the same process and listener.
It owns the users and their passwords; sync still keys data by `users.subject`, which is
`pb:<record id>` for a PocketBase user. The server accepts PocketBase tokens and, for development,
the static tokens of `-tokens`. The log has one line the first time each subject is accepted by
`pocketbase`.

- **Volume.** `sync.compose.yml` mounts `./pb_data` at `/pb_data`. Create it once:
  `mkdir -p /opt/adhd-todo/sync/pb_data && chown 65532:65532 /opt/adhd-todo/sync/pb_data` (the
  container runs as distroless `nonroot`). `deploy.sh` copies it to `pb_data.bak-STAMP` before each
  deploy; back it up with the Postgres dump too, as it holds every password hash.
- **nginx** proxies all of PocketBase: the users' sign-in, and the admin UI at
  `https://todo.sri-ai.xyz/_/` with the API it calls (public since 2026-10-22). It sets `X-Real-IP`
  from the connection on every location, so a client cannot choose the address the rate limits count.
- **Password guessing** is limited to 10 user sign-ins and 5 superuser sign-ins a minute per client address (PocketBase's rate limit,
  counting by the `X-Real-IP` nginx sets).
- **Mail** goes out through Resend's SMTP (`smtp.resend.com:465`, TLS, user `resend`, the password a
  send-only API key for `sri-ai.xyz`), from `Withdo <noreply@sri-ai.xyz>`. It is set in
  PocketBase's settings (admin UI > Settings > Mail), not in this repo; the rename migration
  (`pbauth/migrate_withdo.go`) turns a sender name of `ADHD Todo` into `Withdo`. The key is also in the owner's
  keychain as `adhd-todo-resend-smtp`. The domain's DKIM, the `send` and `rsend` CNAMEs and DMARC are
  Cloudflare records Resend gave (2026-10-22); no MX, so Resend does not receive mail for the domain.
  The app URL in the same settings is `https://todo.sri-ai.xyz`, for the links in the emails.
- **Superusers need two steps** (MFA, since 2026-10-22): the password, then an 8-digit code mailed to
  them. If mail stops working, reset a superuser from the server with `pb-superuser` below; MFA is
  turned off in the `_superusers` collection's options.
- **Superuser** (for the admin UI), once, from the server with the password on stdin:
  `docker compose --env-file sync.env -f sync.compose.yml exec -T sync /adhd-sync pb-superuser -pb-dir=/pb_data -email=EMAIL < password-file`.
  The account commands below run the same way (`exec -T sync /adhd-sync …`); they work while
  the server runs, and `move-subject` needs `-database-url` (the compose file's value).
- **Move an account** from an older identity to PocketBase, without losing its data:
  1. `adhd-sync pb-import-user -pb-dir=/pb_data -email=EMAIL -name=NAME` reads the password from stdin
     (no echo on a terminal) and prints the new subject, `pb:<id>`.
  2. `adhd-sync move-subject -database-url=… -from=<old subject> -to=pb:<id>` rewrites
     `users.subject` in one transaction. It refuses when the target already has devices or rows; a
     target workspace that only the first request made, empty, is replaced.
     By default the old subject is kept as an alias (`-keep-alias=false` to skip it), so a device
     still signed in with it reaches the same workspace instead of getting a new empty one.
     `move-subject` refuses when the target is an alias.
  3. Sign in on each device; the data is there.
  4. `adhd-sync list-subjects -database-url=…` prints `user_id subject kind devices rows` (kind is
     `primary` or `alias`). `adhd-sync add-alias -database-url=… -subject=<old> -target=<current>`
     backfills an alias for an account moved without one (an empty user the old device already
     created is replaced). `adhd-sync drop-alias -database-url=… -subject=<old>` removes one once
     no device uses it.

## Voice input

`POST /v1/transcribe` turns a recording from the app into text. The app uploads a WAV (16 kHz mono
PCM16, 60 s at most), the server sends it to the MiMo speech recognizer (`mimo-v2.5-asr`) with its
own key and returns `{"text": "..."}`. The app never holds the key. It is configured by environment
variables, which `sync.compose.yml` passes through from `sync.env`:

| Variable | Meaning |
|---|---|
| `MIMO_API_KEY` | The recognizer's key. Empty or unset turns voice off: the route answers `503 voice_disabled`. |
| `MIMO_BASE_URL` | The recognizer's API root; default `https://api.xiaomimimo.com/v1`. |
| `VOICE_DAILY_LIMIT` | Recordings per account per UTC day; default 200. Past it: `429 voice_quota` with `Retry-After`. The count is in memory, so a restart resets it. |

Answers: 401 `unauthenticated`, 413 `audio_too_large` / `audio_too_long`, 415 `unsupported_media`,
400 `bad_audio`, 502 `transcribe_failed` (the recognizer failed; that request is not counted). Errors
never include the key or the recognizer's body. Put the key only in `sync.env` on the host (mode 600),
never in the repository.

## Checks

- `curl https://todo.sri-ai.xyz/v1/config` is `{"auth":"pocketbase"}`.
- `curl https://todo.sri-ai.xyz/healthz` is `ok`; `POST /v1/devices` without a token is 401.
- `POST /v1/transcribe` without a token is 401; with a token and no key configured it is 503.

## Updating the server

`deploy/sync/deploy.sh` builds `adhd-sync` from the current checkout, copies it and
`sync.compose.yml` to the host, recreates the `sync` container (the binary is bind-mounted) and checks `GET /v1/config` on
`127.0.0.1:18080` for 200. The owner confirms each production deploy; do not run it unasked.

```sh
deploy/sync/deploy.sh --dry-run   # builds, prints every remote command, runs none
deploy/sync/deploy.sh             # ADHD_SYNC_HOST overrides the ssh host (default adhd-srv)
```

It first copies `bin/adhd-sync` and `sync.compose.yml` to `*.bak-YYYYMMDD-HHMM`, and never touches
`sync.env`. At the end it prints the one-line rollback command (restore both
backups, `up -d --wait sync`). If the health check fails it prints that command prominently and
exits 1; it does not roll back by itself.

## Backups

`backup.sh` writes `backups/<UTC date-time>/` every day: `postgres.sql.gz` (the tasks),
`pocketbase-data.db.gz` (the accounts) and `pocketbase-storage.tar.gz`. It deletes the folders
older than 14 days (`BACKUP_KEEP_DAYS`). The backups stay on the same disk: they cover a bad deploy
or a broken database, not a lost host.

Install, or update after a change to these files:

```sh
scp deploy/sync/backup.sh adhd-srv:/opt/adhd-todo/sync/
scp deploy/sync/withdo-backup.service deploy/sync/withdo-backup.timer adhd-srv:/etc/systemd/system/
ssh adhd-srv 'chmod 700 /opt/adhd-todo/sync/backup.sh && systemctl daemon-reload && systemctl enable --now withdo-backup.timer'
```

`systemctl list-timers withdo-backup.timer` shows the next run; `journalctl -u withdo-backup` the
last ones. A run by hand: `/opt/adhd-todo/sync/backup.sh`.

Restoring, in `/opt/adhd-todo/sync` with `b=backups/<date-time>`:

1. `docker compose --env-file sync.env -f sync.compose.yml stop sync`
2. Tasks: drop and recreate the database, then load the dump:
   `docker compose --env-file sync.env -f sync.compose.yml exec -T postgres psql -U adhd_sync -d postgres -c 'DROP DATABASE adhd_sync' -c 'CREATE DATABASE adhd_sync'`, then
   `gzip -dc $b/postgres.sql.gz | docker compose --env-file sync.env -f sync.compose.yml exec -T postgres psql -v ON_ERROR_STOP=1 -U adhd_sync -d adhd_sync`.
3. Accounts: `rm -f pb_data/data.db-wal pb_data/data.db-shm`,
   `gzip -dc $b/pocketbase-data.db.gz > pb_data/data.db`, and for the files
   `rm -rf pb_data/storage && tar -xzf $b/pocketbase-storage.tar.gz -C pb_data`; then
   `chown -R 65532:65532 pb_data`.
4. `docker compose --env-file sync.env -f sync.compose.yml up -d --wait sync`

Tested on 2026-10-29: the dump loaded into a fresh `postgres:17.10-alpine` with the same row count
in every table, and the copied `data.db` passed `integrity_check` with the same users.

## Upgrades

- adhd-sync: use `deploy/sync/deploy.sh` (above). Its Postgres migrations run at start.
