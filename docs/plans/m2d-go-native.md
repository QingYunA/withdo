# M2d: all Go — Go core + mygo native UI

Decision (user, 2026-10-14): build everything with mygo v0.4.8. The TypeScript core
(`packages/core`) and the React frontend (`apps/desktop/src`) are replaced by Go. The UI is
redesigned per `docs/design/ui-spec.md`.

Why: one language, one process for the app, no sidecar (58 MB, ~34 MB idle RAM), no HTTP /
token / CORS layer, no webview.

## Target layout

One Go module at the repository root: `module github.com/QingYunA/adhd-todo`, `go 1.27.1`.

```
go.mod
cmd/adhd-core/        # CLI: `mcp` (stdio MCP server) and `overview`. Same name and flags as today,
                      # so the existing `claude mcp add ... ~/.local/bin/adhd-core mcp` keeps working.
internal/paths/       # data dir per OS, ADHD_TODO_DB override
internal/store/       # open (WAL, busy_timeout 5000, foreign_keys), migrations via user_version
internal/gitctx/      # cwd -> repo slug, main repo path (git-common-dir), branch
internal/domain/      # projects, milestones, tasks, reports, overview, reminders
internal/mcpserver/   # 8 tools + instructions, official Go MCP SDK
internal/watch/       # data_version poller: change notifications and due reminders for the app
app/                  # the mygo native UI app (main.go, mygo.config.*, views, theme, glyphs)
docs/
```

If mygo's CLI needs the app in its own module or directory layout, follow mygo and record the
deviation; the shared packages stay importable by both binaries.

## Phase A: Go core (fresh, idiomatic)

User decision: no parity port. Write the core fresh in idiomatic Go, using mygo where it
applies. The TS code is background, not a spec to mirror test by test.

- Keep the SQLite schema and `user_version` numbering, so the existing database
  (`~/Library/Application Support/adhd-todo/adhd-todo.db`, `user_version = 2`) still opens.
- Keep the 8 MCP tool names, their input fields and the server `instructions`
  (agents depend on them). Official SDK `github.com/modelcontextprotocol/go-sdk`.
- Keep `cmd/adhd-core` with `mcp` and `overview`, and the worktree project resolution
  (`git rev-parse --git-common-dir`).
- `internal/watch`: `PRAGMA data_version` poller (500 ms) for change events, and reminders
  claimed once inside `BEGIN IMMEDIATE`.
- Pure Go SQLite, no cgo. No HTTP server.
- `go test -race` passes with at least 80% coverage per package. Behavior may be simplified
  where the TS version was awkward; list every change.

## Phase B: native UI (in parallel with Phase A)

The UI owns a small interface for the data it needs and is built and reviewed against an
in-memory fake with realistic seed data. Wiring to `internal/domain` happens after both phases.

- Follow `docs/design/ui-spec.md` exactly: tokens, glyphs, panel, main window (sidebar, board
  with drag and drop, inspector), settings sheet, empty states, light and dark.
- Shell behavior from the mygo web spike stays: Dock icon visible, no global shortcut, tray left
  click toggles the panel flyout, right click shows Open main window / Settings / Quit, main
  window close hides it, `quitting` flag so Quit works, notifications sent from Go with click
  opening the task.
- Render every surface to PNG in light and dark with mygo's view testing (`ui.Render` /
  `tt.Image()`) into `docs/design/screens/`, at 2x. These images are the design review.
- View tests for interaction logic (filtering, keyboard selection, drop changes status,
  inspector edits call the store).

## Phase C: integration and cleanup

- Implement the UI's data interface on `internal/domain` + `internal/watch`.
- Delete `packages/core`, `apps/desktop` (TS/React), Bun workspace files that are no longer used.
- README: build, run, register the MCP server, data location.
- Build the app with `mygo build`; install the new `adhd-core` to `~/.local/bin` (atomic
  replace); verify the real Claude Code registration still connects (`claude mcp get adhd-todo`).
- Manual checklist for the user.

## Out of scope

The built-in agent (M3), gh polling (M4), daily briefing (M5), signing and notarization,
Windows and Linux builds.

## Phase B notes (native UI)

- Layout: the app is `app/` (package main, shell, tray, notifications) with three subpackages:
  `app/data` (the UI's `Store` interface, domain mirror types, in-memory fake), `app/theme` (tokens,
  type scale, vector glyphs) and `app/views` (panel, main window, tests). `app/mygo.json` is the
  mygo config; the root `go.mod` carries `mygo` as a tool, so `cd app && go tool mygo build`
  produces `app/build/darwin-arm64/ADHD Todo.app`. No separate module was needed.
- mygo v0.4.8 is the newest release, but the docs on the main branch are ahead of it: `Element.OnEnter`,
  `Start`/`End` logical edges and `Timeline` are not in v0.4.8. Check `$GOMODCACHE/github.com/egoist/mygo@v0.4.8`,
  not the docs, before using an API.
- Design review images are regenerated with `SCREENS_OUT=$PWD/docs/design/screens go test ./app/...`.
