# Sync server deployment (M11 S4b)

Production runs on one Debian 12 host that already serves other sites through nginx on 80/443.
Everything here is added beside them; nothing else on the host is changed.

| Name | What | Upstream |
|---|---|---|
| `todo.sri-ai.xyz` | `adhd-sync` (this repo's `cmd/adhd-sync`) | `127.0.0.1:18080` |
| `auth.todo.sri-ai.xyz` | Zitadel v4 (OIDC), its Traefik routes the API and the login UI | `127.0.0.1:18081` |

Both DNS records are A records, **DNS only** (not proxied by Cloudflare): certbot answers the
HTTP challenge itself, and the sync server's SSE stream would be cut by the proxy's timeout.

## Layout on the host

```text
/opt/adhd-todo/                 mode 700, root
├── zitadel/
│   ├── zitadel.compose.yml     upstream deploy/compose/docker-compose.yml (v4.16.0)
│   ├── zitadel.override.yml    loopback-only proxy, external TLS, admin service account
│   ├── .env                    generated secrets, mode 600 (masterkey, cookie, DB, admin password)
│   ├── admin-sa.pat            the provisioning service account's PAT, mode 600
│   └── provisioned.env         PROJECT_ID, CLIENT_ID of the adhd-todo app
└── sync/
    ├── sync.compose.yml        adhd-sync + its own Postgres
    ├── sync.env                SYNC_DB_PASSWORD, OIDC_ISSUER, OIDC_AUDIENCE (= PROJECT_ID), OIDC_CLIENT_ID,
    │                           OIDC_SCOPES (= urn:zitadel:iam:org:project:id:PROJECT_ID:aud), mode 600
    ├── login-client.pat        the login machine user's PAT, mode 400, owned by uid 65532
    └── bin/adhd-sync           static linux/amd64 build
/etc/nginx/sites-available/adhd-todo   nginx-adhd-todo.conf, linked from sites-enabled
/etc/letsencrypt/live/adhd-todo/       one certificate for both names; certbot.timer renews it
```

The Zitadel admin's password is also kept on the owner's Mac, in the keychain item
`adhd-todo-zitadel-admin` (account `zitadel-admin@zitadel.auth.todo.sri-ai.xyz`).

## From scratch

1. **DNS**: A records for both names to the host, DNS only.
2. **Certificate**: install the port-80 server block of `nginx-adhd-todo.conf` alone, `nginx -t`,
   reload, then
   `certbot certonly --webroot -w /var/www/certbot --cert-name adhd-todo -d todo.sri-ai.xyz -d auth.todo.sri-ai.xyz --deploy-hook "systemctl reload nginx"`.
3. **Zitadel**: copy `zitadel.*` to `/opt/adhd-todo/zitadel`, make `.env` from
   `zitadel.env.example` with `ZITADEL_DOMAIN=auth.todo.sri-ai.xyz`,
   `PROXY_HTTP_PUBLISHED_PORT=127.0.0.1:18081`, `ZITADEL_EXTERNALPORT=443`,
   `ZITADEL_EXTERNALSECURE=true`, `ZITADEL_PUBLIC_SCHEME=https`, fresh random values for every
   secret, `ADMIN_HUMAN_PASSWORD` (upper, lower, digit, symbol) and `ADMIN_SA_PAT_EXPIRATION`, then
   `docker compose --env-file .env -f zitadel.compose.yml -f zitadel.override.yml up -d --wait`.
   The masterkey cannot change after the first start.
4. **nginx**: install the whole `nginx-adhd-todo.conf`, `nginx -t`, reload.
5. **Provision** with the service account's PAT (read it from the `zitadel_zitadel-bootstrap`
   volume with a throwaway container; the Zitadel image has no shell):
   - `POST /management/v1/projects {"name":"adhd-todo"}`
   - `POST /management/v1/projects/{id}/apps/oidc` with `appType OIDC_APP_TYPE_NATIVE`,
     `authMethodType OIDC_AUTH_METHOD_TYPE_NONE` (PKCE), grant types authorization code and
     refresh token, `accessTokenType OIDC_TOKEN_TYPE_JWT` (the sync server verifies JWTs),
     `devMode true` and the redirect `http://127.0.0.1/callback`. Zitadel ignores the port of a
     loopback redirect for native apps (RFC 8252), so the client's random port works; another
     path is refused. Set `idTokenUserinfoAssertion true` too: without it the ID token has no
     name or email, and Settings shows no account.
   - `PUT /admin/v1/policies/login` with the current policy and `allowRegister false`: accounts
     are made by the admin, not by whoever finds the sign-in page.
