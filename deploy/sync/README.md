# Sync server deployment (M11 S4b)

Production runs on one Debian 12 host that already serves other sites through nginx on 80/443.
Everything here is added beside them; nothing else on the host is changed.

| Name | What | Upstream |
|---|---|---|
| `api.withdo.si` | `adhd-sync` (this repo's `cmd/adhd-sync`), with PocketBase embedded as the account system (M13), through a Cloudflare Tunnel (below) | `127.0.0.1:18080` |
| `todo.sri-ai.xyz` | the same server under its legacy name, served directly by nginx until old clients have updated | `127.0.0.1:18080` |

`api.withdo.si` is the official address since 2026-11-03. The legacy name's DNS record is an A
record, **DNS only** (not proxied by Cloudflare): certbot answers the HTTP challenge itself. Its
sections below ("From scratch", the certificate) describe that legacy path.

## Layout on the host

```text
/opt/adhd-todo/                 mode 700, root
└── sync/
    ├── sync.compose.yml        adhd-sync + its own Postgres
    ├── pb_data/                PocketBase's data (users, password hashes, its settings), owned by uid 65532
    ├── sync.env                SYNC_DB_PASSWORD, the voice, model and Waffo variables below, mode 600
    ├── secrets/                Waffo's API private key (waffo-test.pem), mounted read-only at /secrets
    ├── bin/adhd-sync           static linux/amd64 build
    ├── backup.sh               the daily backup (Backups, below)
    └── backups/                one folder per backup, mode 700, kept 14 days
/etc/systemd/system/withdo-backup.{service,timer}   run backup.sh every day at 03:17 UTC
/etc/nginx/sites-available/adhd-todo   nginx-adhd-todo.conf (legacy name), linked from sites-enabled
/etc/nginx/sites-available/api-withdo  nginx-api-withdo.conf (127.0.0.1:8088, for cloudflared), linked from sites-enabled
(cloudflared itself is installed by `cloudflared service install`; its config lives at Cloudflare)
/etc/letsencrypt/live/adhd-todo/       the certificate for todo.sri-ai.xyz; certbot.timer renews it
```

## api.withdo.si through a Cloudflare Tunnel

`api.withdo.si` is a proxied Cloudflare record that points at a **remotely managed** tunnel
(`withdo`): its ingress is kept at Cloudflare, not in a file on the host. `cloudflared` on the host
dials out to Cloudflare and hands every request to nginx on `127.0.0.1:8088`
(`nginx-api-withdo.conf`), which proxies to `adhd-sync` on `127.0.0.1:18080` as for the legacy name.
Cloudflare terminates TLS, so there is no certificate to renew on the host for this name. The
`withdo.si` zone must be on the Cloudflare account that owns the tunnel. Steps 1 to 3 run on the
owner's Mac with the `cf` CLI, signed in to that account; the rest on the host.

1. **Create the tunnel** (config kept at Cloudflare):
   `cf tunnels create --name withdo --config-src cloudflare`. Note the tunnel id it prints.
2. **Set the ingress** with `cf tunnels config update` (see `cf tunnels config update --help` for how
   it takes the configuration): `api.withdo.si` goes to `http://127.0.0.1:8088`, and the last rule
   is `http_status:404`, so anything else that reaches the tunnel gets a 404:
   ```json
   {"ingress": [
     {"hostname": "api.withdo.si", "service": "http://127.0.0.1:8088"},
     {"service": "http_status:404"}
   ]}
   ```
3. **Route the name**: a proxied CNAME record `api.withdo.si` -> `<tunnel id>.cfargotunnel.com`,
   made with `cf dns records create` (type CNAME, proxied on).
4. **nginx on the host**: copy `deploy/sync/nginx-api-withdo.conf` to
   `/etc/nginx/sites-available/api-withdo`, link it from `sites-enabled`, then `nginx -t` and
   `systemctl reload nginx`. Check the listener: `curl -s http://127.0.0.1:8088/healthz` is `ok`.
5. **Install cloudflared** from Cloudflare's apt repository (Debian 12):
   ```sh
   sudo mkdir -p --mode=0755 /usr/share/keyrings
   curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg | sudo tee /usr/share/keyrings/cloudflare-main.gpg >/dev/null
   echo 'deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main' | sudo tee /etc/apt/sources.list.d/cloudflared.list
   sudo apt-get update && sudo apt-get install cloudflared
   ```
6. **Run it as a service** with the tunnel's token. The token is a secret: pipe it from `cf tunnels
   token get` to the host over ssh, so it is never printed, written to a file in the repo or put in
   shell history. For example, from the Mac (replace `<tunnel>` with the tunnel's id or name as the
   `cf` command takes it):
   `cf tunnels token get <tunnel> | ssh adhd-srv 'read -r token && sudo cloudflared service install "$token"'`.
   That writes and starts the `cloudflared` systemd unit. `systemctl status cloudflared` should say
   active and the tunnel shows as healthy at Cloudflare.
7. **PocketBase** (admin UI, over the SSH tunnel below > Settings > Application): set the app URL to `https://api.withdo.si`.
   Once `withdo.si` is verified in Resend (DKIM and the other records Resend gives, added in
   Cloudflare), set the sender to `Withdo <noreply@withdo.si>` (Settings > Mail) and use a send-only
   key for that domain. The English code-mail template stored in PocketBase still ends with
   `todo.sri-ai.xyz`; change its footer to `withdo.si` in the same place (Collections > users >
   Options > Mail templates > OTP).
8. **Google sign-in** (Google Cloud console, the OAuth client): add
   `https://api.withdo.si/api/oauth2-redirect` as an authorized redirect URI (keep the old
   `https://todo.sri-ai.xyz/api/oauth2-redirect` while old clients exist) and `withdo.si` as an
   authorized domain on the consent screen.

The legacy name keeps being served directly by nginx (`nginx-adhd-todo.conf`) until old clients have
updated; new clients migrate a saved `todo.sri-ai.xyz` to `api.withdo.si` when they start. Then its
DNS record, its nginx file and its certificate can go.

Visitor addresses: `nginx-api-withdo.conf` takes the address from `CF-Connecting-IP`, trusted only
from loopback (`set_real_ip_from 127.0.0.1`), so `X-Real-IP`, and PocketBase's per-address rate
limit, count the visitor and not `127.0.0.1`. The check is in "Checks" below.

## From scratch (the legacy name, todo.sri-ai.xyz)

1. **DNS**: an A record for `todo.sri-ai.xyz` to the host, DNS only.
2. **Certificate**: install the port-80 server block of `nginx-adhd-todo.conf` alone, `nginx -t`,
   reload, then
   `certbot certonly --webroot -w /var/www/certbot --cert-name adhd-todo -d todo.sri-ai.xyz --deploy-hook "systemctl reload nginx"`.
3. **nginx**: install the whole `nginx-adhd-todo.conf`, `nginx -t`, reload.
4. **adhd-sync**: build `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/adhd-sync` into
   `sync/bin/`, write `sync.env`, create `sync/pb_data` (see below) and
   `docker compose --env-file sync.env -f sync.compose.yml up -d --wait`.

Old clients use `server https://todo.sri-ai.xyz`, new ones `https://api.withdo.si`; `GET /v1/config` answers `{"auth":"pocketbase","llm":true}` and
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
- **nginx** proxies PocketBase's user sign-in and realtime stream. On `api.withdo.si` the **admin is
  not public**: `/_/` and `/api/collections/_superusers/` answer 404. Reach the admin UI over an SSH
  tunnel to the server's own port: `ssh -N -L 18080:127.0.0.1:18080 adhd-srv`, then open
  `http://localhost:18080/_/`. Only the legacy `https://todo.sri-ai.xyz` block (`nginx-adhd-todo.conf`)
  still serves the admin UI publicly (decided 2026-10-22), until that name is retired. Both files set
  `X-Real-IP` from the connection on every location (behind the tunnel: from `CF-Connecting-IP`), so a
  client cannot choose the address the rate limits count.
- **Realtime pings.** `/api/realtime` sends nothing after `PB_CONNECT` until an event is subscribed,
  and Cloudflare closes an idle stream after about 100 s, while the Google sign-in waits on it for up
  to 5 minutes. `pbauth/realtime.go` sends a `PB_PING` event every 30 s on every realtime stream;
  clients ignore events they did not subscribe to.
- **Password guessing** is limited to 10 user sign-ins and 5 superuser sign-ins a minute per client address (PocketBase's rate limit,
  counting by the `X-Real-IP` nginx sets).
- **Mail** goes out through Resend's SMTP (`smtp.resend.com:465`, TLS, user `resend`, the password a
  send-only API key for `sri-ai.xyz`), from `Withdo <noreply@sri-ai.xyz>` until `withdo.si` is verified
  in Resend, then from `Withdo <noreply@withdo.si>` (step 7 above). It is set in
  PocketBase's settings (admin UI > Settings > Mail), not in this repo; the rename migration
  (`pbauth/migrate_withdo.go`) turns a sender name of `ADHD Todo` into `Withdo`. The key is also in the owner's
  keychain as `adhd-todo-resend-smtp`. The domain's DKIM, the `send` and `rsend` CNAMEs and DMARC are
  Cloudflare records Resend gave (2026-10-22); no MX, so Resend does not receive mail for the domain.
  The app URL in the same settings is `https://api.withdo.si`, for the links in the emails.
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

