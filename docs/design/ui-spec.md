# adhd-todo UI spec

The app answers one question at a glance: **what is happening right now, and what needs me?**
Every screen is judged against that question. The audience is a developer with ADHD who runs
several coding agents in parallel. Visual noise costs them focus, so the UI is quiet by
default and loud only where something needs attention.

Built with mygo native UI (`github.com/egoist/mygo/ui`, v0.4.8). No webview.

## References and what we take from each

| Reference | Take | Leave |
|---|---|---|
| Things 3 | Calm whitespace, typographic hierarchy instead of boxes, one accent color, delight in small motion (checkbox completion). | Its date-centric model. |
| Linear | Status and priority glyphs that read without color, hairline borders, dense but legible rows, keyboard-first, inspector on the right. | Its heavy feature surface. |
| Raycast | One large input at the top, a list with a keyboard-highlighted row. | Command-palette complexity, the floating panel. |
| macOS 27 system apps (Reminders, Notes) | Sidebar material, native metrics, Liquid Glass on floating surfaces. | |

When in doubt, remove. A screen is finished when nothing more can be taken away.

## Principles

1. **Attention has a budget.** Only three things may use strong color: blocked, stale, and the
   keyboard selection. Everything else is neutral.
2. **Status reads without color.** Every status has a distinct glyph shape; color is a second cue.
3. **Hierarchy through type, not boxes.** Prefer weight, size and secondary color over borders,
   cards and backgrounds. Cards exist only on the board, where they are dragged.
4. **Agents and me are different.** Agent work shows its latest report and its age. My work
   shows when it is due or when I asked to be reminded.
5. **Never a dead end.** Every empty state says what will appear there and how to make it appear.
6. **Fast is a feature.** The ask field is one keystroke away (⌘K). No spinner for local data.

## Tokens

### Color

Neutrals are a cool gray. One accent. Status colors are muted so a board of them stays calm.

| Token | Light | Dark | Use |
|---|---|---|---|
| `bg` | `#FFFFFF` | `#1C1C1E` | window content |
| `bg.sidebar` | material (vibrancy) | material (vibrancy) | sidebar |
| `bg.subtle` | `#F5F5F7` | `#232326` | board columns, input wells |
| `bg.hover` | `#0000000A` | `#FFFFFF0F` | row hover |
| `bg.selected` | `accent @ 12%` | `accent @ 22%` | selected row, keyboard highlight |
| `border` | `#0000001A` | `#FFFFFF1A` | hairlines (1 px) |
| `text` | `#1D1D1F` | `#F5F5F7` | primary |
| `text.secondary` | `#6E6E73` | `#A1A1A6` | metadata, summaries |
| `text.tertiary` | `#AEAEB2` | `#636366` | placeholders, hints |
| `accent` | `#5B5BD6` | `#7C7CF0` | selection, primary button, focus ring, links |
| `status.backlog` | `#AEAEB2` | `#636366` | |
| `status.todo` | `#8E8E93` | `#8E8E93` | |
| `status.doing` | `#3B82F6` | `#60A5FA` | |
| `status.blocked` | `#E5484D` | `#FF6369` | the only red in the app |
| `status.review` | `#E0A800` | `#F5C542` | |
| `status.done` | `#30A46C` | `#4CC38A` | |
| `stale` | `#D97706` | `#F59E0B` | stale label text |

Follow the system appearance (light/dark) and the system accent only if the user changed it
from the default; otherwise use `accent`.

### Type

System font (SF Pro), SF Mono for branches and PR numbers. Use at most three sizes in one view.

| Token | Size / line | Weight | Use |
|---|---|---|---|
| `title` | 26 / 32 | 700 | main window project title |
| `input` | 17 / 25 | 400 | the composer |
| `body` | 15 / 22 | 400 | rows, cards, fields, menus |
| `body.strong` | 15 / 22 | 500 | task titles |
| `meta` | 13 / 18 | 400 | summaries, ages, field labels |
| `section` | 13 / 18 | 600, `text.secondary`, sentence case, no tracking | section headers |
| `mono` | 12.5 / 18 | 400 | branch, `#42` |

