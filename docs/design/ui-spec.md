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
| `title` | 20 / 26 | 600 | main window project title |
| `input` | 15 / 20 | 400 | ask field |
| `body` | 13 / 18 | 400 | rows, cards, fields |
| `body.strong` | 13 / 18 | 500 | task titles |
| `meta` | 12 / 16 | 400 | summaries, ages, field labels |
| `section` | 11 / 14 | 600, tracking +0.4, uppercase | section headers |
| `mono` | 11.5 / 16 | 400 | branch, `#42` |

### Space, shape, motion

- 4 pt grid. Common gaps 4, 8, 12, 16, 24.
- Radii: 6 (pills, inputs), 8 (cards, rows), 12 (ask field, Needs you container, popovers).
- Shadows only on floating things: drag ghost, popovers, the window.
- Motion 160 ms ease-out for hover, selection and drawer; 220 ms spring for a card
  appearing and a card settling after a drop. Respect Reduce Motion: fade only.

## Glyphs

16 pt, drawn as vector icons. Shape carries meaning; color repeats it.

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

Clicking the status glyph on a row or card opens a small status menu. Marking done plays a
220 ms fill-and-check animation, then the item fades from active lists.

## Surfaces

One window. No tray icon, no flyout panel, no global shortcut (decided 2026-10-14: the user
found the tray pointless and shortcuts conflict). The Dock icon opens the window; closing the
window hides it; Quit quits. The high-fidelity reference for this section is
`docs/design/mockups/now.html` (rendered: `now.png`). Match it.

### Window (1100 x 720, min 860 x 540)

Unified title bar. Columns: sidebar (216 pt) | content | inspector (360 pt, only when a task
is open). Content is a centered column, max 820 pt wide, 56 pt side padding, so lists read
like a document, not a spreadsheet.

**Sidebar.** `Now` (count = items in Needs you + Agents at work + Yours), `Inbox` (count),
then `PROJECTS`: one row per project, name only; a 6 pt red dot at the right when it has a
blocked task, otherwise the count of active tasks in `text.tertiary`. A project with no active
task shows no count. Settings is a gear button at the bottom. Selected row: `bg.selected`,
7 pt radius.

### Now (home)

Header: `Now` in `title` + the date in `text.tertiary`.

**Ask field.** 46 pt tall, 12 pt radius, `bg.subtle`, a small accent sparkle, placeholder
`Tell me what to remember — "remind me to review PR #12 at 4pm"`, a `⌘K` key cap at the right.
`⌘K` focuses it from anywhere. Until M3, Enter creates an Inbox task and shows an inline
"Added to Inbox" confirmation under the field for 1.5 s.

Then up to three sections. **A section with nothing in it is not drawn at all.**

1. **Needs you** — inside one bordered container (12 pt radius, rows split by hairlines):
   - blocked tasks (any source): summary line `Agent is waiting: "<next_step or summary>"`,
     action **Open** (opens the inspector);
   - review tasks with a PR: `PR #12 is ready for your review`, action **Open PR** (browser);
   - stale agent tasks (no report for `stale_hours`): `No update for 3d`, action **Open**;
   - reminders that are due: `You asked to be reminded at 4:00 PM`, action **Done**.
   Order: blocked, due reminders, review, stale. Each row: glyph (reminders use a bell in
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

**All clear.** When all three sections are empty: a centered calm message under the ask field —
"All clear." and in secondary "Agents report here through the adhd-todo MCP server." with a
"How to connect" link (opens Settings).

### Project

Header: project name in `title`, repo slug in `text.tertiary`, a milestone menu (only if the
project has milestones) and a `List | Board` segmented control at the right (List default).

**List view.** Groups in this order: Blocked, In review, Doing, To do. Group header: glyph +
name (`body.strong`) + count in `text.tertiary`. Rows indent under the header text (no glyph
per row; the group says the status). Row: title, branch pill, `#PR`, summary line for agent
tasks, age or due label on the right, stale label when stale. **Empty groups are not drawn.**
After To do: a quiet "+ Add a task" row (inline input; Enter creates a todo in this project).
Last line: `Done 4 · Backlog 2 — show` in `text.tertiary`; clicking expands those two groups.

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
  content, and the branch, pull request and age below 460 pt.
- **Inline add** keeps its input open and empty after Enter, to catch several thoughts in a row;
  Esc closes it.
- A notification click opens the task in the inspector over Now.

### Inbox

Same list view as a project, without the board toggle; tasks with no project.

### Inspector (360 pt, slides in 160 ms)

Unchanged from the previous spec: large editable title; property list (Status, Priority,
Milestone, Due, Remind, Branch, Pull request) with left labels in `meta` secondary and values
editable in place; Notes; `ACTIVITY` timeline newest first. Save on blur or Enter. Esc closes.

### Keyboard

`⌘K` ask field · `↑ ↓` move selection through rows on the current screen · `↵` open the
selected row in the inspector · `Space` toggles done for your own tasks · `Esc` closes the
inspector, then clears the selection · `⌘1` Now, `⌘2` Inbox.

### Settings

A sheet: "Connect Claude Code" with the copyable registration command, data folder path,
About. Nothing else.

## Writing

- English UI copy for now. Sentence case everywhere ("Add a task", not "Add A Task").
- Relative times: "now", "2m", "3h", "2d", then a date ("Oct 12").
- Never blame: "Couldn't reach the data store. Retrying…" rather than "Error".

## Review checklist

Before calling a screen done, render it in light and dark and check:

- [ ] At most three type sizes are visible.
- [ ] Only blocked, stale and selection use strong color.
- [ ] Every status is distinguishable in grayscale.
- [ ] Nothing is boxed that could be separated by space.
- [ ] Text is aligned to a shared left edge in each section.
- [ ] Empty, loading and error states exist and read well.
- [ ] Hit targets are at least 24 px; rows at least 32 px.
