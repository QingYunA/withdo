# M1: Core data layer + MCP server

Goal: coding agents (Claude Code first) can report what they are doing into a local
SQLite database, and the user can see every active project, task and PR from one
command. No UI, no built-in agent, no gh polling, no Claude Code hooks yet.

## Stack

| Concern | Choice |
|---|---|
| Runtime, package manager, tests, compile | Bun 1.3 (`bun test`, `bun build --compile`) |
| Language | TypeScript, `strict: true` |
| Database | `bun:sqlite`, WAL mode, `busy_timeout = 5000`, `foreign_keys = ON` |
| MCP | `@modelcontextprotocol/server` 2.x (`McpServer`, `StdioServerTransport`), `zod` 4 |
| MCP tests | `@modelcontextprotocol/client` (`Client`, `InMemoryTransport.createLinkedPair`) |

M1 does not use the AI SDK. It arrives in M3.

## Repository layout

```
adhd-todo/
  package.json              # bun workspaces: packages/*, apps/*
  tsconfig.base.json
  packages/core/
    src/
      cli.ts                # entry: mcp | overview
      paths.ts              # data dir per OS, ADHD_TODO_DB override
      db/
        open.ts             # open + pragmas
        migrations.ts       # ordered SQL list, PRAGMA user_version
      git.ts                # cwd -> { repoSlug, toplevel, branch }
      domain/
        projects.ts
        milestones.ts
        tasks.ts
        reports.ts
        overview.ts
      mcp/
        server.ts           # createServer(db) + instructions
        tools/*.ts          # one file per tool
    test/
  apps/desktop/             # M2 (Tauri), empty in M1
```

## Data location

- macOS: `~/Library/Application Support/adhd-todo/adhd-todo.db`
- Linux: `$XDG_DATA_HOME/adhd-todo/adhd-todo.db` (fallback `~/.local/share`)
- Windows: `%APPDATA%\adhd-todo\adhd-todo.db`
- `ADHD_TODO_DB` overrides the path (tests use a temp file).

## Schema (migration 1)

IDs are `INTEGER PRIMARY KEY` so agents and humans can say "task 42".
Timestamps are ISO 8601 UTC strings.

```sql
CREATE TABLE projects (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  repo_slug TEXT UNIQUE,          -- "owner/repo" from git remote; NULL for non-GitHub dirs
  repo_path TEXT,                 -- last seen toplevel; key when repo_slug is NULL
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  archived_at TEXT
);

CREATE TABLE milestones (
  id INTEGER PRIMARY KEY,
  project_id INTEGER NOT NULL REFERENCES projects(id),
  title TEXT NOT NULL,
  due_at TEXT,
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','closed')),
  created_at TEXT NOT NULL
);

CREATE TABLE tasks (
  id INTEGER PRIMARY KEY,
  project_id INTEGER REFERENCES projects(id),       -- NULL = inbox
  milestone_id INTEGER REFERENCES milestones(id),
  title TEXT NOT NULL,
  notes TEXT,
  status TEXT NOT NULL DEFAULT 'todo'
    CHECK (status IN ('backlog','todo','doing','blocked','review','done','cancelled')),
  priority TEXT NOT NULL DEFAULT 'P2' CHECK (priority IN ('P0','P1','P2','P3')),
  due_at TEXT,
  remind_at TEXT,                 -- stored in M1, fired by the app in M2
  branch TEXT,
  pr_url TEXT,
  pr_number INTEGER,
  source TEXT NOT NULL CHECK (source IN ('user','agent')),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  completed_at TEXT
);

CREATE TABLE reports (
  id INTEGER PRIMARY KEY,
  task_id INTEGER NOT NULL REFERENCES tasks(id),
  agent TEXT NOT NULL,
  status TEXT,                    -- task status set by this report, if any
  summary TEXT NOT NULL,
  next_step TEXT,
  created_at TEXT NOT NULL
);

CREATE INDEX tasks_project_branch ON tasks(project_id, branch);
CREATE INDEX reports_task_created ON reports(task_id, created_at);
```

Rules:
- Setting `status` to `done` or `cancelled` sets `completed_at`; moving back clears it.
- `pr_url` is parsed into `pr_number` when it is a GitHub PR URL.