M4c made type one step larger everywhere (2026-10-15): the first version read small on a retina
display. Views keep using the tokens; no literal sizes.

### Space, shape, motion

- 4 pt grid. Common gaps 4, 8, 12, 16, 24.
- Radii: 6 (pills, inputs), 9 (cards, rows), 12 (composer, Needs you container, popovers).
- Shadows only on floating things: drag ghost, popovers, the window.
- Motion 160 ms ease-out for hover, selection and drawer; 220 ms spring for a card
  appearing and a card settling after a drop. Respect Reduce Motion: fade only.

## Glyphs

18 pt, drawn as vector icons. Shape carries meaning; color repeats it.

| Status | Glyph |
|---|---|
| backlog | dashed circle |
| todo | empty circle |
| doing | circle with the right half filled |
| blocked | filled circle with a white `!` |
| review | circle with an inner dot (waiting on someone) |
| done | filled circle with a white check |
| cancelled | circle with a slash, tertiary color |

Priority: three ascending bars. P0 = three bars in `status.blocked` color, P1 = three bars,
P2 = two bars, P3 = one bar, in `text.secondary`. P2 is the default and shows **nothing** on
cards and rows, so only deliberate priorities draw the eye.

Clicking the status glyph of **my own** task marks it done at once (one click, no menu); marking
done plays a 220 ms fill-and-check animation (a plain fill with Reduce Motion), then the item moves
to Done. Clicking the glyph of an **agent's** task, or of any finished task, opens the **Task menu**
(below), so an agent's work is never completed by accident and a finished task can be reopened.

## Surfaces

One window. No tray icon, no flyout panel, no global shortcut (decided 2026-10-14: the user
found the tray pointless and shortcuts conflict). The Dock icon opens the window; closing the
window hides it; Quit quits. The high-fidelity reference for this section is
`docs/design/mockups/now.html` (rendered: `now.png`). Match it.

### Window (1100 x 720, min 860 x 540)

Unified title bar. Columns: sidebar (216 pt) | content | inspector (360 pt, only when a task
is open). Content is a centered column, max 780 pt wide, 48 pt side padding, so lists read
like a document, not a spreadsheet.

**Sidebar.** A search field first (see **Search**), then `Now` (count = items in Needs you + Agents
at work + Yours), `Inbox` (count) and `Completed` (no count: it only grows), then `Projects` (sentence
case, `section` style) with a quiet `+` at the right of its header: one row per project, name only,
**on one line, ellipsized** (`answer-me-with-ht…`); a 6 pt red dot at the right when
it has a blocked task, otherwise the count of active tasks in `text.tertiary`. A project with no active
task shows no count. Settings is a gear button at the bottom. Rows are 32 pt tall in `body` 14 pt
(the sidebar is one step smaller than the page); selected row: `bg.selected`, 7 pt radius.

- **`+` on Projects** opens the **New project** sheet (below). It is a 24 pt target in `text.tertiary`,
  `text.secondary` on hover.
- **Right-click a project**: `Rename`, `Archive`, a separator, `Delete…`. Rename edits the name in the
  row itself: the label becomes an input with the cursor at the end, Enter saves, Esc and a click away put
  the old name back. Archive acts at once (the project is in Settings > General > Archived projects,
  one click from coming back); Delete… asks first (**Delete confirmation**). Archiving or deleting the
  project on screen goes to Now.
- A project with no repository shows no repository line under its name on its page.

### Now (home)

Header: `Now` in `title` + the date in `text.tertiary`.

**Ask field.** 46 pt tall, 12 pt radius, `bg.subtle`, a small accent sparkle, placeholder
`Tell me what to remember — "remind me to review PR #12 at 4pm"`, a `⌘K` key cap at the right.
`⌘K` focuses it from anywhere. With a model connected, Enter sends the text to the built-in
assistant (see **Assistant** below). With none, Enter creates an Inbox task and shows an inline
"Added to Inbox" confirmation under the field for 1.5 s.

