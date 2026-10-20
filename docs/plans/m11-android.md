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

## A2a: the app shell, Mine and Agents

The Compose app on `mobile/core`, first half: everything a user sees on opening it. Spec: **Android**
in `docs/design/ui-spec.md`; mockup `docs/design/mockups/v26.html` (frames 1 and 2, light and dark).
The skeleton in `android/` builds and loads the core (an emulator showed the Mine JSON).

### One source for the pictures

- Sprites: the app reads `app/theme/sprites/avatars/*.png` as assets (`sourceSets.main.assets`
  pointing at the repo's folder, no copies) and draws them with nearest-neighbour filtering
  (`FilterQuality.None`). Android densities are not whole numbers (420 dpi is 2.625 px a dp), so a
  12-cell sprite meant for 18 dp is drawn at the nearest whole multiple of 12 *physical pixels*
  (`round(18 * density / 12) * 12` px) and centered in its 18 dp slot, so every cell is a whole number
  of pixels. The same rule holds for glyphs and pixelarticons (a 24-unit icon is 12 cells).
- Interface icons: the pixelarticons SVGs in `app/theme/pixel-icons` as assets too, their `d`
  attributes parsed with Compose's `PathParser` and filled in the text color.
- Status glyphs: the 12 x 12 grids of `app/theme/glyph.go` ported to Kotlin as rectangles, with a unit
  test that the Kotlin grids equal the Go ones (read `glyph.go` in the test with a small parser, or
  have a Go test write `mobile/core/testdata/glyphs.json` and compare with it).
- Colors and type: the tokens of ui-spec's tables (light and dark) as a Kotlin theme object; no
  literal colors in screens.

### Screens and behavior

- `MainActivity` opens the core in `filesDir`, implements `Listener` (refresh the visible page on
  `OnChange`; keep the sync status), and shows the bottom bar: Mine, Agents, Calendar (a placeholder
  that says it comes later), Projects (a placeholder for A2b).
- **Mine**: the header (title, date, sync cloud), From your agents (the one box; buttons under the
  summary; Accept calls `Accept`, Answer opens a reply sheet that calls `Reply`, Open PR opens the
  URL), Today, Later, Done today, each section hidden when empty, the empty state per the spec.
  Tapping a status glyph of my task completes it.
- **Agents**: Working with subtasks on their rail and pills, Done today, the empty state.
- **Composer** pinned above the bottom bar: creates a plain task (`CreateTask`), the Inbox on Mine.
- Data classes for the page JSON with `kotlinx.serialization`, decoded leniently (unknown keys
  ignored).

### Verification

1. **JSON contract**: a JVM unit test decodes every `mobile/core/testdata/*.golden.json` into the data
   classes with no error and spot-checks fields.
2. **Glyphs**: the Kotlin grids equal the Go grids (above).
3. `./gradlew assembleDebug testDebugUnitTest` passes (offline from `GRADLE_USER_HOME=/tmp/adhd-gradle`).
4. Screens are checked on an emulator by the reviewer against v26 frames 1 and 2, light and dark, with
   the demo data; a `debug` build seeds the demo data when started with an intent extra `demo=true`
   (through a small `SeedDemo(dataDir)` added to `mobile/core`, reusing `app/data`'s demo seeding).