## Project resolution (`git.ts` + `projects.ts`)

From a `cwd`:
1. `git rev-parse --show-toplevel` and `git rev-parse --abbrev-ref HEAD`.
2. `git remote get-url origin`, parsed into `owner/repo` (ssh, https, with or without `.git`).
3. Match project by `repo_slug`; fall back to `repo_path` when there is no remote.
   Worktrees of one repo share the remote, so they map to one project.
4. Create the project when nothing matches (`name` = repo name or directory name).
5. A non-git `cwd` resolves to no project; tasks go to the inbox.

## MCP tools

Server name `adhd-todo`. The server `instructions` tell agents to call
`report_progress` when they start a task, get blocked, open a PR, and finish.

| Tool | Input (all optional unless marked) | Behavior |
|---|---|---|
| `report_progress` | `summary`*, `status`, `next_step`, `task_id`, `title`, `cwd`, `branch`, `pr_url` | Resolve project from `cwd` (default: server process cwd). Find the open task for `task_id` or (project, branch); create one if none (`title` or branch name). Apply `status`/`pr_url`, append a report, return task id + status. |
| `create_task` | `title`*, `cwd` or `project_id`, `notes`, `priority`, `milestone_id`, `due_at`, `remind_at`, `branch`, `pr_url` | Create a task. |
| `update_task` | `task_id`*, any task field | Patch a task. |
| `get_task` | `task_id`* | Task plus its report timeline. |
| `list_tasks` | `project_id` or `cwd`, `status[]`, `milestone_id`, `limit` (default 50) | Filtered list, newest first. |
| `get_overview` | `stale_hours` (default 48) | Per project: tasks in doing/blocked/review, latest report, stale flag. |
| `upsert_project` | `cwd` or `repo_slug`, `name` | Explicit project creation or rename. |
| `create_milestone` | `title`*, `cwd` or `project_id`, `due_at` | Create a milestone. |

Tool results return short text plus `structuredContent`. Validation errors return
`isError: true` with a message that names the bad field.

## CLI

| Command | Purpose |
|---|---|
| `adhd-core mcp` | Run the stdio MCP server. |
| `adhd-core overview` | Print the overview: projects, active tasks, stale flags. |

## Steps

Each step is test-first. Run `bun test` and `bunx tsc --noEmit` after every step.

1. Scaffold: `git init`, bun workspaces, strict tsconfig, `packages/core`, `.gitignore`.
   Verify: empty test run passes, typecheck passes.
2. DB: `paths.ts`, `open.ts`, `migrations.ts`.
   Verify: fresh migrate, re-run is a no-op, two connections write concurrently without `SQLITE_BUSY`.
3. Git: URL parser (table-driven), resolver against temp repos, worktree, no remote, non-git dir.
4. Domain services: projects, milestones, tasks (status/`completed_at`, `pr_number`), reports, overview (stale flag).
5. MCP server: all 8 tools through `InMemoryTransport`.
   Verify: tool list, `report_progress` creates a task then reuses it on the same branch, validation errors.
6. CLI `overview`, plus a README section with the registration command:
   `claude mcp add --scope user adhd-todo -- <path to adhd-core> mcp`.
7. Build and smoke test: `bun build --compile` to `packages/core/dist/adhd-core`; a script
   connects over stdio and lists tools. Then register it in Claude Code and dogfood.

## Done when

- `bun test --coverage` passes with at least 80% line coverage in `packages/core`.
- `bunx tsc --noEmit` passes.
- The compiled binary answers `tools/list` over stdio.
- In a real Claude Code session, a `report_progress` call appears in `adhd-core overview`
  under the right project and branch, and a task with no report for 48 hours is flagged stale.

## Out of scope for M1

Desktop UI, notifications, built-in agent, provider settings, gh polling, daily briefing,
automatic edits to `~/.claude/settings.json`.

Claude Code hooks are deferred. M1 relies on the MCP server `instructions` alone. If
dogfooding shows agents forget to report, add a `SessionStart` hook first: it fires once
per session. Avoid `Stop`: it fires at the end of every turn.
