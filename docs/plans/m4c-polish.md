# M4c: polish — agent-first, larger, calmer

Requested 2026-10-15 after M4. The user's feedback: type is too small, the right-click menu is
plain, completed tasks are hidden, adding a task in a project skips the assistant, the ask field is
small, Settings is ugly.

Product principle (user, 2026-10-15): **agents and people are equal users of this todo list.**
Interaction is simple and the UI is beautiful. The assistant understands what the user means, so
the user does as little as possible by hand (no picking priorities or reminder times from menus
unless they want to); when they must act, the action is one click.

High-fidelity reference: `docs/design/mockups/v2.html` (render: `v2.png`). Match it.

## 1. Type and density (theme)

| Token | Old | New |
|---|---|---|
| `title` | 20/26 600 | 26/32 700 |
| `input` (composer) | 15/20 | 17/25 |
| `body`, `body.strong` (rows, fields) | 13/18 | 15/22 (strong = 500) |
| `meta` (summaries, ages, labels) | 12/16 | 13/18 |
| `section` | 11/14 600 caps +0.4 | 13/18 600, sentence case, `text.secondary`, no tracking |
| `mono` | 11.5/16 | 12.5/18 |
| sidebar rows | 30 pt, 13 | 32 pt, 14; names ellipsize on one line |
| glyphs | 16 | 18 |
| content column | max 820, 56 side padding | max 780, 48 side padding |
| row radius | 8 | 9 |

All views keep using the tokens; no literal sizes.

## 2. The composer (replaces the ask field and the inline add inputs)

- A card on `bg` with a 1 pt `border` (stronger than hairline), 14 pt radius, a soft shadow; focus:
  accent border at 55 % and a 3 pt accent ring at 12 %.
- A text area, 2 lines tall empty, growing to 6 lines then scrolling. Enter sends, Shift+Enter
  breaks the line, Esc as today (cancel, close the reply, blur).
- A bar inside the card: a context **chip** at the left, a hint in `text.tertiary` at the right,
  and a 28 pt round send button (accent when there is text, neutral when empty).
- **Where it is**: top of Now (chip: "✦ Assistant · <model short name>", or "Set up the assistant"
  linking to Settings > Assistant when none), top of every project ("# <project>"), top of Inbox
  ("Inbox"), and the bottom of the inspector ("Change anything — 'move it to friday afternoon'").
- **Scope**: the composer passes its place to the assistant — the project (new tasks go there
  unless the user names another) or the task (the inspector's composer edits that task: "this
  task" means it). Without a model: the text becomes a plain task in that place (the inspector
  composer is hidden then).
- The inline "+ Add a task" rows and the board column "+" inputs are removed; the board's "+"
  focuses the composer.
- Placeholder (Now): `What should I remember? Write it the way you'd say it — "review PR 12 before
  standup, it's urgent"`.
- The reply (assistant sentence, receipts with Undo, questions with options) shows directly under
  the composer that sent it, as today.

## 3. The assistant does the work

- Prompt: infer priority from words ("urgent", "asap", "blocker" → P1, "when I get to it",
  "someday" → P3, P0 only for "drop everything"/"production is down"); infer due vs reminder
  ("remind me" → reminder; "by Friday" → due); default times: morning 9:00, afternoon 14:00,
  evening 19:00, "tonight" 20:00, a bare day → 9:00 for reminders, end of day for due; scope
  rules from section 2; "this"/"it" in the inspector composer = the focused task.
- Receipts show what was inferred: "Fri · P1", "Tomorrow 9:00 AM · reminder".
- Fields the assistant set show a small accent ✦ after the value in the inspector for 10 s.
- Tests: prompt contains scope; scripted-model tests for project-scoped create and task-scoped
  update.

## 4. Completed tasks are visible

- **Now** gains a last section **Done today**: tasks of anyone (agents and me) completed since
  local midnight, newest first, the same row with `done` glyph, title in `text.secondary` weight
  400, and at the right "Claude Code · PR #245 merged" or "You", then the time. Max 5, then
  "Show all in Completed".
- **Project and Inbox lists**: **Done** is a normal group after To do, expanded, newest first, 3
  rows then "Show all N". Backlog stays folded ("Backlog 2 — show").
- The sidebar item "Logbook" is renamed **Completed** (zh: 已完成).
- Clicking the hollow status circle of **my own** task completes it in one click (220 ms fill
  animation); an agent's task's glyph opens the menu (section 5).

## 5. Our own menu (right-click, glyph click, inspector values)

- A drawn popover at the pointer (`RightClicked` + `PointerPosition`, overlay with `AttachTo`),
  268 pt wide, 12 pt radius, `bg` at 96 % with blur if available, shadow; Escape or a press
  outside closes it.
- Contents, top to bottom: **Status** label + a strip of the 6 status glyphs (one click sets it;
  hover tints and shows the name; current one tinted `bg.selected`); **Priority** + a strip
  P0 P1 P2 P3 (P0 in `status.blocked`); separator; **Move to…** (replaces the menu body with a
  filter field and the projects + Inbox); **Remind me…** (Later today 18:00 · Tomorrow 9:00 ·
  Next Monday 9:00 · This weekend Sat 10:00 · No reminder · Custom… opens the calendar);
  **Open PR** when there is one; **Ask about this task…** (⌘J, opens the inspector with its
  composer focused); separator; **Delete** (⌘⌫) in `status.blocked`.
- Delete does not ask: it deletes and shows a toast "Deleted “title” — Undo" for 6 s (mygo
  toast). Undo restores the task with its id, fields and reports (domain `RestoreTask` from a
  snapshot taken before deleting). Projects keep their confirmation (they hold many tasks).
- Keyboard while a row is selected: S status strip, P priority, M move, R remind, ⌘⌫ delete.
- The same pieces replace the native menus of the inspector's Status, Priority, Project and
  Remind values. The project sidebar menu (Rename, Archive, Delete…) may stay native.

## 6. Settings is a window

- `⌘,` and the sidebar gear open a separate settings window (mygo `NewWindow`, 680 pt wide,
  height to content, not resizable), the macOS toolbar-pane pattern: General · Assistant · Agents ·
  GitHub, each an icon above a label; the window title is the pane's name.
- Panes use **grouped lists**: `#F5F5F7` (dark `#1C1C1E`) background, white (dark `#2C2C2E`)
  rounded 10 pt groups, rows ≥ 44 pt with label (+ optional `meta` line) left and control right,
  hairlines between rows, an optional group label above in `section` style.
  - **General**: Language (pop-up: System / English / 简体中文); Archived projects (rows with
    Restore, or the empty line); Data folder (path + Show in Finder); About (version, MIT, GitHub
    link).
  - **Assistant**: Provider (pop-up of presets), Model (pop-up of the preset's suggestions +
    "Other…" which turns into a text field), API key (secure field, "Kept in your Keychain", hidden
    for presets with a fixed key), a status row ("● Works · 640 ms", or the friendly error, or
    "Not tested yet") with "Test"; group **Advanced**: Base URL. Saving a key tests it
    automatically.
  - **Agents**: "Connect Claude Code" with the command in a mono well and Copy; one line on what
    agents report.
  - **GitHub**: the M4a status line and Check now, or the fix.
- The old in-window settings sheet is removed.

## Out of scope

M5 daily briefing, new settings that were not asked for (notification toggles, stale threshold).

## Done when

- Every surface in v2.html exists in the app, in light and dark, English and Chinese, rendered to
  `docs/design/screens/` and reviewed by eye (the lead reviews every new PNG).
- `go test -race ./...`, `go vet`, `gofmt`, `mygo i18n check` clean; the MCP tools unchanged.
