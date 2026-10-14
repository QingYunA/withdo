# adhd-todo desktop

Menu bar app for macOS, built with [mygo](https://github.com/egoist/mygo) (Go shell, system
webview) and the React frontend in `src/`. It has a Dock icon and a tray icon. Left-clicking the
tray icon opens a quick panel under it: quick capture, what your agents are doing, and your own
tasks. Esc or a click elsewhere closes it. The main window holds the project kanban and a task
drawer. Reminders arrive as system notifications. There is no global shortcut.

It runs `adhd-core serve` as a sidecar and talks to it over the local HTTP API and SSE stream
described in `docs/plans/m2-desktop-shell.md`. The Go side only starts the sidecar, owns the
windows and the tray, and turns reminders into notifications; all data logic lives in
`packages/core`. This is the spike described in `docs/plans/m2c-mygo-spike.md`.

## Requirements

- macOS 12+, [Bun](https://bun.sh) 1.3+, Go 1.27+ (the `go` command fetches the right toolchain),
  Xcode Command Line Tools.
- `bun install` at the repository root.

## Develop

```sh
cd apps/desktop
bun run dev          # mygo dev: compiles the sidecar, starts Vite on :1420, opens the app
```

`mygo dev` rebuilds and restarts the app when you edit a `.go` file. The dev app is a real bundle
named `ADHD Todo Dev` with the identifier `dev.adhdtodo.desktop.dev`, so notifications and
permissions are separate from the built app. `bun run generate` rewrites `src/mygo.ts`, the typed
client of the bound Go services (`Core`, `Shell`) and events; `dev` and `build` do it for you.

`bun run sidecar` rebuilds the sidecar on its own. It compiles `packages/core/src/cli.ts` to
`resources/<os>-<arch>/bin/adhd-core`, which mygo copies to `Contents/Resources/bin/adhd-core`.

Vite must stay on port 1420: the core's CORS allow-list contains `http://localhost:1420`.

## Build

```sh
cd apps/desktop
bun run build        # mygo build: sidecar, Vite, Go, then the .app and a .dmg
```

The output is `build/darwin-arm64/ADHD Todo.app` and `ADHD Todo <version> arm64.dmg`. The app is
ad hoc signed, not notarized; macOS may ask you to allow it the first time (right-click, Open).
To run the built app against a throwaway database, start its binary from a shell:

```sh
ADHD_TODO_DB=/tmp/adhd-mygo.db "build/darwin-arm64/ADHD Todo.app/Contents/MacOS/ADHD Todo"
```

## Run the UI in a plain browser

The same UI runs in any browser against a dev instance of the core, which is the quickest way to
review it without the mygo shell.

```sh
# 1. The core, with a known port and token. `tail -f /dev/null` keeps stdin open, because
#    `serve` exits when stdin closes.
rm -f /tmp/adhd-m2b.db*
tail -f /dev/null | ADHD_TODO_DB=/tmp/adhd-m2b.db ADHD_CORE_PORT=4317 ADHD_CORE_TOKEN=devtoken \
  bun packages/core/src/cli.ts serve

# 2. Sample projects, tasks and reports (optional; run once, from the repository root).
ADHD_TODO_DB=/tmp/adhd-m2b.db bun apps/desktop/scripts/seed-dev.ts

# 3. Vite on port 1420.
cd apps/desktop
VITE_CORE_URL=http://127.0.0.1:4317 VITE_CORE_TOKEN=devtoken bun run dev:web
```

Then open `http://localhost:1420/#/panel` (the quick panel) and `http://localhost:1420/#/main` (the
kanban). In a browser, clicking a task in the panel navigates to it in the main view; Esc does
nothing, and reminders do not produce notifications (the Go shell sends them). Stop everything with
Ctrl-C (and `pkill -f "cli.ts serve"` for the core).

Routes: `#/panel`, `#/main`, `#/main/task/<id>` (opens that task).

## Test

```sh
cd apps/desktop
go vet ./... && go test ./...  # sidecar manager (fake sidecar) and reminder stream
bunx vitest run --coverage     # coverage gate: >= 80% lines on src/core and src/view
bunx tsc --noEmit
```

The API client and event stream are tested against a real `adhd-core serve` started from source
(`src/test/core-server.ts`), so `bun` must be on the PATH.

## Where things live

| Path | What |
|---|---|
| `main.go` | wiring: bindings, lifecycle, sidecar start in `WhenReady`, kill in `OnQuit` |
| `sidecar.go`, `sidecar_proc.go` | runs `adhd-core serve`, parses its ready line, restarts it once, kills it on quit |
| `core.go` | the `Core.Conn` binding the page calls to learn the port and token |
| `shell.go`, `panel.go`, `tray.go` | main window (close hides it), tray-anchored flyout panel, tray menu |
| `reminders.go`, `notify.go` | Go subscribes to `/events` and shows the notifications; a click opens the task |
| `src/core/` | typed API client, endpoint discovery, SSE hook, query hooks |
| `src/view/` | pure logic: grouping, sorting, stale badge, kanban columns and drops, form patches |
| `src/panel/`, `src/main/` | the two windows |
| `src/platform/shell.ts` | everything that only exists inside mygo (close panel, open main, links) |
| `src/mygo.ts` | generated client of the Go bindings; do not edit |

The database is the core's usual one (`ADHD_TODO_DB` overrides it when the app is launched from a
terminal).

Notes:

- If the sidecar dies it is restarted once, on the same port and token, so the page reconnects by
  itself. A second failure shows an error with a retry button.
- Reminders come from Go, so they fire with the main window closed or hidden. Notifications need a
  packaged app (`mygo dev` and `mygo build` both are); the first one asks for permission.
- The quick panel is created on each tray click, so it always shows fresh data and a focused input.
- The panel closes on Esc because the page does not call `preventDefault()` on it. Keep it that
  way: an Escape handler that must not close the panel has to call `e.preventDefault()`.

## Manual checks (cannot be automated here)

1. The Dock icon is visible. Close the main window with the red button, click the Dock icon: it
   comes back.
2. Left-click the tray icon: the panel opens under it with the input focused. Click again, press
   Esc, or click elsewhere: it closes.
3. With the panel open, type a title (try Cmd+V too) and press Enter: a task appears under Mine and
   in the Inbox.
4. From a Claude Code session with the MCP server registered, call `report_progress`: the task
   shows in the panel's Agents section and on the board within about a second.
5. Drag a card to another column. Set a reminder one minute ahead on a task, close the main window
   and wait: a system notification appears (allow notifications when asked); clicking it opens the
   task.
6. Right-click the tray icon: the menu offers Open main window and Quit.
7. Quit from the tray menu or with Cmd+Q: no `adhd-core serve` process remains
   (`pgrep -fl "adhd-core serve"`).