Then up to four sections. **A section with nothing in it is not drawn at all.**

1. **Needs you** — inside one bordered container (12 pt radius, rows split by hairlines):
   - blocked tasks (any source): summary line `Agent is waiting: "<next_step or summary>"`,
     action **Open** (opens the inspector);
   - review tasks with a PR: `PR #12 is ready for your review`, action **Open PR** (browser);
   - stale agent tasks (no report for `stale_hours`): `No update for 3d`, action **Open**;
   - reminders that are due: `You asked to be reminded at 4:00 PM`, action **Done**;
   - a pull request closed without merging (M4a): `PR #12 was closed without merging`, actions
     **Cancel task** (status cancelled) and **Keep** (the row goes away for good, until the pull
     request changes state again);
   - a pull request whose checks are failing: `Checks are failing on PR #12`, action **Open PR**;
   - a pull request where changes were requested: `Changes were requested on PR #12`, action
     **Open PR**.
   The three pull request rows come from the GitHub poller (see **GitHub** under Settings) and
   apply to any task with a pull request that is not done or cancelled; failing checks and
   requested changes only while the pull request is open (not a draft).
   Order: blocked, due reminders, closed pull requests, failing checks, requested changes, review,
   stale. A task shows once, under the first reason it has. Screens `now-pull-requests-*`. Rows age
from GitHub's own report, not the agent's last word; GitHub's reports (agent `github`) show in the
task's Activity but never count as the agent speaking. A row with two buttons (the closed pull
request) puts them under its summary when the content is under 560 pt wide, so the title keeps
its room beside the inspector. Each row: glyph (reminders use a bell in
   `accent`), title, project name in `text.tertiary`, one summary line, age, and the action as
   a small tinted button (`bg.selected`, accent text). This container is the only box on the
   screen.
2. **Agents at work** — agent tasks in doing, newest report first, across all projects. Row:
   glyph, title, branch pill, project name (only when not inside a project), latest report
   summary on line 2, age on the right. A 6 pt `status.doing` dot with a soft ring before the
   age when the last report is under 10 minutes old ("live").
3. **Yours** — user tasks in todo / doing: due or reminded today and overdue first, then the
   next 7 days, then undated (at most 7 undated; a "Show all in Inbox" link after). Row: glyph,
   title, project name, due label on the right ("Today", "Tomorrow", "Fri", "Oct 20").

4. **Done today** — what was finished since local midnight, by anyone: agents and me, newest first,
   so the day shows what got done and not only what is left. The same row as the others with the
   `done` glyph, the title in `text.secondary` at weight 400, and at the right who did it, then
   the time of day: `Claude Code · PR #245 merged` (the agent's name, then what its latest report
   says, cut to one line) or `You`, then `2:05 PM`. At most five; past them a `Show all in
   Completed` link opens Completed. The section does not count toward the sidebar's Now badge and
   never makes Now "not clear": **All clear.** looks at the first three. Who finished a task:
   the agent that filed a report at the moment it closed (a report within five seconds of the
   completion, or later), else me.

**All clear.** When the first three sections are empty (Done today may still show below it): a centered calm message under the ask field —
"All clear." and in secondary "Agents report here through the adhd-todo MCP server." with a
"How to connect" link (opens Settings).

### Assistant (M3)

The ask field is the entry to a small built-in agent (Eino, `internal/agent`). It acts at once and
reports what it did with an Undo; it asks only when an action would be wrong without the answer.
Screens: `now-thinking`, `now-reply-receipt`, `now-question`, `now-no-model` (light and dark).

- **Working.** The sparkle pulses (a plain accent sparkle with Reduce Motion), the empty field's
  placeholder reads `Thinking…`, and the typed text has moved into the sheet. One request at a
  time: Enter while it works keeps what is typed. **Esc cancels** the request (its context is
  cancelled); with nothing changed yet the sheet closes and the text returns to the field.
