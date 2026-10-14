# M2: Desktop shell

Goal: a menu bar app that shows what every agent is doing and lets the user manage
tasks by hand. A global hotkey opens a Spotlight-style panel; a main window holds the
project kanban. Reminders fire as system notifications. The input box creates plain
tasks in M2; the built-in agent replaces it in M3.

M2 has two parts, built in order:

- **M2a** `adhd-core serve`: a local HTTP API over the existing domain services.
- **M2b** `apps/desktop`: the Tauri app that runs `adhd-core serve` as a sidecar.

## Stack

| Concern | Choice |
|---|---|
| Shell | Tauri 2.12 (not 3.x alpha), Rust stable |
| Panel | `tauri-nspanel` 2.1 (non-activating NSPanel, needs `macos-private-api`) |
| Plugins | `tauri-plugin-shell` (sidecar), `tauri-plugin-global-shortcut`, `tauri-plugin-notification`, `tauri-plugin-opener` |
| Frontend | React 19, Vite 8, TypeScript strict, Tailwind CSS 4 |
| Data fetching | TanStack Query 5, invalidated by server-sent events |
| Kanban drag | `@dnd-kit/core` 6 |
| Frontend tests | Vitest + Testing Library |

`macos-private-api` rules out the Mac App Store. Distribution is direct download.

## M2a: `adhd-core serve`

### Process contract

- `adhd-core serve` listens on `127.0.0.1` with port 0 (random).
- On ready it prints one JSON line to stdout: `{"port":<n>,"token":"<random hex>"}`.
  Nothing else goes to stdout; logs go to stderr.
- Every request needs `Authorization: Bearer <token>`, except `OPTIONS` preflight.
  Wrong or missing token: `401`.
- `ADHD_CORE_TOKEN` and `ADHD_CORE_PORT` env vars override the random values
  (browser dev mode and tests use them).
- CORS: allow `http://localhost:1420` (Vite dev), `tauri://localhost` and
  `http://tauri.localhost`. Nothing else.
- The process exits when stdin closes, so it dies with the Tauri app.

### Endpoints

Responses use `{ "data": ... }` on success and `{ "error": "<message naming the field>" }`
with status 400/404 on failure. Request bodies are validated with the zod schemas
the MCP tools already use.

| Method | Path | Body / query | Returns |
|---|---|---|---|
| GET | `/overview` | `stale_hours` | Same shape as the `get_overview` tool. |
| GET | `/projects` | | Projects that are not archived, with counts per status. |
| GET | `/tasks` | `project_id` (or `inbox=1`), `status` (repeatable), `milestone_id` | Task list. |
| GET | `/tasks/:id` | | Task plus report timeline. |
| POST | `/tasks` | `create_task` fields; `source` is always `user` | Created task. |
| PATCH | `/tasks/:id` | `update_task` fields | Updated task. |
| GET | `/projects/:id/milestones` | | Milestones. |
| POST | `/milestones` | `create_milestone` fields | Created milestone. |
| GET | `/events` | token in `?token=` (EventSource cannot set headers) | Server-sent events, below. |

### Events

- `changed`: data changed. The server checks `PRAGMA data_version` every 500 ms, which
  catches writes from MCP processes. It also fires after its own writes.
- `reminder`: `{task_id, title, project_name}` for each task with
  `remind_at <= now`, `reminded_at IS NULL` and status not `done`/`cancelled`.
  The server sets `reminded_at` when it sends the event.
- A comment line every 15 s keeps the connection open.

### Migration 2

```sql
ALTER TABLE tasks ADD COLUMN reminded_at TEXT;
```

Setting `remind_at` through any path clears `reminded_at`.

### M2a steps (test-first, `bun test` + `bunx tsc --noEmit` after each)

