# M11 Multi-device sync

Desktop and Android keep their own SQLite database and sync through a self-hosted server. Later
there are many users, each with their own account and data.

## Decisions (2026-10-19)

- **Server-authoritative, offline-first**, in the manner of PowerSync, Replicache and Linear: a
  device writes locally, queues the change, uploads it; the server checks it, gives it a version
  and serves the changes after a cursor. None of those engines has a Go client, so we implement
  the protocol in Go once, and desktop and Android (gomobile) share that code.
- **Accounts**: one account per person; data never crosses accounts. Every synced row belongs to a
  workspace (one per user for now), so shared workspaces can come later.
- **Login**: self-hosted OIDC (Zitadel), PKCE through the system browser on desktop and Android.
- **Transport**: HTTPS on a subdomain the user provides, certificates by Caddy.
- **Server storage**: Postgres; every row keyed by `(workspace_id, entity, id)` with a version.
- **IDs**: the integer ids stay the ids, so "#44" means the same task on every device. Once a device
  is linked, it takes new ids from blocks the server leases to it per workspace, so it can create
  rows offline without collisions. Ids have gaps; that is accepted.
- **Conflicts**: per column, last write wins in the order the server accepts writes. A downloaded
  row does not overwrite a local row that still has changes waiting to upload.
- **Device-local columns never sync**: `projects.repo_path`, `reports.cwd`, `tasks.reminded_at`.
  Secrets stay in the keychain.
- PR QingYunA/adhd-todo#2 (Expo) is not continued; its `internal/mobilesync` ideas (idempotent
  receipts, tombstones, cursors) inform the server.

## Phases

| Phase | Scope | Verified by |
|---|---|---|
| S1 | Local change capture and id blocks (this document) | tests below |
| S2 | `cmd/adhd-sync` server: push, pull, id blocks, Postgres, auth stub | two Cores + httptest converge after random operations |
| S3 | Desktop sync client, linking a device, Settings | two app instances on different `ADHD_TODO_DB` see each other's changes |
| S4 | Deploy with Caddy and Zitadel | S3's test against the real server |
| A1–A3 | Android: gomobile core, Compose UI from approved mockups, end to end | CI build, emulator screenshots, phone ↔ desktop |

## S1: local change capture and id blocks

No networking. After S1 a linked database records every synced change in an outbox, and new rows
take ids from leased blocks; an unlinked database behaves exactly as today.

Synced tables in S1: `projects`, `milestones`, `tasks`, `reports`, `categories`. (`images`,
`task_images` and `summaries` come in S3.)

### Migration `014_sync.sql`

- `sync_state(key TEXT PRIMARY KEY, value TEXT NOT NULL)`. Keys: `enabled` (`'1'` once linked),
  `applying` (`'1'` while changes from the server are written).
- `sync_outbox(seq INTEGER PRIMARY KEY, entity TEXT NOT NULL, row_id INTEGER NOT NULL,
  op TEXT NOT NULL CHECK (op IN ('put','delete')), columns TEXT, created_at TEXT NOT NULL)`.
  `columns` is a JSON array of the synced columns an update changed; NULL for an insert (all
  columns) and a delete. `created_at` uses the existing time layout (`strftime('%Y-%m-%dT%H:%M:%fZ','now')`).
- `sync_id_blocks(entity TEXT NOT NULL, next INTEGER NOT NULL, end INTEGER NOT NULL)`: the half-open
  range `[next, end)` a linked device may still use for that entity. Several rows per entity are
  allowed; use the one with the smallest `next`.
- For each synced table, `AFTER INSERT`, `AFTER UPDATE` and `AFTER DELETE` triggers append to
  `sync_outbox`, only `WHEN` `enabled` is `'1'` and `applying` is not `'1'`.
  - The update trigger fires only when at least one synced column changed (`OLD.c IS NOT NEW.c`),
    and lists exactly those columns in `columns`. Device-local columns are not synced columns, so
    changing only `reminded_at`, `repo_path` or `cwd` records nothing.
  - Synced columns are every column of the table except `id` and the device-local ones above.

### Domain

