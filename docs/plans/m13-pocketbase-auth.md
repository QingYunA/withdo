# M13 Accounts on PocketBase

Sign-in moves from the self-hosted Zitadel to PocketBase, embedded in `adhd-sync` as a Go library.
Sync, its protocol and its Postgres stay as they are.

## Why (2026-10-21)

- The product is for people outside China, and the owner wants as little hand-written code as
  possible. Zitadel needed about 1240 lines of our own (in-app sign-in through its session API, our
  own rate limit, PKCE plumbing) plus three compose files, a service account and an admin account,
  and it still has no self sign-up, no email codes and no password reset wired in.
- Supabase was tried first. Its free tier was full, so it would have shared the picpocket project:
  the same user table, email templates and Google consent screen for two products. A new project
  costs a paid plan. (An empty `adhd_todo` schema was made in picpocket on 2026-10-21; drop it with
  `drop schema adhd_todo;` once this lands.)
- PocketBase is Go, runs inside the process we already deploy, and brings sign-up, password, email
  one-time codes, OAuth2 (Google), password reset, email templates and an admin UI. Its users live in
  its own SQLite file; nothing else moves.
- Risk: one main author (2501 of 2557 commits) and pre-1.0 (v0.40.5). We pin the version in `go.mod`
  and read the release notes before each upgrade; being a library, it can be forked if it stalls.

## What stays

- `syncserver.Authenticator` (`Subject(r) (string, error)`) and `Chain`: every route, voice included,
  already goes through them. PocketBase is one more Authenticator.
- `users.subject` → workspace. A user's data is found by subject, so moving an account is one
  `UPDATE users SET subject = …`.
- The sync protocol, `/v1/*` routes, Postgres, the desktop and Android clients' sync code.

## Phases

| Phase | Scope | Verified by |
|---|---|---|
| P1 | Embed PocketBase; password sign-in through it; both logins accepted; move the two accounts | tests; the owner signs in on desktop and phone with the same password as today |
| P2 | Sign-up, email codes and password reset (needs SMTP) | a new account made from the app, a code arrives |
| P3 | Google sign-in (needs a Google OAuth client; opens a browser on desktop) | sign in with Google on desktop and Android |
| P4 | Remove Zitadel: code, config, containers | `git grep -i zitadel` is empty; the Zitadel stack is stopped |

## P1: embed PocketBase, keep the sign-in screen

**Server (`cmd/adhd-sync`, `internal/syncserver`).**

