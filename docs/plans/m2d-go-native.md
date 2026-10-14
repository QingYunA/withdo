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

## Phase A: Go core (parity port)

Port `packages/core` behavior exactly. The TS code and its 191 tests are the spec; read them.

- **Database compatibility is mandatory.** The user's real database is at
  `~/Library/Application Support/adhd-todo/adhd-todo.db`, `user_version = 2`. The Go store must
  open it unchanged: same tables, columns, constraints, indexes, migration numbering. A
  migration 3 is allowed only if needed, and must be additive.
- SQLite driver: pure Go, no cgo (`modernc.org/sqlite`, or mygo's `plugins/sqlite` if it is a
  better fit — decide by reading it, and record why).
- Git: include the worktree fix merged in PR #1 (`git rev-parse --git-common-dir` maps worktrees
  of a remote-less repo to the main repo's project).
- MCP: official SDK `github.com/modelcontextprotocol/go-sdk`. Same server name `adhd-todo`, the
  same 8 tools with the same names, input fields, defaults, validation messages (naming the bad
  field), text results plus structured content, and the same `instructions` text (including
  "Always pass cwd"). Agent name from the client's `clientInfo.name`.
- `overview` CLI prints the same text format as today.
- `internal/watch`: polls `PRAGMA data_version` every 500 ms on its own connection; emits
  "changed"; claims due reminders once (`reminded_at`), inside `BEGIN IMMEDIATE`.
- No HTTP server. `adhd-core serve` is dropped.
- Tests: port every TS test case to Go table tests (`go test -race`), including the concurrent
  two-connection write test and a temp git repo matrix. Coverage >= 80% per package
  (`go test -cover`).
- Parity check: a script runs the TS binary and the Go binary over stdio against copies of the
  same database, calls `tools/list` and a fixed sequence of tool calls, and diffs the results
  (ignoring timestamps and ids where they must differ).

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