6. **adhd-sync**: build `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/adhd-sync` into
   `sync/bin/`, write `sync.env` (audience = the project id), create `sync/login-client.pat` as in
   "In-app sign-in" below, and
   `docker compose --env-file sync.env -f sync.compose.yml up -d --wait`.

Clients use `server https://todo.sri-ai.xyz`, `issuer https://auth.todo.sri-ai.xyz`, the app's
client id, and the scopes `openid offline_access profile email
urn:zitadel:iam:org:project:id:{PROJECT_ID}:aud` (the last one puts the project in the token's
audience). `adhd-sync` asks for the first four plus the extra scopes in `-oidc-scopes`, and the
apps read the list from `GET /v1/config`.

## In-app sign-in

`adhd-sync` can complete the whole OIDC flow with an account and password, so clients never open a
browser. The server holds a machine user's PAT; it checks the password through the Session API and
returns the tokens. Its Zitadel calls go through the external `zitadel` Docker network that
`sync.compose.yml` joins (`-zitadel-internal-url=http://proxy`, with the issuer's host as `Host`),
not the public address, which keeps each call at a few milliseconds.

1. **Machine user.** With the admin PAT, `POST /management/v1/users/machine`:

   ```json
   {
     "userName": "adhd-sync-login",
     "name": "adhd-sync login",
     "description": "Completes in-app sign-in for adhd-todo",
     "accessTokenType": "ACCESS_TOKEN_TYPE_BEARER"
   }
   ```

2. **Only `IAM_LOGIN_CLIENT`.** `POST /admin/v1/members` with the new user's id:

   ```json
   {"userId": "<machine user id>", "roles": ["IAM_LOGIN_CLIENT"]}
   ```

3. **PAT with an expiry.** `POST /management/v1/users/{id}/pats`:

   ```json
   {"expirationDate": "2027-01-01T00:00:00Z"}
   ```

   The response contains the token once; copy it now. Do not commit it or paste it into this file.

4. **Install the PAT.** Write it to `/opt/adhd-todo/sync/login-client.pat`. The container runs as
   distroless `nonroot` (uid 65532), so give that user read access, for example
   `chown 65532:65532 login-client.pat && chmod 400 login-client.pat`. `sync.compose.yml` mounts
   the file read-only at `/run/secrets/login-client.pat` and passes
   `-zitadel-login-pat-file=/run/secrets/login-client.pat`.

5. **Restart adhd-sync.** `GET /v1/config` then answers `"password_sign_in": true` and
   `POST /v1/sign-in` is registered. Without the flag both are absent, which is the old behaviour.

Accounts with a second factor are refused: Zitadel would complete the auth request after the
password alone, so `adhd-sync` reads the user's authentication methods and deletes the session
before answering `403 second_factor`. Zitadel's lockout is not configured here, so `adhd-sync`
limits sign-in itself: 10 requests per 10 minutes per source IP and 10 failed passwords per login
name, both in memory.

## Checks

- `curl https://auth.todo.sri-ai.xyz/.well-known/openid-configuration` names the issuer and S256.
- `curl https://todo.sri-ai.xyz/healthz` is `ok`; `POST /v1/devices` without a token is 401.
- A client-credentials JWT of a throwaway machine user (access token type JWT, scope with the
  project audience) registers a device; the same token with one character changed is 401.
  Delete the machine user afterwards.

## Updating the server

`deploy/sync/deploy.sh` builds `adhd-sync` from the current checkout, copies it and
`sync.compose.yml` to the host, recreates the `sync` container (the binary is bind-mounted) and checks `GET /v1/config` on
`127.0.0.1:18080` for 200. The owner confirms each production deploy; do not run it unasked.

```sh
deploy/sync/deploy.sh --dry-run   # builds, prints every remote command, runs none
deploy/sync/deploy.sh             # ADHD_SYNC_HOST overrides the ssh host (default adhd-srv)
```

It first copies `bin/adhd-sync` and `sync.compose.yml` to `*.bak-YYYYMMDD-HHMM`, and never touches
`sync.env` or the Zitadel files. At the end it prints the one-line rollback command (restore both
backups, `up -d --wait sync`). If the health check fails it prints that command prominently and
exits 1; it does not roll back by itself.

## Upgrades

- adhd-sync: use `deploy/sync/deploy.sh` (above). Its Postgres migrations run at start.
- Zitadel: bump `ZITADEL_VERSION` in `.env`, then `pull` and `up -d --wait` with the same files.
