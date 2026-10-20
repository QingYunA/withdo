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

## A2b-core: what the phone's sync screens need, below the UI

Runs beside A2a, so it adds files and touches no screen: the Go API the Sync settings and the
reminders need, and the two Android pieces the core's interfaces ask for. The screens that use them
are A2b-UI, after A2a.

### Go (`mobile/core`, new file `account.go`; tests in new `*_test.go` files)

- **Sign-in with a choice**: `SignIn(opener, deviceName) (string, error)` now returns JSON
  `{"state":"linked"}` or, when the phone has tasks and the account has tasks too
  (`syncclient.ErrLocalData`), `{"state":"choose"}`, keeping the tokens so the user does not sign in
  again. Then `LinkMerge(deviceName string) error` (`Client.LinkMerge`) or
  `LinkReplace(deviceName string) (string, error)` (`Client.LinkReplace`, returning the backup's
  path) links the phone. `SignIn` when already linked returns `{"state":"linked"}`.
- `Account() (string, error)`: `{"name":..,"email":..}` from `sync.json`, `{}` when signed out.
- `Devices() (string, error)`: the server's device list as JSON (`syncproto.Device` fields), plus
  `"this":true` on this phone's device (`sync_state` `device_id`).
- `ClaimDueReminders() (string, error)`: `domain.ClaimDueReminders(now)` as a JSON list
  `[{"task_id":..,"title":..,"project":..}]`; `[]` when none. Each reminder comes back once.
- `NextReminderAt() (int64, error)`: the earliest `remind_at` of an open task not reminded yet, as
  Unix milliseconds, `0` when there is none (the phone sets an alarm for it).
- Golden files for each JSON shape in `mobile/core/testdata/`; the bindable test covers the new
  functions.

### Kotlin (`android/app/src/main/java/xyz/sriai/adhdtodo/platform/`, new files only)

- `KeystoreSecrets(context) : Secrets`: an AES-256-GCM key in `AndroidKeyStore` (alias
  `adhd-todo-secrets`), each value encrypted with a fresh IV and kept as base64 in a private
  `SharedPreferences` file. `get` of a missing key throws an exception whose message is `not found`.
  The storage logic sits behind a small `Cipher`-like interface, so a JVM unit test checks set, get,
  overwrite, delete and the `not found` message with a fake cipher.
- `CustomTabOpener(context) : BrowserOpener`: opens the URL in a Custom Tab
  (`androidx.browser`, already a dependency), with `FLAG_ACTIVITY_NEW_TASK`; when no browser handles
  Custom Tabs, an `ACTION_VIEW` intent.

### Verification

1. Go: `go test -race ./mobile/...` with and without `ADHD_SYNC_TEST_DATABASE_URL`; on Postgres, a
   phone with tasks signing in to an account with tasks gets `choose`, then `LinkMerge` keeps both
   sides' tasks; `LinkReplace` leaves the account's; `Devices` marks this phone; a due reminder is
   claimed once; `NextReminderAt` is the earliest.
2. `scripts/android-core.sh` rebuilds the `.aar`; `./gradlew --offline assembleDebug
   testDebugUnitTest` passes with the new unit test.
3. `scripts/check.sh full` passes.

## A2b-UI: projects, a task, sync and reminders

