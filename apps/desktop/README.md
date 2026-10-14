# adhd-todo desktop

Menu bar app for macOS (Tauri 2). A global shortcut (default `Option+Space`) opens a
Spotlight-style panel: quick capture, what your agents are doing, and your own tasks. The main
window holds the project kanban and a task drawer. Reminders arrive as system notifications.

It runs `adhd-core serve` as a sidecar and talks to it over the local HTTP API and SSE stream
described in `docs/plans/m2-desktop-shell.md`. The Rust side only spawns the sidecar, owns the
windows, the tray and the shortcut; all data logic lives in `packages/core`.

## Requirements

- macOS 12+, [Bun](https://bun.sh) 1.3+, Rust (stable, 1.80+), Xcode Command Line Tools.
- `bun install` at the repository root.

## Develop

```sh
cd apps/desktop
bun run tauri dev      # compiles the sidecar, starts Vite on :1420, opens the app
```

`bun run sidecar` rebuilds the sidecar on its own. It compiles `packages/core/src/cli.ts` to
`src-tauri/binaries/adhd-core-<rustc host triple>`. Cargo needs that file to exist, so run it once
before `cargo clippy` or `cargo test` on a fresh checkout.

Vite must stay on port 1420: the core's CORS allow-list contains `http://localhost:1420`.

## Build

```sh
cd apps/desktop
bun run tauri build --debug    # add no flag for a release build (slower, smaller)
```

The bundle is `<cargo target dir>/debug/bundle/macos/ADHD Todo.app` (the cargo target dir is
`src-tauri/target` unless `CARGO_TARGET_DIR` is set). The sidecar is inside it at
`Contents/MacOS/adhd-core`. The app is not signed or notarized; macOS may ask you to allow it
the first time (right-click, Open).

## Run the UI in a plain browser

The same UI runs in any browser against a dev instance of the core, which is the quickest way to
review it without the Tauri shell.

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
VITE_CORE_URL=http://127.0.0.1:4317 VITE_CORE_TOKEN=devtoken bun run dev
```

Then open `http://localhost:1420/#/panel` (the quick panel) and `http://localhost:1420/#/main` (the
kanban). In a browser, clicking a task in the panel navigates to it in the main view; Esc and the
global shortcut do nothing. Stop everything with Ctrl-C (and `pkill -f "cli.ts serve"` for the core).

Routes: `#/panel`, `#/main`, `#/main/task/<id>` (opens that task), `#/main/settings`.

## Test

```sh
cd apps/desktop
bunx vitest run --coverage     # coverage gate: >= 80% lines on src/core and src/view
bunx tsc --noEmit
cargo clippy --manifest-path src-tauri/Cargo.toml --all-targets -- -D warnings
cargo test --manifest-path src-tauri/Cargo.toml
```

The API client and event stream are tested against a real `adhd-core serve` started from source
(`src/test/core-server.ts`), so `bun` must be on the PATH.

## Where things live

| Path | What |
|---|---|
| `src/core/` | typed API client, endpoint discovery, SSE hook, query hooks, notifications |
| `src/view/` | pure logic: grouping, sorting, stale badge, kanban columns and drops, form patches |
| `src/panel/`, `src/main/` | the two windows |
| `src/platform/shell.ts` | everything that only exists inside Tauri (invoke, events, opener) |
| `src-tauri/src/sidecar.rs` | spawns `adhd-core serve`, parses its ready line, restarts it once |
| `src-tauri/src/panel.rs` | the non-activating NSPanel (`tauri-nspanel`) |
| `src-tauri/src/shortcut.rs`, `settings.rs` | global shortcut and `settings.json` |

Settings are stored in the app config dir, `~/Library/Application Support/dev.adhdtodo.desktop/settings.json`.
The database is the core's usual one (`ADHD_TODO_DB` overrides it when the app is launched from a terminal).

Notes:

- If the sidecar dies it is restarted once, on the same port and token, so the UI reconnects by
  itself. A second failure shows an error with a retry button.
- Reminder notifications are sent only by the main window's event stream. That window stays loaded
  while hidden, and sending from both windows would notify twice.
- The panel uses private macOS APIs (`macos-private-api`), so the app cannot go on the Mac App Store.

## Manual checks (cannot be automated here)

1. Press `Option+Space` from another app: the panel appears with the input focused. Press it
   again, press Esc, or click elsewhere: it hides.
2. Put another app in full screen, press `Option+Space`: the panel shows over it without leaving
   the full-screen space.
3. With the panel open, type a title and press Enter: a task appears under Mine and in the Inbox.
4. From a Claude Code session with the MCP server registered, call `report_progress`: the task
   shows in the panel's Agents section within about a second.
5. In the main window drag a card to another column; set a reminder one minute ahead on a task
   and wait: a system notification appears (allow notifications for ADHD Todo when asked).
6. Settings: change the shortcut, quit and relaunch: the new shortcut is still active.
7. Tray: left click toggles the panel; the menu offers Open main window, Settings and Quit.
   Closing the main window with the red button keeps the app running.
8. Quit from the tray: no `adhd-core` process remains (`pgrep -fl adhd-core`).
