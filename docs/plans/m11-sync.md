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
- **Seeding** (the first device's snapshot, which spans many pushes): `POST /v1/devices` on a
  workspace with no rows and no seeding device registers the device with `devices.seeding = true`
  and answers `seeding: true`. A seeding device's puts of new rows may use any ids; every push
  raises `id_counters.next` above the largest id it wrote. While a workspace has a seeding device,
  every other device's push, pull and `POST /v1/ids` is 409 `workspace is seeding` (they retry).
  `POST /v1/devices/seeded {device_id}` (only the seeding device) ends it. A new registration by the
  same user while a device is seeding takes the seeding over (the old device stops seeding), so a
  client that crashed before it kept its device id can start again.
- JSON numbers are decoded with `UseNumber` everywhere the server merges or forwards `data`, so ids
  above 2^53 survive.
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

## S2b: the sync client

`internal/syncclient`: one Go library for the desktop now and Android (gomobile) later. It talks
to the S2a server over HTTP and to the local database only through `internal/domain` (S1's
`Outbox`, `DeleteOutbox`, `ApplyChanges`) plus plain SQL inside `ApplyChanges`. No UI and no
background loop (S3).

State lives in `sync_state`: `device_id`, `cookie` (the last pulled version), besides S1's
`enabled` and `applying`.

### Link

`Link(ctx, name)` registers the device (`POST /v1/devices`) and stores `device_id`.
- **Seeding** (`seeding: true`): in one local transaction, set `enabled = '1'` and put one outbox
  entry (`put`, all columns) for every existing row of the five entities, parents first (projects,
  categories, milestones, tasks, reports). The snapshot then uploads through the ordinary push path,
  in as many pushes as it takes; when the outbox is empty, `POST /v1/devices/seeded`.
- **Joining** (`seeding: false`): the local database must hold no user rows (the built-in Work and
  Life categories do not count); otherwise `ErrLocalData` and nothing changes (S3 decides how to
  merge). Set `enabled = '1'`, then pull everything.
- A workspace another device is seeding answers 409; `Link` returns `ErrSeeding` (try again later).
- Then lease an id block of 1000 for each entity (`POST /v1/ids`) into `sync_id_blocks`.

### Sync

`Sync(ctx)` pushes, then pulls, then refills id blocks.
- **Push**: read the outbox oldest first, at most 1000 entries a push; a `put` sends the row's
  current values of the listed columns (all synced columns when `columns` is NULL); a `put` whose
  row is gone sends nothing (its `delete` follows); the mutation id is the outbox `seq`. After a
  200, `DeleteOutbox` through the last seq sent. Repeat until the outbox is empty.
- **Pull**: from `cookie`, page until `more` is false. Each page is one `ApplyChanges` write that
  also stores the new cookie. A change to a row that still has outbox entries is skipped (the local
  change wins until it is pushed, then the server's merge comes back). A put upserts only the
  columns in `data` and leaves device-local columns alone; a delete deletes the local row. Foreign
  keys are deferred inside the write (`PRAGMA defer_foreign_keys = ON`) so a page may hold a child
  before its parent. A 409 for the cookie resets it to 0 and pulls again.
- **Refill**: an entity with fewer than 200 ids left leases another 1000.
- JSON numbers are decoded with `UseNumber` and written as integers.

### Tests (acceptance; need `ADHD_SYNC_TEST_DATABASE_URL`, skipped otherwise)

Each test runs the S2a server in `httptest` and two or three `domain.Core`s on their own temporary
databases (`testutil.OpenDB`). "Equal" means every synced column of every row of the five tables
matches, compared with plain SQL.

1. **Seed and join**: device A with existing data (projects with sub-projects, categories,
   milestones, 1500 tasks with subtasks, reports) links and syncs until done; device B (empty)
   links; both are equal, and A's `repo_path`, `cwd`, `reminded_at` are unchanged.
2. **Convergence**: a seeded random sequence of 300 operations (create, update, complete, delete
   and restore tasks; add reports; create, rename, archive and delete projects; create, update and
   delete categories) spread over A, B and C, with `Sync` calls of random devices in between; after
   two rounds of `Sync` on all three they are equal. Run five seeds; print the seed on failure.
3. **Stable numbers**: a task made on B while it cannot reach the server gets an id from B's block
   and shows on A under the same id after both sync.
4. **Columns**: A changes a task's title and B its notes; after syncing both keep both. A and B
   both change the title; both end with the one the server took last.
5. **Deletes**: deleting a project on A removes its tasks, reports and milestones on B.
6. **Join refused**: linking B with its own tasks to A's workspace is `ErrLocalData` and leaves B
   unlinked and unchanged.
7. **Seeding wait**: while A is seeding, linking B is `ErrSeeding`; after A finishes, B links.

### Review fixes (S2a/S2b, 2026-10-20)

1. **Seeding is taken over only on purpose.** `POST /v1/devices {name, resume_seeding}`: while
   another device seeds, a registration without `resume_seeding: true` is 409 `workspace is
   seeding`; with it, it takes the seeding over. The client sends `resume_seeding: true` only when
   its local database has user rows (it would seed); an empty joining device gets `ErrSeeding`.
2. **The server keeps references whole**, as the local foreign keys and `internal/domain` do.
   `internal/syncproto` lists every reference and what a delete of its target does:

   | Column | Target | On delete of the target |
   |---|---|---|
   | `milestones.project_id` | projects | delete the row |
   | `tasks.project_id` | projects | delete the row |
   | `reports.task_id` | tasks | delete the row |
   | `tasks.milestone_id` | milestones | set NULL |
   | `tasks.parent_id` | tasks | set NULL |
   | `tasks.category_id` | categories | set NULL |
   | `projects.parent_id` | projects | set NULL |

   In the push transaction: a delete also deletes or nulls every live row that references the
   deleted one (recursively, each change taking a version); a put whose reference points at a
   deleted row nulls it, or becomes a delete for a "delete the row" reference. Clients then never
   receive an orphan, and the client's `repairReferences` goes.
3. **A re-insert revives a deleted row.** A mutation carries `full: true` when its outbox entry
   has `columns` NULL (an insert: every column). A full put on a tombstone brings the row back with
   exactly that data; a partial put on a tombstone is still ignored (delete wins over an edit). So
   Undo of a delete (`RestoreTask` re-inserts with the old id) survives a sync.
4. **One pull, one write.** All pages of one `Sync` pull are applied in a single `ApplyChanges`
   write with deferred foreign keys, so a child whose parent comes on a later page is fine.

New tests: an empty device linking while another seeds is `ErrSeeding` and does not take over;
a device with data and `resume_seeding` takes over; deleting a project on A while B adds a task to
it converges with the task gone everywhere (server and both devices); deleting a task and
restoring it (Undo) before and after a sync ends with the task on every device; deleting a category
on A while B files a task under it ends with `category_id` NULL everywhere; the convergence test
also compares each device with the server's rows.