## The official model

`POST /v1/llm/chat/completions` is the model the app's input uses for anyone signed in (M15). It is
an OpenAI-compatible chat completions request, forwarded to the upstream with the server's key; the
server sets `model` to its own and returns the answer (streamed or not) unchanged. The app never
holds the key. Environment variables, passed through from `sync.env` by `sync.compose.yml`:

| Variable | Meaning |
|---|---|
| `LLM_API_KEY` | The upstream's key. Empty or unset turns the official model off: the route answers `503 llm_disabled` and `GET /v1/config` says `"llm": false`, so apps do not offer it. |
| `LLM_BASE_URL` | The upstream's API root; default `https://api.deepseek.com` (the route is `<root>/chat/completions`). |
| `LLM_MODEL` | The upstream model, whatever the app names; default `deepseek-chat`. |
| `LLM_DAILY_TURNS` | Turns per free account per UTC day; default 20. |
| `LLM_DAILY_TURNS_PRO` | Turns per Pro account (Withdo Pro, below) per UTC day; default 100. |

A turn is one thing the user sent, however many model calls it takes: the app sends
`X-Withdo-Turn: <uuid>` on every request of one turn, and the server counts distinct ids per account
per day. A request without the header is a turn of its own. A turn may make at most 6 requests, and
an account at most its daily turns × 6 a day. Past the quota: `429 {"code":"llm_quota"}` with
`Retry-After`. Every answer carries `X-Withdo-Turns-Left`; `GET /v1/llm/quota` returns
`{"limit":100,"left":86,"resets_at":"2026-11-04T00:00:00Z","plan":"pro"}` (the account's own limit and
plan, `free` or `pro`). Counts are in memory, so a restart
resets them. A request is given back when the upstream is unreachable, refuses the server's key
(502 `llm_upstream`) or is busy (429 `llm_busy`, not the account's quota).

The request body is at most 1 MiB and the upstream gets 90 s. Requests and replies are never
logged; the log line has the account, the turns left and the status. Put the key only in
`sync.env` on the host (mode 600), never in the repository. Streamed replies need nginx's
`proxy_buffering off` on `/v1/llm/` (see `nginx-api-withdo.conf`).

## Withdo Pro (M16)

Pro raises the official model's daily turns from `LLM_DAILY_TURNS` to `LLM_DAILY_TURNS_PRO`; sync stays
free. Waffo Pancake is the merchant of record (see `docs/plans/m16-billing.md`). The server makes the
checkout page (`POST /v1/billing/checkout`), receives Waffo's webhooks (`POST /v1/billing/waffo`) and
reports the plan (`GET /v1/billing`). Without `WAFFO_KEY_FILE` billing is off: those routes answer
`503 billing_disabled` and `GET /v1/config` says `"billing": false`, so apps do not offer Pro.

| Variable | Meaning |
|---|---|
| `WAFFO_KEY_FILE` | Path **inside the container** of the API key's private key (PKCS#8 PEM), under `/secrets`, e.g. `/secrets/waffo-test.pem`. Empty turns billing off. |
| `WAFFO_MERCHANT_ID` | The merchant (`MER_…`). |
| `WAFFO_STORE_ID` | The store (`STO_…`); events of other stores are ignored. |
| `WAFFO_PRODUCT_MONTHLY`, `WAFFO_PRODUCT_YEARLY` | The two subscription products (`PROD_…`). |
| `WAFFO_MODE` | `test` (default) or `prod`. Webhooks of the other mode are acknowledged and ignored. |

Test mode (set up 2026-11-04) in `sync.env`:

```text
WAFFO_KEY_FILE=/secrets/waffo-test.pem
WAFFO_MERCHANT_ID=MER_39eInAPdm3tF3TfUVkZyBx
WAFFO_STORE_ID=STO_5sXFc60AlpLP2ifCBlbxGm
WAFFO_PRODUCT_MONTHLY=PROD_0MGY2QF0efduP3znHPthbD
WAFFO_PRODUCT_YEARLY=PROD_52fSnkiAZRUesjicT4ptwh
WAFFO_MODE=test
```

The key file is on the host at `/opt/adhd-todo/sync/secrets/waffo-test.pem` and cannot be downloaded
again from Waffo. The container runs as uid 65532, so the file and its folder must belong to it:
`chown -R 65532:65532 secrets && chmod 700 secrets && chmod 600 secrets/*.pem`. A missing or
unreadable file stops the server at start with a clear error (it does not start with billing silently
off). The Waffo webhook (Raw format, all events) points at `https://api.withdo.si/v1/billing/waffo`; it
is verified with Waffo's public keys, committed as `internal/syncserver/billing/waffo-{test,live}.pub.pem`,
chosen by the payload's `mode` (45 minute tolerance, events deduplicated by id). Test card:
`4576 7500 0000 0110`.

Going live: create the products in production mode (or publish them), a production API key and a
production webhook, put the new key next to the old one (`waffo-live.pem`), and change `WAFFO_KEY_FILE`,
`WAFFO_PRODUCT_*`, `WAFFO_STORE_ID` if it differs, and `WAFFO_MODE=prod`; then
`docker compose --env-file sync.env -f sync.compose.yml up -d --force-recreate sync`.

Logs carry the event type, the account id and the outcome only; request bodies, keys, emails and
signatures are never logged.

## Checks

- `curl https://api.withdo.si/v1/config` is `{"auth":"pocketbase","llm":true,"billing":true}` (`false` while `LLM_API_KEY` / `WAFFO_KEY_FILE` is empty).
- `curl -s -o /dev/null -w '%{http_code}' -X POST https://api.withdo.si/v1/billing/waffo -d '{}'` is 401 (no signature), 503 with billing off.
- `curl https://api.withdo.si/healthz` is `ok`; `POST /v1/devices` without a token is 401.
- `/v1/events` stays open: `curl -N -H "Authorization: Bearer $TOKEN" https://api.withdo.si/v1/events`
  prints a `: keepalive` line every 15 s and is still connected after more than 2 minutes.
- The visitor's address reaches PocketBase: `tail -f` nginx's access log with `$remote_addr` (the
  default format) while requesting from outside shows your address, not `127.0.0.1`.
- The admin is closed on the new name: `curl -s -o /dev/null -w '%{http_code}' https://api.withdo.si/_/` and
  `.../api/collections/_superusers/auth-with-password` (POST) are 404.
- A `GET /api/realtime` stream shows `PB_CONNECT` and then `PB_PING` every 30 s, and is still open
  after more than 2 minutes.
- The legacy name still answers the same three on `https://todo.sri-ai.xyz` until its record goes.
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
