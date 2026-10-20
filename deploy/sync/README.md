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
    ├── sync.env                SYNC_DB_PASSWORD, OIDC_ISSUER, OIDC_AUDIENCE (= PROJECT_ID), mode 600
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
     path is refused.
6. **adhd-sync**: build `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/adhd-sync` into
   `sync/bin/`, write `sync.env` (audience = the project id), and
   `docker compose --env-file sync.env -f sync.compose.yml up -d --wait`.

Clients use `server https://todo.sri-ai.xyz`, `issuer https://auth.todo.sri-ai.xyz`, the app's
client id, and the scopes `openid offline_access urn:zitadel:iam:org:project:id:{PROJECT_ID}:aud`
(the last one puts the project in the token's audience).

## Checks

- `curl https://auth.todo.sri-ai.xyz/.well-known/openid-configuration` names the issuer and S256.
- `curl https://todo.sri-ai.xyz/healthz` is `ok`; `POST /v1/devices` without a token is 401.
- A client-credentials JWT of a throwaway machine user (access token type JWT, scope with the
  project audience) registers a device; the same token with one character changed is 401.
  Delete the machine user afterwards.

## Upgrades

- adhd-sync: build, copy over `bin/adhd-sync`, `docker compose … up -d --force-recreate sync`.
  Its Postgres migrations run at start.
- Zitadel: bump `ZITADEL_VERSION` in `.env`, then `pull` and `up -d --wait` with the same files.