- Build the PocketBase app from code (`pocketbase.NewWithConfig`, data dir `-pb-dir`, default next to
  the binary's data; in production `/opt/adhd-todo/sync/pb_data`, a Docker volume). Run its server
  instead of ours and mount our existing `http.Handler` for `/v1/` on its router in `OnServe`
  (`apis.WrapStdHandler`), so one listener serves both. Its own CLI is not used; `adhd-sync` keeps its
  flags. Check the exact bootstrap API against the pinned version before writing it.
- A Go migration creates the setup on first start: the built-in `users` auth collection with
  password auth on (identity: email), sign-up off for now, OTP off until P2; token duration as
  PocketBase defaults; the app name `ADHD Todo`; the sender left for P2.
- `PocketBaseAuthenticator.Subject`: the bearer token through `app.FindAuthRecordByToken(token,
  core.TokenTypeAuth)` from the `users` collection; the subject is `pb:<record id>`, so it can never
  equal a Zitadel subject.
- The server's authenticator becomes `Chain{PocketBase, OIDC(Zitadel), StaticTokens}` until P4.
- `/v1/config` gains `"auth": "pocketbase"` (old clients ignore it and keep using Zitadel until they
  update). The in-app `/v1/sign-in` keeps answering for old clients until P4.
- The admin UI (`/_/`) was first reachable only through `ssh -L`; since 2026-10-22 it is public at
  `https://todo.sri-ai.xyz/_/`, guarded by the superuser's random 32-character password and 5
  sign-ins a minute per address. MFA for superusers comes with the mail sender (P2). Superuser created
  once from the server with `adhd-sync pb-superuser`.
- Backups: `pb_data/data.db` joins what `deploy/sync/deploy.sh` backs up.

**Client (`internal/syncclient`, shared by desktop and Android through `mobile/core`).**

- `PocketBaseSource` implements the same token-source interface as `OIDCSource` (`Token`, `Logout`,
  `Account`, `SignIn(server, login, password)`): `POST /api/collections/users/auth-with-password`,
  keeps the token in the keychain (Keystore on Android) where the refresh token is today, and calls
  `auth-refresh` when less than a day is left. `Account()` comes from the returned record (email,
  name).
- `app/data` and `internal/cli` pick the source from `/v1/config`: `pocketbase` → `PocketBaseSource`,
  otherwise the OIDC one. Errors map to the existing `ErrInvalidCredentials` / `ErrTooManyAttempts`,
  so the desktop Account pane and Android's sign-in screen do not change in P1.
- A device signed in with Zitadel keeps working (the chain accepts its token) until it signs out; on
  the next sign-in it uses PocketBase.

**Moving the two existing accounts** (the owner's and `e2e@todo.sri-ai.xyz`):

1. `adhd-sync pb-import-user -email … -name …` creates the PocketBase user and asks for the password
   on stdin (the owner's current one, from the keychain item `adhd-todo-account`, so nothing changes
   for them). It prints the new subject.
2. `adhd-sync move-subject -from <zitadel user id> -to pb:<id>` rewrites `users.subject` in one
   transaction and refuses when the target already has a workspace.
3. Sign in on desktop and phone; their data is there.

Tests: the authenticator with a test PocketBase app (`tests.NewTestApp`), the chain order, `move-subject`
on the Postgres test container, `PocketBaseSource` against an httptest PocketBase (sign-in, refresh,
logout, wrong password, rate limit), and the existing sync acceptance test signed in through
PocketBase.

## P2: sign-up, email codes, password reset

Needs an SMTP sender. Recommended: Resend, sending from `noreply@sri-ai.xyz` (DNS records added with
the owner's `cf` CLI). Set in PocketBase's mail settings, not in our code.

- Turn on sign-up and OTP for `users`; edit the email templates (code, verification, reset) in
  PocketBase, in English and Chinese.
- The Account pane and Android's screen get a two-step flow: email → 6-digit code (plus "use a
  password instead"). A mockup comes first (`docs/design/mockups/`, drawn by Claude), then
  `ui-spec` Settings > Account.
- Rate limits: PocketBase's built-in rate limiter on the auth routes.

## P3: Google

Needs a Google Cloud OAuth client of our own (consent screen "ADHD Todo"). PocketBase's OAuth2 is the
code flow, so on desktop it opens the browser once and returns to a loopback address; Android uses a
Custom Tab. This reverses the "no browser for sign-in" rule of 2026-10-20 for Google only; email
sign-in stays in the app. Confirm with the owner before building.

## P4: remove Zitadel

- Delete `internal/syncserver/{signin,zitadel,oidc}.go`, `internal/syncclient/{oidc,signin}.go` and
  their tests, the OIDC fields of `/v1/config` and `syncproto`, the Zitadel flags, and
  `deploy/sync/zitadel*`.
- Only after every device of both accounts has signed in through PocketBase (the server logs which
  authenticator accepted each request; Zitadel's count reaching zero for a week is the signal).
- On the server: stop and remove the Zitadel compose stack and its nginx site
  `auth.todo.sri-ai.xyz`, after a final backup. The owner approves this step.

## Owner's to-dos

- P1: nothing beyond approving the deploy.
- P2: a Resend account (or another SMTP sender) and its API key.
- P3: a Google Cloud OAuth client, and a yes to the browser on desktop.
- After P1: `drop schema adhd_todo;` in Supabase picpocket, if wanted.
