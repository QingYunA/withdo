# M11 Android

The phone app (design: `docs/design/mockups/v26.html`, spec: **Android** in `docs/design/ui-spec.md`)
is Kotlin + Jetpack Compose over the same Go core as the desktop, bound with gomobile, as Tailscale
and Anytype do. The sync protocol, the client and the server are in `docs/plans/m11-sync.md`.

| Phase | Scope | Verified by |
|---|---|---|
| A1 | `mobile/core`: the Go core's API for Kotlin, built into an `.aar` | Go tests; `scripts/android-core.sh` builds the `.aar` |
| A2 | The Compose app on that `.aar`, screen by screen from v26 | emulator screenshots against v26 |
| A3 | End to end with the desktop through the server | a task made on the emulator shows on a desktop, and back |

## A1: the core for Kotlin

`mobile/core` is the only package Kotlin sees. It wraps `app/data` (the page models the desktop
already builds, which do not import mygo), `internal/domain` and `internal/syncclient`, so the phone
shows what the desktop shows and syncs the same way. `GOOS=android go build ./app/data
./internal/syncclient ./internal/domain` already passes.

### Rules of the API (gomobile)

- Only types gomobile binds: `string`, `bool`, `int64`, `float64`, `[]byte`, `error`, pointers to
  structs of the package with those fields, and interfaces the Kotlin side implements.
- **Page data crosses as JSON strings**: the same structs `app/data` gives the desktop's pages,
  marshalled with `encoding/json`, so Kotlin decodes them with one set of data classes. Each read
  documents its JSON in a Go doc comment and a golden test file under `mobile/core/testdata/`.
- Errors are short sentences fit to show the user; no tokens, paths or SQL in them.

### API

- `Open(dataDir string, secrets Secrets) (*Core, error)`: opens `dataDir/adhd-todo.db` (migrating it)
  and starts watching it (`internal/watch`), like `data.OpenReal`. `Close()`.
- Reads, each `(string, error)` of JSON: `Mine()`, `Agents()`, `Projects()`, `Project(id int64)`,
  `Task(id int64)` (the inspector's content: properties, subtasks, activity), `Search(query string)`.
- Writes: `CreateTask(inputJSON string) (string, error)` (the composer without a model: a plain task,
  in a project or the Inbox), `UpdateTask(id int64, patchJSON string) error` (title, notes, status,
  priority, due, remind, category, project), `Complete(id int64) error`, `Accept(id int64) error`
  (the Accept of From your agents), `Reply(id int64, text string) error` (below), `DeleteTask` and
  `RestoreTask` (Undo).
- **Reply** is new in `internal/domain`: `AddReply(ctx, taskID, text)` files a report with agent
  `you` and kind `reply`; the task's activity shows it as the user's, the MCP tool `get_task` lists
  it among the reports (so the agent reads it with its next look at the task), and it syncs like any
  report. It does not change the task's status or count as the agent speaking for staleness.
- `SetListener(Listener)`: `Listener.OnChange()` after any change to the database (the watcher's
  signal, so changes from sync count), `Listener.OnStatus(statusJSON)` when the sync status changes.
- Sync, mirroring `adhd-core sync`: `SyncConfigure(server, issuer, clientID, scopes string)` (scopes
  space separated; kept in `dataDir/sync.json`), `SignIn(opener BrowserOpener, deviceName string)
  error` (the loopback PKCE sign-in of `OIDCSource`, then link; `BrowserOpener.Open(url)` is Kotlin's
  Custom Tab), `SignOut() error`, `SyncNow() error`, `StartSync()` / `StopSync()` (the Runner),
  `SyncStatus() string` (the Runner's status as JSON).
- `Secrets` is a Kotlin interface: `Get(key string) (string, error)`, `Set(key, value string) error`,
  `Delete(key string) error` (Android Keystore on the Kotlin side). A missing key is an error whose
  text is `not found`.

### Build

`scripts/android-core.sh` runs `gomobile bind -target=android/arm64,android/amd64 -androidapi 26
-javapkg xyz.sriai.adhdtodo -o android/core/core.aar ./mobile/core` (ANDROID_HOME, ANDROID_NDK_HOME
and JAVA_HOME from the environment, with the defaults of a Homebrew openjdk@17 and
`~/Library/Android/sdk`), and checks the `.aar` holds `classes.jar` and both `libgojni.so`.
`android/core/core.aar` is not committed (`.gitignore`). CI does not build it (no NDK there).

### Tests (acceptance)

1. **Same pages as the desktop**: on a database seeded with `app/data`'s demo data, `Mine()`,
   `Agents()`, `Project(id)` and `Task(id)` decode to exactly what `data.Real`'s methods return for
   the same database (compare the decoded JSON with the marshalled desktop structs), and the golden
   files pin the JSON shape.
2. **Writes**: create, update, complete, accept, delete and restore through the API change the
   database as the desktop's `data.Real` does (check through `internal/domain`), and each fires
   `OnChange` once.
3. **Reply**: `Reply` files a `reply` report by `you`; `Task(id)` shows it in activity as the user's;
   the MCP `get_task` output includes it; it neither changes status nor resets staleness.
4. **Sync** (Postgres, like the S2b tests): a desktop `domain.Core` and a `mobile/core` Core linked to
   the same workspace with a static token; a task made on the phone shows on the desktop after
   `SyncNow` on both, and a reply made on the phone reaches the desktop's activity.
5. **Bindable**: a test runs `go vet` and checks with `golang.org/x/tools/go/packages` (or a simple
   reflection over the exported API) that every exported function and method uses only bindable
   types; `scripts/android-core.sh` produces the `.aar` (run locally; it needs the NDK).