- `internal/domain`: a helper that gives the id for a new row of an entity inside the write
  transaction: when the database is linked, the next id of the entity's block (advance `next`;
  delete the row when it is used up); when unlinked, nil, so SQLite assigns it as today.
  - When linked and no block is left, the write fails with an exported error `ErrNoSyncIDs`
    (S3 refills blocks before they run out).
- Use it in the six INSERTs that create new rows: `tasks.go` (insertTask), `milestones.go`,
  `categories.go` (the user-category insert, not the built-in seed), `projects.go`, `reports.go`.
  `restore.go` keeps reinserting with the original ids.
- An exported way to run a write with `applying` set, for S3 to write server changes, which
  resets it in the same transaction.
- An exported way to read and to delete outbox entries up to a `seq`, for S3's uploader.
- Tests may set `enabled` and insert blocks with plain SQL; S1 needs no public link API.

### Tests (the acceptance criteria)

1. **Every write path is captured.** A table-driven test runs each exported `Core` method that
   writes a synced table on a linked database and checks the outbox entries it adds (entity,
   row_id, op, columns). It covers at least: create, update, complete, delete and restore a task;
   bulk update and bulk delete; add a report (including the handoff clear); create, rename, archive,
   reparent and delete a project (cascading deletes of its tasks, reports and milestones); create
   and update milestones; create, update, reorder and delete categories (tasks' `category_id` set
   to NULL); subtask auto-completion; recording PR state; handing off and accepting a task.
2. **Local-only changes record nothing**: claiming a due reminder, a project's `repo_path` update
   from git, a report's `cwd`.
3. **Unlinked or applying records nothing**: the same writes with `enabled` unset, and inside the
   applying write, add no outbox rows; the applying flag is off after that write, also when it fails.
4. **Id blocks**: on a linked database with a block `[1000,1003)`, three new tasks get 1000, 1001,
   1002, the block row is gone, and the fourth fails with `ErrNoSyncIDs` and writes nothing. An
   unlinked database still assigns ids as before. Two blocks are used smallest first.
5. **Migration**: a database at version 13 with data migrates to 14 and keeps its data
   (`internal/store` already has `migrateTo` for this).
6. Concurrency: two Cores on one file (as `internal/domain/concurrency_test.go`) both writing on a
   linked database never hand out the same id.

All existing tests keep passing; `scripts/check.sh full` exits 0.

### As built (S1)

- Five INSERT sites take block ids (`insertTask`, milestones, user categories, `insertProject`, reports);
  `restore.go` keeps the original ids.
- `findOrCreateByGit` no longer touches `projects.updated_at` when only `repo_path` changes, so a
  device-local change records nothing. Nothing sorts projects by `updated_at`.
- There is no exported project reparent; `CreateProject` with a parent covers `parent_id`.
- `Outbox(upTo)` / `DeleteOutbox(upTo)` read and acknowledge through a seq; `ApplyChanges` runs a
  write with `applying` set and clears it in the same transaction.

### Out of scope for S1

Networking, linking a device, the server, images and summaries, Settings UI, Android.

## S2a: the sync server

A Go HTTP service, `cmd/adhd-sync`, with its logic in `internal/syncserver` and the wire types in
`internal/syncproto` (shared with the S2b client). Postgres through `github.com/jackc/pgx/v5`
(`pgxpool`). No client, no UI, no OIDC yet.

### Data (Postgres, migrations embedded and applied at start, like `internal/store`)

- `users(id BIGSERIAL PK, subject TEXT UNIQUE NOT NULL, created_at)`: `subject` is the OIDC `sub`.
- `workspaces(id BIGSERIAL PK, owner_id → users, created_at)`: one per user for now, made on the
  user's first request.
- `devices(id TEXT PK, workspace_id, user_id, name, last_mutation_id BIGINT NOT NULL DEFAULT 0,
  created_at)`: `id` is a random UUID the server makes.
- `rows(workspace_id, entity TEXT, row_id BIGINT, data JSONB, deleted BOOL NOT NULL DEFAULT false,
  version BIGINT NOT NULL, PK (workspace_id, entity, row_id))`, index `(workspace_id, version)`.
- `workspace_versions(workspace_id PK, version BIGINT NOT NULL)`: the workspace's clock; each push
  transaction takes the next value for every row it changes.
