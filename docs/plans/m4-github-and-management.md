# M4: GitHub polling and managing things by hand

Requested 2026-10-15. Two halves, built in parallel:

- **M4a GitHub polling**: pull request state comes in by itself.
- **M4b management**: everything the user could so far only get through an agent's MCP calls, or
  not at all, can be done in the app.

Decisions (user, 2026-10-15): projects can be archived or deleted; a new project takes a name and
an optional GitHub repository; a pull request closed without merging goes to Needs you.

## M4a: GitHub polling

- **Source**: the `gh` CLI the user is already signed in to. No token of our own. The app is
  started from Finder with a short PATH, so look for `gh` in PATH, `/opt/homebrew/bin` and
  `/usr/local/bin`. Without `gh`, or signed out, polling is off and Settings says how to fix it
  (`brew install gh`, `gh auth login`).
- **What is polled**: tasks that are not done or cancelled and have a `pr_url`. Per pull request:
  `gh pr view <url> --json state,mergedAt,closedAt,isDraft,reviewDecision,statusCheckRollup`, at most
  4 at a time, each with a timeout. At launch, then every 5 minutes while the app runs, and when
  the window comes back to the front if the last poll is older than 1 minute.
- **Stored on the task** (migration 003): `pr_state` (open, draft, merged, closed), `pr_checks`
  (passing, failing, pending, none), `pr_review` (approved, changes_requested, review_required,
  none), `pr_checked_at`, and `pr_closed_seen_at` (the user dismissed the closed state).
- **Effects**, each recorded as a report from source `github` only when the state changes:
  - merged: the task becomes done ("PR #12 merged").
  - closed without merging: Needs you row "PR #12 was closed without merging", actions
    **Cancel task** and **Keep** (Keep sets `pr_closed_seen_at`, the row goes away).
  - checks failing: Needs you row "Checks are failing on PR #12", action **Open PR**.
  - changes requested: Needs you row "Changes were requested on PR #12", action **Open PR**.
  - Needs you order: blocked, due reminders, closed PRs, failing checks, changes requested,
    review, stale.
- **Code**: `internal/ghpoll` (runs `gh` through a small runner interface, fake in tests; parses the
  JSON; decides effects as a pure function of old and new state) and `internal/domain` writes. It
  runs inside the app process only; `adhd-core mcp` does not poll.
- **Settings > GitHub**: "Connected as QingYunA · checked 2 min ago · Check now", or the fix.

## M4b: management

### Projects

- **New project**: a "+" on the PROJECTS header in the sidebar opens a small sheet: Name, GitHub
  repository (optional, `owner/repo`, validated). With a repository, agent reports from that
  repository land in this project (the existing slug match).
- **Right-click a project**: Rename, Archive, Delete…. Delete asks first, naming how many tasks go
  with it, and removes reports, tasks, milestones and the project in one transaction.
- **Archived projects** are listed in Settings with Restore. A report from an agent into an
  archived project restores it (the agent is working there again).

### Tasks

- **Right-click a row or card**: Status ▸, Priority ▸, Move to ▸ (projects and Inbox), Open PR (when
  there is one), Delete…. Delete asks first.
- **Inspector**: a Project row (same menu as Move to) and a Delete button at the bottom.

### Finding done tasks

- **Search** (`⌘F`, and a search field at the top of the sidebar): matches titles, notes, branch and
  PR number across all projects, done and cancelled included; results grouped by project, Enter
  opens the inspector.
- **Logbook** sidebar item under Inbox: done and cancelled tasks, newest first, grouped by day
  ("Today", "Yesterday", "Oct 12").

### Milestones

- The project header's milestone menu gains "New milestone…" (title, optional due date) and, per
  milestone, Rename, Due date, Close / Reopen. The menu shows even when a project has no
  milestones yet.

## For both halves

- Every new string goes into `app/locales/en.json` and `zh-Hans.json` (see ui-spec Writing).
- Spec first: update `docs/design/ui-spec.md` with the new surfaces before building them.
- Screens in light and dark, English and Chinese, for each new surface.
- TDD; `go test -race ./...`, `go vet`, `gofmt` clean; the MCP tools and their inputs stay
  compatible.
