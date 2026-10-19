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

### Out of scope for S1

Networking, linking a device, the server, images and summaries, Settings UI, Android.