- **Conversation sheet.** Opens directly under the ask field, inside the content column, and
  pushes the sections down; a hairline marks where it ends. It is part of the page, not a chat:
  no bubbles, no avatars, no timestamps, everything on the column's left edge. It holds this
  conversation only (the last three exchanges), never stored.
  - *My line*: `body` in `text.secondary`.
  - *Reply*: `body` in `text`, one short sentence, 6 pt under my line.
  - *Receipts*: one row per change, in the list-row style: status glyph (a bell in `accent` when
    a reminder was set), title in `body.strong`, a quiet detail in `text.tertiary`
    (`Tomorrow 9:00 AM · reminder`, `Added to Inbox`, `Marked done`, `Updated · due Fri, P1`), and
    **Undo** at the right (`meta`, medium, `text.secondary`, tinted on hover). Clicking the row
    opens the task. After Undo the title is struck through in `text.tertiary` and the row says
    `Undone`.
  - *Question*: `body` text; its 2 to 4 options are small tinted buttons, the same style as the
    Needs you actions, wrapping onto a second line when they must. A click sends the option as
    my answer; typing a free answer works too (the placeholder says `Type an answer, or pick one
    below`). A question never times out. Once answered its buttons are gone.
  - *Error*: neutral words ("Couldn't reach the model. Is the address right and the server
    running?", "The provider didn't accept the API key.") with **Retry** as a link. No red.
  - After a turn that asked nothing and failed nothing the sheet **closes itself after 4 s**
    (Undo restarts the 4 s), or at once on **Esc**. A second **Esc** with nothing else to close
    makes the assistant forget the conversation (it also forgets after 10 minutes idle).
- **No model connected.** Focusing the ask field shows one line under it, `Connect a model to
  let me understand you — Set up` (`meta`, `Set up` is a link), which opens Settings > Models.
  It stays until you leave the screen so the click still finds it. Enter keeps adding Inbox
  tasks (the M2 behavior).
- Times in receipts follow the Now screen's words: `Today`, `Tomorrow`, a weekday, then a date,
  followed by the clock (`9:00 AM`).

### Project

Header: project name in `title`, repo slug in `text.tertiary`, a milestone menu (always shown, even
before the project has a milestone) and a `List | Board` segmented control at the right (List default).

**Milestone menu.** The button reads `All milestones` or the milestone being filtered by. The menu:

1. `All milestones`, a separator, then one item per milestone (checked when it filters the page),
   as before: `v0.2 · due Oct 28`, `v0.1 · closed`;
2. a separator and `New milestone…`;
3. when there are milestones, a separator and one submenu per milestone, `Edit “v0.2”`, holding
   `Rename…`, `Due date…` and `Close milestone` (`Reopen milestone` for a closed one).

`New milestone…`, `Rename…` and `Due date…` open the **Milestone sheet**. Closing and reopening act
at once. A project that has no milestone shows the menu with `All milestones` and `New milestone…`.

**List view.** Groups in this order: Blocked, In review, Doing, To do. Group header: glyph +
name (`body.strong`) + count in `text.tertiary`. Rows indent under the header text (no glyph
per row; the group says the status). Row: title, branch pill, `#PR`, summary line for agent
tasks, age or due label on the right, stale label when stale. **Empty groups are not drawn.**
After To do comes **Done**, a group like the others and **expanded**: glyph + `Done` + count,
its rows newest first in `text.secondary`, with who finished them and the day at the right (`Claude
Code  Yesterday`, `You  Oct 12`), three rows and then a `Show all 4 →` link in `accent` that expands
the group in place. Last line: `Backlog 2 — show` in `text.tertiary` (folded: Backlog stays out of
the way); clicking it opens the group. With no backlog the line is not drawn.

**Board view.** Kept for people who like columns, but it shows only columns that have tasks,
plus To do (always, as the place to add). No collapsed strips, no zero-count columns. Cards as
before; drag between visible columns changes status. Dragging also reveals thin drop targets
for hidden statuses at the right edge.

Decisions made while building it (2026-10-14):

- **Backlog and Done on the board** follow the list's "Done · Backlog — show" line, which sits under
  the board header. Folded, they are drop targets that appear while a card is dragged. With them
  shown by default the seeded project needed six columns and scrolled sideways at 1100 pt.
- **Drop targets** for hidden statuses float over the right edge of the board, so they are in reach
  however far it is scrolled.
- **Review without a PR** still goes to Needs you (summary "Ready for your review", action Open).
  A review task of my own does not: it waits on someone else.
- **Yours** shows dated tasks due within 7 days (or overdue); later ones appear in their project or
  the Inbox only.
- **Reminders** stay in Needs you until marked done; the notification fires once, the row does not
  go away.
- **Tight rows** (the inspector open on a small window) drop the project name below 600 pt of
  content, and the branch, pull request and age below 460 pt. Needs you rows with two buttons put
  them under the summary below 560 pt, and every Needs you row does below 420 pt (the 860 pt window
  with the inspector open leaves 282 pt), so the title keeps its width. A finished row drops who
  finished it below 460 pt and what they said below 600 pt.
- **Inline add** keeps its input open and empty after Enter, to catch several thoughts in a row;
  Esc closes it.
- A notification click opens the task in the inspector over Now.

### Inbox

Same list view as a project, without the board toggle; tasks with no project.

### Inspector (360 pt, slides in 160 ms)

Unchanged from the previous spec: large editable title; property list (Status, Priority, **Project**,
Milestone, Due, Remind, Branch, Pull request) with left labels in `meta` secondary and values
editable in place; Notes; `ACTIVITY` timeline newest first. Save on blur or Enter. Esc closes.

- The **Status**, **Priority**, **Project** and **Remind** values open the pieces of the **Task
  menu** (below) under the value, not a system menu: the status strip, the priority strip, the
  `Move to…` list (`Inbox` and the projects, the current one checked) and the reminder presets.
  Moving a task to another project clears its milestone (milestones belong to one project); the
  Milestone row follows the project.
- At the bottom, below ACTIVITY, a quiet `Delete task…` in `text.secondary` (no red, no box). It deletes
  at once like the menu's Delete (with the Undo toast) and closes the pane.

### Task menu

Our own popover, not the system's, so it can show the status glyphs and priority as one-click
strips. The same menu on every surface that shows the task (list rows, Now rows, search and
Completed rows, board cards), opened by:

- a **right-click** on a row or card (at the pointer);
- a click on the **status glyph** of an agent's task, or of a finished task (at the glyph);
- **S**, **P**, **M**, **R** on the selected row (below the row), and `⌘J`, `⌘⌫` (see Keyboard).

It is 268 pt wide, 12 pt radius, opaque `bg` (the mockup's blur is not available in mygo; at 96 % the
rows behind showed through), a 1 pt `border` and a soft shadow; it moves into the window, or above
the point it hangs from, when it would leave it. `Esc` or a press outside closes it (the press goes on to what is
under it, so a right-click on another row moves the menu there). Reference: frame 3 of
`docs/design/mockups/v2.html`.

Main page, top to bottom (10 pt padding):

1. `Status` label in `section` style and a **strip** of the status glyphs: backlog, to do, doing,
   blocked, review, done, cancelled. One click sets the status and closes the menu. Hover
   tints a button (`bg.hover`) and shows its name as a tooltip; the current status is tinted `bg.selected`.
   (The mockup draws six, without cancelled; the seventh keeps cancelling a task one click away.)
2. `Priority` and a strip `P0 P1 P2 P3` (P0 in `status.blocked`, the current one tinted).
3. a hairline;
4. `Move to…` (key hint `M`): the menu's body is replaced by a filter field and a list of the
   projects and `Inbox` (the current one checked). Typing filters, `↑ ↓` move, `↵` moves the task,
   `Esc` goes back to the main page;
5. `Remind me…` (`R`): replaced by presets, each with the moment it means at the right (the time
   alone for today and tomorrow, `Mon 9:00 AM` for the others): `Later today` (6:00 PM, only while
   that is over half an hour away), `Tomorrow` (9:00 AM), `Next Monday` (9:00 AM, left out on a
   Sunday, when it is tomorrow), `This weekend` (Saturday 10:00 AM, not from Saturday 9:30 or on
   Sunday), `No reminder`, and `Custom…`, which shows a calendar with a time of day, Clear and
   `Set reminder`;
6. `Open PR`, only when the task has a pull request;
7. `Ask about this task…` (`⌘J`): opens the inspector with its composer focused;
8. a hairline and `Delete` (`⌘⌫`) in `status.blocked`, the only red.

**Delete does not ask.** The task goes and a toast says `Deleted “title”` with **Undo** for six
seconds; Undo puts the task back with its id, fields and reports. (A project keeps its confirmation: it
holds many tasks.) The inspector's values open the same pieces alone: Status shows the status strip,
Priority the priority strip, Project the `Move to…` list, Remind the presets.

A project's menu is in the Sidebar and stays a system menu.

### Search

`⌘F`, or a click in the search field at the top of the sidebar, puts the cursor there. The field is a
28 pt well on the sidebar material (`bg.hover`, 7 pt radius) with a magnifier, the placeholder
`Search` and a `⌘F` hint at the right while empty.

Typing replaces the page with **Search results**; while they show, no sidebar row is highlighted, and
the screen under them comes back when the search ends. The page
header reads `Search` with `12 results for “login”` in `text.tertiary`. Results are tasks of every
project and the Inbox, whatever their status, done and cancelled included (the tasks of archived
projects are left out), matching every word of the query in the title, notes or branch, or a pull
request number (`#42` and `42`). They are grouped by project, the Inbox last, groups ordered by the
newest task in them. Group header: `#` mark, project name in `body.strong`, count in `text.tertiary`.
Row: status glyph (a button, like on Now), title, branch pill and `#PR`, and at the right the age of
the last change. Done and cancelled rows keep their glyph (a button, so a finished task can be reopened from here).

- **No match**: `No tasks match “xyz”` and, in secondary, `Search looks in titles, notes, branches and
  pull request numbers, in every project.`
- **Keys**: in the field `↓` moves to the first result (and on down through the rows, as on every
  list), `↑` from the first goes back to the field, `↵` opens the highlighted row in the inspector
  (the first result when none is highlighted yet), `Esc` clears the text and leaves the field.
  With a result highlighted and the inspector open, `Esc` closes the inspector first, then clears the
  highlight, then the search. Clicking a sidebar item or `⌘1` `⌘2` `⌘3` ends the search.
- At most 100 results are shown; a quiet last line says when there are more.

### Completed

The `Completed` sidebar item (`已完成`, was Logbook), under Inbox: the done and cancelled tasks of every
active project and the Inbox, newest first by completion, grouped by day. Page title `Completed`,
subtitle `Done and cancelled, newest first`. Now's `Show all in Completed` and a list's
`Show all` lead the eye here. Day header in `body.strong` with the count in `text.tertiary`: `Today`, `Yesterday`,
then a date (`Oct 12`, `Oct 12, 2025` in another year), written by `app/words`. Row: the status glyph
(done check or cancelled slash, a button: choosing another status reopens the task), title, project
name, and at the right who finished it (`Claude Code`, `You`) and the time of day.

- **Empty**: `Nothing finished yet` and `Tasks you mark done or cancel collect here, newest first.`
- 100 tasks are listed; `Show older` at the end loads 100 more.
- The same keys as any list: `↑ ↓ ↵ Space Esc`, `⌘3` opens it.

### Sheets and confirmations

Small sheets on the dimmed window, built like Settings (12 pt radius, 24 pt padding) but 400 pt wide.
`Esc` and the Cancel button close them; the primary button is the accent one.

- **New project**: title `New project`; field `Name` (focused); field `GitHub repository` (optional,
  mono, placeholder `owner/repo`) with the line `Agent reports from this repository land in the
  project.` Create is off until the name has text. A pasted GitHub address is read as `owner/repo`.
  Problems read under the field in `text.secondary`, never red: `Use the form owner/repo, like
  QingYunA/adhd-todo.` and `“adhd-todo” already uses this repository.` Enter in either field creates;
  the new project opens.
- **Milestone sheet**: `New milestone` (Title, Due date optional), `Rename milestone` (Title) and
  `Due date` (a calendar and a Clear link). Done is the primary button; Enter saves.
- **Delete confirmation** (a project, a task): title `Delete “name”?`; body in `text.secondary`.
  For a project: `This removes its 14 tasks, their reports and its milestones. It can't be undone.
  To keep them, archive it instead.` (`This removes the project and its milestones.` when it has no
  task.) For a task: `Its progress reports go with it. This can't be undone.` Buttons `Cancel`
  (focused, so Enter does not delete) and `Delete` (accent, not red). Esc cancels.

### Keyboard

`⌘K` ask field · `⌘F` search · `↑ ↓` move selection through rows on the current screen · `↵` open the
selected row in the inspector · `Space` toggles done for your own tasks · `Esc` closes the
inspector, then clears the selection, then the search · `⌘1` Now, `⌘2` Inbox, `⌘3` Completed.

With a row selected: `S` status, `P` priority, `M` move to, `R` remind: each opens the Task menu on
that piece, where `←` `→` or `↑` `↓` move, `↵` chooses and `Esc` closes; `⌘J` asks the assistant
about the task (opens the inspector with its composer focused); `⌘⌫` deletes it, with Undo.

While a text input has the keyboard its keys are its own (`Element.OnTextKey`), and the window's
arrows, `↵`, `Space` and `Esc` stay out of the way:

| Input | `↵` | `Esc` | `↓` |
|---|---|---|---|
| Ask field | sends; on an empty field opens the highlighted row | cancels a running request, else closes the sheet | moves into the rows |
| Inline "Add a task" | adds and stays open and empty | closes | |
| Inspector title, branch | saves | puts the stored text back and leaves the field | |
| Inspector notes | breaks a line (saves on blur) | puts the stored text back and leaves the field | |
| Search field | opens the highlighted result (the first when none) | clears the text and leaves the field | moves into the results |
| Rename project (sidebar) | saves | puts the old name back | |
| New project, milestone sheets | creates / saves | closes the sheet | |

### Settings

A 640 pt sheet with a `General | Models` switch (the quiet segmented well of the List | Board
toggle) beside Done.

**General**: "Connect Claude Code" with the copyable registration command, **GitHub**, data
folder path, **Archived projects**, Language, About. Nothing else.

**GitHub** (M4a, screens `settings-github-*`): the pull request poller's state, from the `gh` CLI
the user is already signed in to (the app owns no token). One bold title and one line:
- *Connected*: `Connected as QingYunA · checked 2 min ago` in `meta` / `text.secondary`, and a
  **Check now** link at the right of the line. While a check runs the link reads `Checking…` and
  is inert; before the first check ends the line says `Checking GitHub…`.
- *No `gh`*: `The GitHub CLI isn't installed. Install it to follow your pull requests.` and the
  command `brew install gh` in the mono well of "Connect Claude Code".
- *Signed out*: `You're not signed in to GitHub. Sign in to follow your pull requests.` and
  `gh auth login` in the mono well.
- *Offline*: `Can't reach GitHub. Trying again soon.` Never red; polling retries on its own.
Polling runs at launch, every 5 minutes, and when the window comes back to the front if the last
check is older than a minute; at most 4 `gh` calls at once.

*Archived projects**, Language, About. Nothing else.

*Archived projects* is a block under the data folder: one 32 pt row per archived project (name in
`body`, repository in `text.tertiary`, a tinted **Restore** button at the right). With none it says
`No archived projects. Archive one from its menu in the sidebar.` At most five rows show; the list
scrolls beyond. A report from an agent in an archived project's repository restores it too.

**Models** (screens `settings-models`, `settings-models-edit`): the provider list at the left,
the open provider's fields at the right.
- *List*: one 32 pt row per preset: Anthropic, Magpie, OpenAI, DeepSeek, Kimi, OpenRouter,
  Ollama, Custom. A radio at the left marks the one in use (`accent`); the row being viewed gets
  `bg.selected`. Clicking a row opens its fields; it does not switch the provider.
- *Fields* (30 pt wells on `bg.subtle`, label above in `meta`): **Base URL** (the preset's value,
  its default shown as the placeholder), **Model** (with the preset's suggestions as small mono
  chips under it; a click fills it), **API key** (a secure field; once stored the placeholder
  says "Saved in your Keychain — paste a new one to replace it"; the key is never shown again).
  Fields save on Enter or blur. Magpie and Ollama have no key field; Magpie says "Uses your
  Magpie gateway at 127.0.0.1:3425". Custom adds an `OpenAI-compatible | Anthropic API` switch.
- *Header of the fields*: provider name, a note (or "Needs an API key"), and at the right `In
  use` with a check, or a tinted **Use this one** button.
- **Test** sends a 1-token request with what is typed, saved or not, and says `Works · 640 ms`
  or the error in the words above (`text.secondary`; no green, no red).
- Keys live in the system keychain (service `adhd-todo`), never in the database, the provider
  file or the logs. The active provider and per-provider overrides are in `models.json` next to
  the database.

## Writing

- UI copy comes in English and Simplified Chinese. English is the source; the system language picks
  the language, and Settings > General > Language can override it (System, English, 简体中文).
- English: sentence case everywhere ("Add a task", not "Add A Task"). Never blame: "Couldn't reach
  the data store. Retrying…" rather than "Error".
- Chinese reads like macOS's own Simplified Chinese: short, natural, no translationese, "你" without
  "您", full-width punctuation (，。：“”), a space between a number and Latin text but none
  before a unit ("3天前", "2分钟前"). `⌘K`, PR, MCP, Claude Code and branch names stay as they are.
  Examples: "需要你处理", "Agent 进行中", "你的任务", "全部处理完了。", "已添加到收件箱".
  Management words: "新建项目", "重命名", "归档", "删除…", "移到" (Move to), "搜索", "已完成" for
  Completed (as macOS Reminders names it; the page's subtitle says it holds cancelled tasks too, and
  "日志" would read as log files to a developer).
- Section headers are sentence case, `section` style, not tracked capitals, in both languages.
- Relative times: "now", "2m", "3h", "2d", then a date ("Oct 12"); in Chinese "刚刚", "2分钟前",
  "3小时前", "2天前", then "10月12日". Due labels: "Today", "Tomorrow", "Fri", "Oct 20", the time of
  day for a reminder today ("4:00 PM", "下午4:00"). Dates and hours follow the user's regional
  formats (`ui.Context.Region()`, so 12 or 24 hours), written by `app/words`.
- The assistant's own replies are in the language the user writes in; they are not in the catalog.

### Adding a string

1. Add the key to `app/locales/en.json` with an `@key` description for translators (what it is and
   where it shows; `placeholders` with an `example` for each argument).
2. Add the same key to `app/locales/zh-Hans.json` with the same placeholders.
3. `cd app && go tool mygo generate` writes `locales/catalog_gen.go`; commit it. `go tool mygo i18n
   check` and `go test ./app/locales` must pass.
4. Say it in a view with `m.say.KeyName(args)` (a `locales.Messages` for the window's language);
   outside a view, `locales.KeyName(args)` says it in the language the app shows. Never build a
   sentence from pieces: give the message the arguments it needs, so each language orders them.
5. Times go through `app/words` (`m.clock`), never `time.Format`. Packages in `internal/` hold no
   wording: they return data (kinds, times, values) and `app/views` words it.
6. Keys of elements (`c.Key`) never use a translated text.

## Review checklist

Before calling a screen done, render it in light and dark and check:

- [ ] At most three type sizes are visible.
- [ ] Only blocked, stale and selection use strong color.
- [ ] Every status is distinguishable in grayscale.
- [ ] Nothing is boxed that could be separated by space.
- [ ] Text is aligned to a shared left edge in each section.
- [ ] Empty, loading and error states exist and read well.
- [ ] Hit targets are at least 24 px; rows at least 32 px.