The second half of the Compose app, on A2a's shell and A2b-core's API. Mockup: v26 frames 3 and 4;
the Sync screen has no phone mockup and follows the rules below (v27's groups in one column).

### Screens

- **Projects tab**: the project list (folder icon in the project's color, name, open count), then a
  project page per v26 frame 3: back bar `Projects`, the title with its folder, the segmented
  control `Mine 5 | Agents 3`, groups by status with the glyph in the header, rows 26 dp in with the
  author's avatar, Done folded into one `Completed 12` row that unfolds. The composer adds to the
  project (`CreateTask` with its project id).
- **A task** (tap any row on Mine, Agents or a project): full screen per v26 frame 4: back bar with
  the task's `#N` at the right in `mono` `text.tertiary`, the title at 22 sp 700, property rows (18 dp
  icons, 104 dp label column): status, from, priority strip P0–P3 (tapping sets it, `UpdateTask`),
  project, branch, PR (tapping opens it); Subtasks with status pills; Activity on its rail, the
  user's replies as `You`. The composer at the bottom says `Reply to <agent>` for an agent's task
  (`Reply`) and `Add a note` for the user's own (`UpdateTask` notes append is out of scope: reply too).
- **Sync** (tap the header's cloud on any tab): back bar `Sync`, then by state, the same states and
  words as the desktop's pane (`app/locales` keys `settingsSync*` are the source for the strings;
  copy them into `strings.xml`, en and zh):
  - signed out: a card with the 36 dp pixel cloud, the title, the body, the accent button
    `Sign in and sync` (`SignIn` through `CustomTabOpener`, device name `Build.MODEL`), the note;
  - signing in: the button disabled with `Waiting for the browser…`; when the app comes back to the
    front without the sign-in finished, the button is enabled again;
  - `choose`: a modal bottom sheet with the two radio cards of v27 (Merge recommended and picked),
    `Cancel` and `Continue` (`LinkMerge` or `LinkReplace`);
  - signed in: the account group (name, `Signed in · 3 devices`, `Sign out` as a text button), the
    This phone row (status line in `status.done` / `status.blocked` / `text.secondary`, `Sync now`);
    `Devices` section: laptop or phone pixelarticon by name (a name with Mac/Book/PC is a laptop),
    the name, a `This phone` tag, the age at the right; the first upload's 40-cell pixel bar while
    seeding; a failure's reason, retry time and `Retry`;
  - the footer `Server todo.sri-ai.xyz · Change`, Change edits the address in place.
  The header cloud turns `status.blocked` when the sync status is failed for 15 minutes or signed
  out after having been linked, as on the desktop.
- **Server settings**: the phone's default server is `https://todo.sri-ai.xyz`. Changing it calls a
  new `SyncSetServer(server string) error` in `mobile/core` that fetches `GET /v1/config`
  (`syncclient.FetchConfig`, https only) and writes `sync.json` with the issuer, client id and
  scopes; signing in calls it first when `sync.json` has no issuer.
- **Sync running**: `StartSync` when the app comes to the front and the phone is linked, `StopSync`
  when it goes to the back; `SyncNow` on `Sync now` / `Retry`.

### Reminders

- `NotificationChannel` `reminders` (`Reminders` / `提醒`). A `BroadcastReceiver` claims due
  reminders (`ClaimDueReminders`) and posts one notification each: the title, the project (or
  Inbox) as the text, the app's pixel mark as the small icon (a monochrome vector of the Mine
  sprite's outline), tapping opens the task.
- After any change (`OnChange`), at start and on `BOOT_COMPLETED`, the app sets one alarm at
  `NextReminderAt` with `AlarmManager.setAndAllowWhileIdle(RTC_WAKEUP, …)` (inexact; no exact-alarm
  permission). `POST_NOTIFICATIONS` is asked the first time an alarm is set on Android 13+.

### Verification

1. `./gradlew --offline assembleDebug testDebugUnitTest`: unit tests for the device-icon rule, the
   header cloud's red rule, and decoding of every new golden JSON.
2. Go: `SyncSetServer` test against an `httptest` server's `/v1/config`; `go test ./mobile/...`.
3. `scripts/check.sh full` passes.
4. The reviewer checks on the emulator with `demo=true`: Projects, a project, a task (light, dark,
   zh) against v26 frames 3–4; each Sync state (a debug intent extra `sync_state=<name>` may fake
   the state for screenshots); a reminder set a minute ahead posts a notification.

## A3-fixes: what the first end-to-end run found

The first run against the real server (a desktop-side client on `internal/cli`, the emulator signing
in to a test account) found these. Each needs a test where one is possible.

1. **The sign-in error is lost.** `App.kt` catches `signIn`'s exception and shows the generic
   `Couldn't sign in`. Show the Go error's text (they are short sentences fit for the user) under
   the button, and log the exception with `Log.w("adhd-todo", ...)`.
2. **`syncError`'s catch-all hides the cause.** In `mobile/core/sync.go`, log the raw error with
   `log.Printf` (gomobile sends it to logcat as GoLog) before mapping it, and map network errors
   (DNS, dial, connection reset or abort, timeout: `net.Error`, `*net.DNSError`, `syscall` errors)
   to `could not reach the server; check the connection`. Never log tokens: errors from `net/http`
   carry the URL, which holds none here; do not log request bodies.
3. **Transient network errors end the sign-in.** Retry the OIDC code exchange and each request of
   `Link` (device create, id lease, seed or join) up to 3 times, 1 s then 2 s apart, only for errors
   where the request may not have reached the server (DNS, dial, connection refused/reset/abort).
   Put the rule in one helper in `internal/syncclient` with a unit test over fake errors.
4. **The runner races the link.** The logs show the runner refreshing the token while `Link` was
   still leasing ids. `mobile/core`: `StartSync` does nothing while a `SignIn`/`LinkMerge`/
   `LinkReplace` is running (a mutex-guarded flag), and the app starts sync only after `SignIn`
   answered `linked` (or after a merge or replace finished). Test: `StartSync` during a slow fake
   link starts no runner.
5. **The browser stays in front after signing in** with "You can close this tab". When `SignIn`
   returns (either answer, or an error), bring the app back: start `MainActivity` with
   `FLAG_ACTIVITY_NEW_TASK | FLAG_ACTIVITY_REORDER_TO_FRONT` (it is `singleTop`), which comes back
   over the Custom Tab.
6. **The choose sheet has no counts.** `SignIn`'s `choose` answer carries `"local_tasks"`,
   `"local_projects"` (`domain.Core.SyncCounts`) and `"account_devices"` (devices that have synced,
   as `conflictAfterLink` in `app/data/sync_link.go` counts); update the golden. The sheet uses the
   desktop's sentence (the `settingsSyncBoth*` keys in `app/locales`, copied to `strings.xml` en and
   zh) with the counts.
7. **Debug `route=projects` lands on Mine.** Fix the route so the reviewer can open the Projects tab.
8. **Desktop: one refresh token for every database.** `syncclient.RefreshTokenKey` is the fixed
   keychain key `sync/refresh-token`, so a second data directory (a test profile, the CLI with
   `ADHD_TODO_DB`) signing in replaces the app's token. Key it per database:
   `sync/refresh-token/<first 16 hex of sha256 of the absolute database path>`. On read, when the
   new key is missing and the old one exists, move the old one to the new key (the app's existing
   sign-in keeps working). Tests in `internal/syncclient` and `app/data` with a fake secrets store.

### Verification

1. `go test -race ./internal/syncclient/... ./mobile/... ./app/data/...` and the new tests above.
2. `scripts/android-core.sh`; `./gradlew --offline assembleDebug testDebugUnitTest`.
3. `scripts/check.sh full`.