1. Migration 2 and the `remind_at` / `reminded_at` rule in `tasks.ts`.
2. `src/http/` with `Bun.serve`: token, CORS, router, error mapping. One file per route group.
3. Endpoints, tested with `fetch` against a server on a temp DB.
4. `/events`: `changed` after a write from a second connection; `reminder` fires once.
5. `serve` in `cli.ts`: stdout line contract, exit on stdin close (spawn test).

## M2b: `apps/desktop`

### Rust side (`src-tauri`)

Keep it thin. All business logic stays in the TS core.

- On start: spawn sidecar `adhd-core serve`, read the first stdout line, keep
  `{port, token}` in state. Command `core_endpoint` returns it to the frontend.
  If the sidecar exits, restart it once, then show an error in the UI.
- Activation policy `Accessory`: no Dock icon.
- Tray icon. Left click toggles the panel. Menu: Open main window, Settings, Quit.
- Global shortcut, default `Alt+Space`, read from `settings.json` in the app config dir.
  Toggles the panel.
- Panel window `panel`: 640 x 480, non-activating NSPanel via `tauri-nspanel`, floats
  over full-screen apps, hides when it loses focus or on `Esc`.
- Main window `main`: 1100 x 720, normal window, hidden at start.
- Settings window or view: change the global shortcut only.

Sidecar binary: `scripts/build-sidecar.ts` runs `bun build --compile` for core and
writes `src-tauri/binaries/adhd-core-<rustc --print host-tuple>`.

### Frontend

One Vite app, two routes: `#/panel` and `#/main`.

`src/core/`: API client (fetch with token), `useCoreEvents()` (EventSource,
invalidates TanStack Query on `changed`, sends a notification on `reminder`).
Endpoint discovery: Tauri `invoke('core_endpoint')`, or `VITE_CORE_URL` +
`VITE_CORE_TOKEN` when running in a plain browser.

**Panel** (keep it minimal):
- Input box at the top. Enter creates a task in the inbox with the typed title.
  Placeholder says the AI arrives in M3.
- Section "Agents": tasks in doing / blocked / review grouped by project. Each row shows
  status, title, branch, last report summary and age, and a stale badge.
- Section "Mine": `source = user` tasks in todo / doing, sorted by remind/due time.
- Clicking a task opens the main window on that task.

**Main window**:
- Sidebar: Inbox and projects.
- Kanban: columns backlog, todo, doing, blocked, review, done. Dragging a card changes
  its status (`PATCH`). Filter by milestone. Cards show priority and PR link.
- Task drawer: edit title, notes, priority, milestone, due, remind; open PR in the
  browser; report timeline, newest first.

Style: neutral, dense, system font, light and dark mode follow the OS.

### M2b steps

1. Scaffold Tauri 2 + React + Vite + Tailwind in `apps/desktop`; sidecar build script;
   Rust spawns the sidecar and exposes `core_endpoint`. Verify: `bun run tauri build --debug` succeeds.
2. API client and `useCoreEvents`, tested with Vitest against a real `adhd-core serve`.
3. Panel view. Component tests for grouping, stale badge, input submit.
4. Main window: sidebar, kanban with drag, task drawer. Component tests for column
   grouping and the status change on drop (logic extracted into pure functions).
5. Tray, global shortcut, NSPanel, notifications, settings.
6. Manual check list in the final report: hotkey toggles the panel over a full-screen
   app; MCP `report_progress` from a Claude Code session appears in the panel within
   1 s; a reminder set 1 minute ahead shows a notification.

## Done when

- M2a: `bun test --coverage` at least 80% lines in `packages/core`; `tsc` passes.
- M2b: Vitest passes with at least 80% lines on `src/core` and the pure view logic;
  `tsc` passes; `cargo clippy` has no warnings; `bun run tauri build --debug` produces the app.
- The UI renders in a plain browser against `adhd-core serve`, so it can be checked
  without the Tauri shell.
- The manual check list passes on the user's Mac.

## Out of scope for M2

Built-in agent and provider settings (M3), gh polling (M4), daily briefing (M5),
autostart, auto-update, code signing and notarization, Windows and Linux builds.
