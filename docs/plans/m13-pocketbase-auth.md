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
| P4 | Remove Zitadel: code, config, containers (done 2026-10-22; the server steps are the owner's, below) | A grep for zitadel finds only this plan and the old mockups; the Zitadel stack is stopped |

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
  sign-ins a minute per address, and since the mail sender (P2) a code mailed to them as a second
  step (MFA). Superuser created
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
   transaction and refuses when the target already has a workspace. It also records the Zitadel id in
   `subject_aliases` (`-keep-alias=true`, the default), because installed older apps only know Zitadel:
   without the alias their next request would create a new empty user and the device id would no
   longer match. `ensureWorkspace` looks a subject up in `users`, then in `subject_aliases`.
   `adhd-sync add-alias` backfills an account moved before this existed; `list-subjects` shows
   `user_id subject kind devices rows`.
3. Sign in on desktop and phone; their data is there.

Tests: the authenticator with a test PocketBase app (`tests.NewTestApp`), the chain order, `move-subject`
on the Postgres test container, `PocketBaseSource` against an httptest PocketBase (sign-in, refresh,
logout, wrong password, rate limit), and the existing sync acceptance test signed in through
PocketBase.

## P2: sign-up, email codes, password reset

Needs an SMTP sender. Done 2026-10-22: Resend, sending from `noreply@sri-ai.xyz` (domain verified,
DNS records added with the owner's `cf` CLI), set in PocketBase's mail settings, not in our code; a
test email was delivered. Superuser MFA is on. What is left is below.

Decided 2026-10-22 on `docs/design/mockups/v32.html` (ui-spec Settings > Account):
- One path for sign-up and sign-in: email → 6-digit code. PocketBase sends no code to an email with
  no account, so an `OnRecordRequestOTPRequest` hook creates the user first (random password, not
  verified, name = the part before `@`); signing in with the code marks it verified.
- `users` OTP on, length 6, 600 s; the OTP email template in English only. A Go migration sets
  these, like the superuser rate limit, so a fresh `pb_data` gets them.
- Rate limits: `users:requestOTP` 3 a minute, `users:authWithOTP` 10 a minute per address.
- A password stays optional (`Use a password instead`), with PocketBase's reset email and its own
  reset page on the server.
- Not now: deleting accounts that never used their code.

## P3: Google

Started 2026-10-22 (owner: go). Google Cloud project `adhd-todo-512407`, consent screen "ADHD Todo"
(external, in testing: only listed test users until it is published, which needs a homepage and a
privacy policy page), a Web client whose only redirect URI is
`https://todo.sri-ai.xyz/api/oauth2-redirect`; its secret is in the owner's keychain as
`adhd-todo-google-oauth` and goes into PocketBase's `users` OAuth2 settings, not into this repo.

PocketBase's own realtime flow, as its JS SDK does it, so no loopback server and no URL scheme: the
client opens `/api/realtime`, subscribes to `@oauth2`, takes Google's auth URL from `auth-methods`
with `state` set to its realtime client id and `redirect_uri` the server's `/api/oauth2-redirect`,
opens the system browser (a Custom Tab on Android), waits for the `@oauth2` message with the code,
then calls `auth-with-oauth2` with the code and the code verifier. PocketBase checks that the browser
and the app come from the same address. New users: `users` create rule
`@request.context = "oauth2"`, so OAuth2 can create accounts and the plain create endpoint still
cannot; an email that has an account signs in to it. This reverses the "no browser for sign-in"
rule of 2026-10-20 for Google only; email sign-in stays in the app.

## P4: remove Zitadel (done 2026-10-22)

The owner approved it on 2026-10-22, before the last devices had signed in again; a device still
signed in with Zitadel is signed out and signs in again through PocketBase.

- Deleted `internal/syncserver/{signin,zitadel,oidc}.go`, `internal/syncclient/{oidc,signin}.go` and
  their tests, the OIDC fields of `/v1/config` and `syncproto` (it answers `{"auth":"pocketbase"}`),
  the Zitadel and OIDC flags of `adhd-sync` and `adhd-core sync login`, and `deploy/sync/zitadel*`.
  The server's chain is PocketBase, then the static tokens.
- Clients: an old `sync.json` with an `issuer` loads as a signed-out PocketBase config (no error, no
  account shown) and the Account pane shows the email card. The password-only card of the desktop
  and Android's `LEGACY` step are gone, with their screenshots.
- Kept: `subject_aliases` and `add-alias`, `drop-alias`, `move-subject`, `list-subjects` (the data
  model; harmless). Drop the aliases of the two accounts when no old device remains.
- On the server, by the owner, after a final backup: stop and remove the Zitadel compose stack
  (`/opt/adhd-todo/zitadel`), remove `sync.env`'s `OIDC_*` lines and `login-client.pat`, install the
  new `nginx-adhd-todo.conf` (no `auth.todo.sri-ai.xyz` block), delete the `auth.todo` DNS record, and
  shrink the certificate to `todo.sri-ai.xyz`.

## v42 (2026-10-29): sign up with a password

The email-and-code path (an account made by the first code request) is replaced by two ways in:
Sign in and Sign up, as Todoist and TickTick do it (`docs/design/mockups/v42.html`). Server side:

- Migration `1790000500_adhd_users_signup.go`: the users create rule is `@request.context = "oauth2"
  || (@request.body.id:isset = false && @request.body.verified:isset = false &&
  @request.body.tokenKey:isset = false)`, so anyone may `POST /api/collections/users/records` with an
  email and a password (at least 8 characters, the password field's `min`), and no client sets
  `verified` (PocketBase's record form refuses it too: a `verified: true` body is a 400). A new
  optional text field `lang` (`en` or `zh-Hans`) keeps the app language. Rate limits `users:create`
  5 a minute and `/api/withdo/reset-password` 10 a minute, beside the existing ones.
- `createUserOnCodeRequest` is gone: `request-otp` for an email with no account is PocketBase's own
  answer (a random `otpId`, no mail). A hook only looks the email up in lower case, which is how
  accounts are stored (a create lower-cases it and names the user after the part before the `@`).
- Unverified users get no token from `auth-with-password`: the right password answers 403 with
  `data.email.code = "email_not_verified"` (a wrong one is the usual 400), and the client then asks
  for a code (`request-otp`) and shows Verify. `auth-with-otp` works for them and verifies the
  account; PocketBase would also replace the password with a random one at that moment
  (pre-hijack protection for the code-only sign-up), which a hook skips because the user chose a
  password. The reset route below replaces a password an impostor chose.
- `POST /api/withdo/reset-password` `{otpId, code, password, passwordConfirm}`: checks the OTP like
  `auth-with-otp` (it exists, belongs to users, is not expired, matches, was mailed to the account's
  own email; five tries in three minutes per user), sets the password, verifies the email, deletes
  the user's OTPs and answers with a token and the record. A short password is a 400 with
  `data.password.code = "validation_min_text_constraint"` (checked first, it says nothing about the
  account); everything else wrong, and an `otpId` of an email with no account, is the same 400
  "Invalid or expired OTP".
- The code mail follows `lang` (`OnMailerRecordOTPSend`): English is the collection's template,
  `zh-Hans` is `{OTP} 是你的 Withdo 验证码`. A Google account has no `lang` and gets English.
- Old accounts made by the code path have a password nobody knows; they use Forgot password.
- Deploy: the new migration runs at start; nothing to configure. The site needs a `terms.html` the
  Sign up card links to.

## Owner's to-dos

- P1: nothing beyond approving the deploy.
- P2: done (Resend, 2026-10-22).
- P3: a Google Cloud OAuth client, and a yes to the browser on desktop.
- After P1: `drop schema adhd_todo;` in Supabase picpocket, if wanted.