- `id_counters(workspace_id, entity, next BIGINT NOT NULL, PK (workspace_id, entity))`.

### Entities and columns

The server knows the five S1 entities and, for each, its synced columns (the S1 trigger lists;
put them once in `internal/syncproto`, with a test that compares them with the triggers of
migration 014 so the two cannot drift). Anything else is rejected with 400.

### Auth

`Authenticator` interface: request → user subject, or 401. S2a ships `StaticTokens` (a map of bearer
token → subject, from a file named by a flag) for development and tests; S4 adds OIDC. Every
query is scoped to the caller's workspace; a device id that belongs to another user is 403.

### Endpoints (JSON; all need a bearer token)

| Method and path | Request | Response |
|---|---|---|
| `POST /v1/devices` | `{name}` | `{device_id, workspace_id, empty}` — `empty` is true when the workspace has no rows yet |
| `POST /v1/ids` | `{device_id, entity, count}` (1–10000) | `{start, end}` — a fresh half-open block from `id_counters` |
| `POST /v1/push` | `{device_id, mutations: [{id, entity, row_id, op, data}]}` | `{last_mutation_id}` |
| `GET /v1/pull?device_id=&cookie=&limit=` | | `{cookie, last_mutation_id, changes: [{entity, row_id, deleted, data, version}], more}` |
| `GET /v1/events?device_id=` | | SSE: an event `poke` when the workspace's version rises |
| `GET /healthz` | | `ok`, no auth |

Push rules (Replicache's, adapted):
- Mutations are applied in order in **one transaction**, with the device's `last_mutation_id`
  updated in the same transaction. A mutation whose `id` is not above `last_mutation_id` is skipped
  (already applied); ids must rise within a request, else 400 and nothing is applied.
- `put` merges `data` (column → value, only synced columns of the entity, else 400) into the row's
  `data`, column by column (last write wins in server order), and sets `version`. A put on a row that
  is deleted is ignored (delete wins). A put of a new row needs `row_id` below the workspace's
  `id_counters.next` for that entity, or the workspace empty-snapshot rule below; else 409.
- `delete` marks the row deleted (`data` NULL), keeps it as a tombstone, sets `version`.
- **Snapshot**: while a workspace has no rows, the first device may push puts with any ids; after
  that push, `id_counters.next` for each entity is raised above the largest id seen.
- Request bodies are limited (8 MiB); a push is at most 1000 mutations.

Pull rules: changes with `version > cookie` in version order, at most `limit` (default 500, max
1000); `cookie` in the response is the last change's version (or the current version when there are
none), `more` says whether more remain. `last_mutation_id` is the calling device's, read in the same
transaction as the changes. A cookie above the workspace's version is 409 (the client resets).

### Tests (acceptance)

Postgres tests run when `ADHD_SYNC_TEST_DATABASE_URL` is set and are skipped otherwise; each test
uses its own schema or database so they can run in parallel. CI gets a `postgres:17` service and
sets the variable.

1. **Isolation**: two users' devices never see or change each other's rows; another user's device
   id is 403; a missing or unknown token is 401.
2. **Push idempotency**: the same push sent twice applies once; a lower or equal mutation id is
   skipped; falling ids in one request are 400 and change nothing.
3. **Column merge**: two devices put different columns of one row; both survive. Two devices put the
   same column; the later push wins.
4. **Delete wins**: a put after a delete leaves the row deleted.
5. **Ids**: blocks from `POST /v1/ids` never overlap, across devices and concurrent requests; a put
   of a new row outside any leased range is 409; the snapshot push raises the counters.
6. **Pull**: paging with `limit` returns every change once, in version order, with a correct `more`;
   a cookie from the future is 409; `last_mutation_id` matches the push.
7. **Columns**: an unknown entity or column is 400; the column lists match the triggers of
   migration 014.
8. **Events**: a push to the workspace wakes an open `/v1/events` stream of another device of it,
   and not one of another workspace.
9. `cmd/adhd-sync` starts with `-addr`, `-database-url`, `-tokens`; `GET /healthz` answers.

`scripts/check.sh full` exits 0 with and without the Postgres variable.
