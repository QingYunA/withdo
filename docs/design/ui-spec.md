# adhd-todo UI spec

The app answers one question at a glance: **what is happening right now, and what needs me?**
Every screen is judged against that question. The audience is a developer with ADHD who runs
several coding agents in parallel. Visual noise costs them focus, so the UI is quiet by
default and loud only where something needs attention.

Built with mygo native UI (`github.com/egoist/mygo/ui`, v0.5.1). No webview.

M4d (2026-10-15) made every screen **one title, one input, one list**: what is not needed to read or
act appears on hover, in a right-click menu or in the inspector. The reference for it is
`docs/design/mockups/v3.html` (rendered: `v3.png`); `docs/plans/m4d-restraint.md` is the plan.

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
   cards and backgrounds. Cards exist only on the board, where they are dragged; a board lane is no
   box, only its header and its cards.
4. **Agents and me are different.** Agent work shows its latest report and its age. My work
   shows when it is due or when I asked to be reminded.
5. **Never a dead end.** Every empty state says what will appear there and how to make it appear.
6. **Fast is a feature.** The composer is one keystroke away (⌘K). No spinner for local data.

## Tokens

### Color

Neutrals are a cool gray. One accent. Status colors are muted so a board of them stays calm.

| Token | Light | Dark | Use |
|---|---|---|---|
| `bg` | `#FFFFFF` | `#1C1C1E` | window content |
| `bg.sidebar` | material (vibrancy) | material (vibrancy) | sidebar |
| `bg.subtle` | `#F5F5F7` | `#232326` | the input at rest, finished cards, the view control's well |
| `bg.tray` | `#F8F8FA` | `#202023` | the Agents column of a project, a tray half as dark as `bg.subtle` |
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
| `badge` | 11.5 / 16 | 600 | the `P0` / `P1` mark |

M4c made type one step larger everywhere (2026-10-15): the first version read small on a retina
display. Views keep using the tokens; no literal sizes.

### Space, shape, motion

- 4 pt grid. Common gaps 4, 8, 12, 16, 24.
- Radii: 6 (pills, inputs), 9 (rows), 10 (board cards), 12 (composer, From your agents container, popovers).
- Shadows only on floating things: drag ghost, popovers, the window.
- Motion 160 ms ease-out for hover, selection and drawer; 220 ms spring for a card
  appearing and a card settling after a drop. Respect Reduce Motion: fade only.

## Glyphs

18 pt (1.5 pt a pixel), drawn on a **12 x 12 pixel grid** like the avatars (`drawStatus`, grids in
`app/theme/glyph.go`, copied from `docs/design/mockups/v6.html`): stepped pixel circles, the check
and the bang two pixels thick. Shape carries meaning; color repeats it. `StatusGlyphSized` draws
them at 12 (the blocked mark of a card) or 24 as well, so every pixel stays on the device grid.

| Status | Glyph |
|---|---|
| backlog | dashed pixel ring (two pixels of every three) |
| todo | empty circle |
| doing | ring with the right half filled |
| blocked | filled disc with a white `!` |
| review | ring with an inner dot (waiting on someone) |
| done | filled disc with a white check |
| cancelled | ring with a one-pixel slash, tertiary color |

Priority: a small outlined text badge, `P0` in `status.blocked` (1 pt ring at 35 %) and `P1` in
`text.secondary` (`border` ring), 5 pt radius, `badge` type. **Only P0 and P1 are drawn**, on rows
and cards, before the due label or the age; P2 is the default and P3 means "whenever", so both
show nothing and only deliberate priorities draw the eye. There is no bars glyph any more: the
inspector and the task menu say all four levels as text (`P1  High`, the strip `P0 P1 P2 P3`).

Clicking the status glyph of **my own** task marks it done at once (one click, no menu); marking
done plays a 220 ms animation in **two steps** (the disc fills, then at the halfway point the check
appears; Reduce Motion goes straight to done), then the item moves to Done. Clicking the glyph of an **agent's** task, or of any finished task, opens the **Task menu**
(below), so an agent's work is never completed by accident and a finished task can be reopened.

## Avatars and icons (M5)

Reference: `docs/design/mockups/v4.html` (frames 2 to 4) and `v6.html` (M6b, one pixel style).
The avatars and empty-state art are sprites in `app/theme/pixel.go` (12 x 12 pixels each, drawn as
rectangles, so they stay sharp at any size).

**Interface icons** are pixelarticons (MIT, `pixelarticons@2.4.2`; a 24-unit viewBox drawn in
2-unit pixels, so also a 12 x 12 pixel grid), vendored under `app/theme/pixel-icons/` with their
LICENSE and a README line, embedded with `go:embed` and tinted with the color of the surrounding
text (`fill="currentColor"`, no strokes). Provider logos (lobe-icons) keep their own colors. Icons
are drawn only at whole multiples of the pixel grid: `theme.IconSmall` 12 pt (meta lines, chevrons,
checks, clear buttons, search fields, the plus of the section header), `IconRow` 18 pt (sidebar,
menus, inspector properties, view toggle, calendar and menu back buttons, the close button, send,
the composer's plus), `IconBar` 24 pt (the Settings toolbar), `IconLarge` 36 pt (spare). The
`Icon*` variables map to pixelarticons names:

| Variable | pixelarticons | Variable | pixelarticons |
|---|---|---|---|
| `IconInbox` | `inbox` | `IconBoard` | `grid-3x2` |
| `IconCalendar` | `calendar` | `IconList` | `bulletlist` |
| `IconDone` | `checkbox-on` | `IconBack` / `IconForward` | `chevron-left` / `chevron-right` |
| `IconFolder` | `folder` | `IconChevron` | `chevron-down` |
| `IconGear` | `gear` | `IconCheck` | `check` |
| `IconSearch` | `search` | `IconClose` | `close` |
| `IconBell` | `bell` | `IconPlus` | `plus` |
| `IconFlag` / `IconMilestone` | `flag` | `IconSend` | `arrow-up` |
| `IconMove` | `arrow-right` | `IconAgent` | `robot` |
| `IconCopy` | `copy` | `IconStatus` | `invert` |
| `IconTrash` | `trash` | `IconDiamond` | `diamond-gem` |
| `IconClock` | `clock` | `IconPR` | `git-pull-request` |
| `IconBranch` | `git-branch` | `IconArrowOut` | `external-link` |
| `IconTerminal` / `IconAgents` | `terminal` | `IconSparkle` / `IconStar` | `sparkle` |
| `IconOctocat` / `IconGitHub` | `github` | `IconGeneral` | `sliders` |
| `IconNow` | `circle` | | |

- **Who wrote a task.** `Source == user` is `theme.You`, the pixel person (a task the assistant wrote
  for me counts as mine). `Source == agent` is `theme.AgentAvatar(name)`, `name` being the agent of
  the task's latest report (`agent` when it has none): Claude Code, Codex, Cursor and Gemini CLI have
  their own, any other agent a robot tinted by its name. The assistant is `theme.Assistant`, a
  purple ghost with a crown.
- **List rows** (a project, the Inbox, the Done group): an 18 pt avatar under the group's name (the
  rows sit 26 pt in from the header, so the header reads as their parent), vertically centered on
  the row; the title is as far from the avatar as the header's name from its glyph (the avatar's
  slot gives back the difference between the row's 12 pt gap and the header's 8 pt). Rows that carry the
  status glyph (Mine's Today and Later, Done today, Completed, search) keep the glyph: it is the way to complete or reopen.
- **Board cards**: a 12 pt avatar first in the meta line, before the blocked mark, the badge, the
  branch and the age. The line is always drawn.
- **Inspector**: a **From** row right under Status: the 18 pt avatar and the agent's name as people
  write it (`Claude Code`, `Codex`, `Cursor`, `Gemini CLI`, `GitHub`), or `You`. Every property label
  has an 18 pt pixel icon in `text.tertiary` before it (Status `IconStatus`, From `IconAgent`, Priority
  `IconFlag`, Project `IconFolder`, Milestone `IconDiamond`, Due `IconCalendar`, Remind `IconBell`,
  Branch `IconBranch`, Pull request `IconPR`); the label and its icon share the 96 pt label column.
- **Activity**: each report starts with its agent's 18 pt avatar (GitHub's own events with its mark)
  on the thin timeline; the name is in `meta` `text.secondary`, red for the newest report when it
  is a blocked one.
- **Assistant**: the composer's mark is the assistant's 18 pt avatar (it pulses while the assistant
  works, as the sparkle did); a one-line answer that changed nothing starts with its 12 pt avatar.
  Receipts keep their green check.
- **Sidebar**: Mine is the pixel you (`theme.You`) and Agents the robot in the accent color
  (`theme.AccentBot`), both 18 pt, the size of the icons of the other rows, so the labels stay in line;
  Inbox `IconInbox`, Calendar `IconCalendar`, Completed `IconDone`, 18 pt in `text.secondary`;
  a project is `IconFolder` tinted `theme.ProjectColor(name)` (the color follows the name, the user
  does not choose it).
- **The drawn task menu** (and its Move to and reminder pages) puts an 18 pt pixel icon before each
  item: Move to `IconMove`, Remind me `IconClock`, Open PR `IconArrowOut`, Ask `IconSparkle`, the
  reminder presets `IconClock` / `IconCalendar`, No reminder `IconClose`, Custom `IconPlus`, a
  project `IconFolder` in its color (white on the highlighted row), Delete `IconTrash` in
  `status.blocked` with its text. The native context menus are text only.
- **Empty states** (Mine, Agents, Inbox, a project, Completed, search with no hits): `theme.DuoIcon` at 60 pt
  (you and the assistant side by side, 5 pt a pixel), 14 pt under it a title in `body.strong`, then one
  line in `text.secondary`, all centered, the line wrapping at 480 pt; a link when there is a way
  forward ("How to connect").

## Surfaces

One window. No tray icon, no flyout panel, no global shortcut (decided 2026-10-14: the user
found the tray pointless and shortcuts conflict). The Dock icon opens the window; closing the
window hides it; Quit quits. The high-fidelity reference for this section is
`docs/design/mockups/now.html` (rendered: `now.png`). Match it.

### Window (1100 x 720, min 860 x 540)

Unified title bar. Columns: sidebar (216 pt) | content | inspector (360 pt, only when a task
is open). Content is a centered column, max 780 pt wide, 48 pt side padding, so lists read
like a document, not a spreadsheet.

**Sidebar.** A search field first (see **Search**), then `Mine` / `我的` (a plain count: the rows
from your agents plus Today; when the user has not seen what agents handed over, a 18 pt red badge
with their number instead, `status.blocked` with white `badge` text) and `Agents` / `Agent` (count =
tasks at work, never their subtasks), `Inbox` (count), `Calendar` and `Completed` (no count: it
only grows), `⌘1` to `⌘5` in that order, then `Projects` (sentence
case, `section` style) with a quiet `+` at the right of its header: one row per project (a folder icon tinted with the project's color, name only),
**on one line, ellipsized** (`answer-me-with-ht…`); a 6 pt red dot at the right when
it has a blocked task, otherwise the count of active tasks in `text.tertiary`. A project with no active
task shows no count. Settings is a gear button at the bottom. Rows are 32 pt tall in `body` 14 pt
(the sidebar is one step smaller than the page); selected row: `bg.selected`, 7 pt radius.

- **`+` on Projects** opens the **New project** sheet (below). It is a 24 pt target in `text.tertiary`,
  `text.secondary` on hover.
- **Right-click a project**: `Open QingYunA/adhd-todo on GitHub ↗` (only with a repository: it opens
  github.com in the browser), `New milestone…` (the project's only way to its first milestone), a
  separator, `Rename`, `Archive`, a separator, `Delete…`. Rename edits the name in the
  row itself: the label becomes an input with the cursor at the end, Enter saves, Esc and a click away put
  the old name back. Archive acts at once (the project is in Settings > General > Archived projects,
  one click from coming back); Delete… asks first (**Delete confirmation**). Archiving or deleting the
  project on screen goes to Mine.
- The repository is not on the project's page: it is in this menu and in the inspector's top line.

### Mine (home)

Header: `Mine` / `我的` in `title` + the date in `text.tertiary`, nothing else. Reference:
`docs/design/mockups/v5.html` (frame 1). The person's tab is Mine, not "Yours"; no accent bar marks a
new item: the hop (below) is the signal.

**Composer.** The one input is the **Composer** (see **Composer** below), the first thing on the page.
`⌘K` focuses it from anywhere. With a model connected, Enter sends the text to the built-in
assistant (see **Assistant**). With none, Enter creates a plain task (an Inbox task on Mine) and
shows a quiet "Added to Inbox" line under the box for 1.5 s.

Then up to four sections. **A section with nothing in it is not drawn at all.** Subtasks never get
a row anywhere (see **Subtasks**).

1. **From your agents** / `Agent 交过来的` — inside one bordered container (12 pt radius, rows
   split by hairlines), the only box on the screen. What agents handed over, blocked first. Each
   row: the agent's 18 pt avatar, the title, one reason line (`reason · project name`, the project
   dropped below 600 pt), the age, and one tinted button (`bg.selected`, accent text):
   - an agent task that is blocked: `Claude Code is stuck: "<next_step or summary>"` (`卡住了`),
     button **Answer** / `回答` (goes to where the agent is, see **Going to the agent** below); with
     nothing to quote, `Claude Code is stuck and waits for you`;
   - an agent task in review: with a PR `PR #12 is ready for your review`, buttons **Open PR** and
     **Accept** / `收下`; without one `Claude Code finished and is waiting for you` (`做完了，等你试用`),
     buttons **Review** / `去看看` (goes to where the agent is, as Answer) and **Accept**. Accept sets
     the task to done (the row fades like any completion);
   - an agent task with no report for `stale_hours`: `No update for 3d`, button **Open** (the inspector);
   - an agent task whose reminder is due: `You asked to be reminded at 4:00 PM`, button **Done**;
   - a pull request closed without merging: `PR #12 was closed without merging`, buttons **Cancel
     task** and **Keep** (the row goes away for good, until the pull request changes state again);
   - a pull request whose checks are failing, or where changes were requested: **Open PR**.
   The three pull request rows come from the GitHub poller (see **GitHub** under Settings) and
   apply to any task with a pull request that is not done or cancelled (failing checks and
   requested changes only while it is open, not a draft). Order: blocked, due reminders, closed
   pull requests, failing checks, requested changes, review, stale; the longest wait first within a
   kind. A task shows once, under the first reason it has. Pull request rows age from GitHub's own
   report, not the agent's last word; GitHub's reports (agent `github`) show in Activity but never
   count as the agent speaking. **Not now.** A row can be put aside: while it is hovered or
   keyboard-selected, a 12 pt `close` icon (`text.tertiary`, `text.secondary` on hover) shows at the
   far right, after the buttons, in a 20 pt square slot that is always reserved so the row does
   not shift. Tooltip `Not now` / `先放一放`, a11y `Not now: <title>`. A click hides the row without
   opening it or changing the task (it stays in its project and on Agents); the row's `Since` is
   kept in `dismissed.json` next to the database, so it comes back when the agent reports again
   or GitHub files a new event, and entries of tasks that are gone are dropped. A hidden row is
   out of the sidebar count, the badge and the hop. A row with two buttons puts them under its summary below 560 pt of
   content, and every row does below 420 pt. Screens `mine-pull-requests-*`.
2. **Today** / `今天` — the user's tasks (a user task that is blocked or doing, too): due or
   reminded today and overdue first by time, then what is in progress. Row: the status glyph (it
   ticks the task off), title, project name, `P0`/`P1` badge, due label at the right ("2:30 PM",
   "Today").
3. **Later** / `之后` — the user's tasks due in the next 7 days by date, then undated by priority
   (at most 7; a "Show all in Inbox" link after). Later ones stay in their project or the Inbox.
4. **Done today** / `今天做完` — what was finished since local midnight, by anyone, newest first:
   the `done` glyph, the title in `text.secondary` at weight 400, and at the right who did it, then
   the time of day: `Claude Code · PR #245 merged` or `You`, then `2:05 PM`. At most five; past
   them a `Show all in Completed` link. It does not count toward the sidebar and never makes Mine
   "not clear". Who finished a task: the agent that filed a report at the moment it closed (within
   five seconds of the completion, or later), else me.

The sidebar count is the rows of From your agents plus Today. **Nothing needs you.** When those
three sections are empty (Done today may still show): the empty state under the composer,
`Nothing needs you right now` and in secondary `Your agents are at work. Whatever happens shows up
here.` (zh `现在没有要你处理的事` / `Agent 们在干活，有动静会出现在这里`), with a "How to connect" link.

Tight rows (the inspector open on a small window): the project name goes below 600 pt of content;
the branch, pull request and age below 460 pt.

**Going to the agent.** Answer and Review do not open the inspector; they take the user to where the
agent works, the way Settings > Agents says (default: **the agent's app**). The row itself still opens
the inspector, as every row does. The folder the agent works in is the `cwd` it passed to
`report_progress`, kept on each report (`reports.cwd`, migration 005; a report that passed none keeps
`''`, the server's default folder is not stored): the newest report that has one, else the project's
folder, else none. The places:
- **The agent's app**: `open -a <App>` for the desktop app that matches the agent (Claude Code
  `Claude`, Codex `Codex`, Cursor, Windsurf, Zed, Copilot `Visual Studio Code`; Gemini CLI, Qwen Code,
  Kimi CLI, opencode, Cline and unknown agents have none). It brings the app forward, and for the
  editors also opens the folder. **It cannot open the exact session**: MCP gives no session id, so
  only the app is brought forward. An agent with no app, or an app that is not installed, falls back
  to a terminal at the folder;
- **Terminal**: `open -a Terminal <folder>`;
- **Editor**: the first installed of Cursor, Visual Studio Code and Zed (found with `open -Ra`, which
  does not launch it), opened on the folder;
- **Finder**: the folder.
Terminal, Editor and Finder need a folder. When opening fails, one quiet line in `meta` and the blocked
red shows under the row's reason line (`Couldn't open it: …`, `Nothing to open: the agent did not
say where it works.`, `No editor found. Install Cursor, Visual Studio Code or Zed.`); it goes away
with the next successful open or when the row is opened.

### Agents

Header: `Agents` / `Agent` in `title` + `3 working` / `3 个在做` in `text.tertiary` (nothing when
none). The composer's placeholder is `Ask what your agents are doing` / `问问 Agent 们在做什么`.
Frame 2 of v5.

1. **Working** / `在做` — agent tasks in doing or blocked, newest report first, across all
   projects. A row: the agent's 18 pt avatar, the title, one meta line `Claude Code · main · what it
   said; next: …` (agent, branch when the row is wide enough, summary), and at the right the live
   dot (last report under 10 minutes old) or, for a blocked task, a `Blocked` pill, then the age.
   Its subtasks hang under it on a 1 pt `border` rail that starts at the middle of the avatar:
   12 pt avatar, title in `body`, a status pill and the age. Pills are 13 pt `meta` labels on a
   12 % tint of the status color: `Done` / `完成` (green), `Doing` / `进行中` (blue), `Handed back`
   / `交回主代理` (grey, an agent in review: the main agent reviews it, not the user), other
   statuses by their own name. Cancelled subtasks are left out. A click on a subtask opens it; on
   the parent, the parent.
2. **Done today** / `今天做完` — agent tasks finished since midnight, newest first, five at most:
   the title in `text.secondary`, `3 subtasks` / `3 个子任务` when it had any, and the time. Their
   subtasks are not rows.

Nothing at work and nothing finished: the empty state `No agents at work` / `没有 Agent 在做事`
with `When an agent reports its progress, it shows up here.` and the "How to connect" link.

### Subtasks

A subtask is a task with a parent (`Task.ParentID`): the work a subagent reports under the task of
the agent that started it. **A subtask never gets a row of its own** in a project's list, the board,
the Inbox, Mine, Agents, Completed, the calendar or the sidebar counts; it shows only under its
parent on Agents, and in two other places:

- **Search** finds subtasks like any task and puts the parent's title where a result keeps its
  project name (`Port the MCP tools  Port core to Go`).
- **The inspector** of a parent lists its subtasks under the properties (header `Subtasks` /
  `子任务`, 12 pt avatar, title, status pill; every one, cancelled included; a click opens it). The
  inspector of a subtask has a `Parent` / `父任务` row (`IconBranch`) with the parent's title as an
  accent link that opens it.

### The hop

When an agent hands something over and the user is not looking at Mine, the agent's 24 pt sprite
jumps from the Agents row to the Mine row (frame 3 of v5): it rises from the avatar of the Agents
row, arcs (the top is 30 pt above the Mine row), lands left of Mine's badge, squashes (30 % wider and
shorter, feet on the row) and settles; the badge counts up as it lands. 0.9 s (648 ms in the air).
Several items hop 120 ms apart, three at most at once; the others count at once. The motion is
`theme.HopAt`, a function of the time since take-off and of the two rows' rectangles (the rows'
`Bounds()` of the frame before), drawn in a `ui.Overlay` from the app's clock, **only while a hop
plays**: a frame is asked for every 16 ms meanwhile and none when it is over or when nothing hops
(`hopper.asked`, tested with the fake clock). With **Reduce Motion** nothing moves: the badge counts
at once and fades in over 160 ms.

"New" means a row of From your agents the user has not seen: shown with its hand-over time (the
latest report; the due time of a reminder; GitHub's report for a pull request) later than the one
recorded. **Seen** is recorded per task id with that time, in `seen.json` in the data folder
(`data.FileSeen`; the demo keeps it in memory), whenever Mine is shown, and entries of tasks that no
longer exist are dropped. A row handed over again (a newer report) is new again. A launch that finds
rows recorded as seen does not hop for them.

### Composer

One component, five places: the top of **Mine**, of **Agents**, the top of **every project** (under the header, above
the list or the board), the top of the **Inbox**, and the bottom of the **inspector**. It replaces the
old ask field and the inline "+ Add a task" inputs; the Completed page and the search results have
none. Screens `composer-*`, `project-composer-*`, `inspector-composer-*`. It is **one line**: the
page's title already says where a task goes, so the input has no label, no chip and no hint text.

- **At rest.** One line, 50 pt tall, 12 pt radius, `bg.subtle`, **no border, no shadow**; a 16 pt
  assistant avatar (18 pt) at the left (it pulses while the assistant works); the placeholder in `input` (17 / 25)
  `text.tertiary`; `⌘K` in `meta` `text.tertiary` at the right (not in the inspector's). The
  placeholder is short: Mine `Tell me anything — I'll turn it into tasks`, Agents `Ask what your agents are doing`; a project and the Inbox
  `Add a task, or ask me anything`; the inspector `Change anything about this task`.
- **Focused or with text.** The background becomes `bg`, with a 1 pt `accent` border at 50 % and a 3 pt
  ring at 10 %. The text area grows with its text, 1 to 6 lines, then it scrolls; the avatar and `⌘K` give
  way to a 28 pt round **send** button at the right, bottom-aligned (`accent` with text, absent while
  the field is empty; neutral while the assistant is still working on the last request). Without a
  model the box keeps its 50 pt from rest to one line of text; with one, the second row below adds
  its height once there is text. Focusing it moves nothing either way. Empty and focused it
  shows only the cursor and the placeholder (and a `+`, below). No `↵ to send` or `⇧↵ new line` text: the button is the
  hint.
- **Second row (M6, with a model connected).** Once the box holds text or an image, a 28 pt row shows
  under the text: `+` (18 pt plus icon, `text.secondary`, round hover well) at the left, then the
  **model chip**, then the send button, which moves down into this row. Focusing an *empty* box still
  moves nothing: it shows only the cursor and the placeholder, with the `+` in the avatar's place so
  images can be added at once. (A box that grows when it is focused would shift the page under a
  press that blurs it, and the click would be lost; the second row appears with text, and stays
  while there is text, focused or not.) Without a model there is no second row, no `+` and no chip.
- **Model chip.** 24 pt tall, 7 pt radius, `bg.subtle`: the provider's logo (16), `Provider · model`
  in `meta` `text.secondary`, a 12 pt chevron; ellipsized when the box is narrow (the inspector's).
  Hidden when no model is connected. A click opens a drawn **popover** (a native menu cannot show
  logos), 270 pt wide, hanging from the chip's right edge (above it where there is no room below):
  the heading `Switch model` in `meta` `text.tertiary`, then one 32 pt row per provider that is set up
  (logo, name in `compact`, its model in `meta` `text.tertiary` at the right), the one in use on
  `bg.selected` with an accent check. Choosing a row makes it the provider in use, exactly as Settings >
  Assistant does, closes the popover and puts the cursor back in the field. `Esc` or a press outside
  closes it. The logo is drawn by one function, `providerMark` in `composer_model.go`.
- **Images.** Pasted with `⌘V` (the pasteboard holds an image or image files), dropped on the box (it
  takes an accent ring and a faint accent tint while files are dragged over it), or chosen with `+`
  (the system's file panel). A paste of text is untouched. Each image shows as a 64 x 48 thumbnail
  (8 pt radius, hairline border, cover-fit) in a row above the text, with an 18 pt round `x` at its
  top-right corner that removes it. At most 4; a fifth, a file that is not a PNG, JPEG, GIF or
  WebP image (a HEIC photo, a PDF), one still over 5 MB after shrinking, or an unreadable file
  shows a quiet `compact` `text.secondary` line under the box for 4 s and is not taken. Images are
  shrunk to at most 1600 px on the long side and re-encoded (a JPEG stays a JPEG, the rest PNG; a
  noisy PNG over 5 MB goes out as JPEG), off the UI thread. They are sent with the text (`↵` or the
  send button; images alone can be sent too) and live in memory only: never in the database, never on
  disk. A turn that fails keeps its images for **Retry**, and a stopped turn puts them back in the box.
- **A model that cannot read images.** The provider's refusal is classified (`providers.KindNoImages`)
  and the error line says `<model> cannot read images.` with the model chip open, to pick another;
  **Retry** then sends the same text and images to it.
- **Keys.** `↵` sends, `⇧↵` breaks the line, `Esc` cancels a running request, else closes the answer,
  else leaves the field. `↓` on a page walks into the rows. `↵` on an empty field opens the highlighted
  row. `⌘Z` in an empty composer undoes the assistant's last change (see **Assistant**).
- **No model connected.** The same field; `↵` adds a plain task in the place of the page (a project,
  the Inbox; Mine → Inbox) and a quiet `✓ Added to Inbox` line shows 1.5 s under the field. The only
  extra is the line `Set up the assistant` (a link to Settings > Assistant) under the field while it
  is focused and empty. The inspector's composer is not drawn without a model.
- **Scope.** The composer tells the assistant where it is. On Mine and Agents: nothing (new tasks go to the
  Inbox unless the words name a project). In a project: that project, so new tasks go there unless
  the words name another. In the Inbox: the Inbox. In the inspector: that task, and “this”, “it”
  mean it.
- **Answer.** What comes back shows as one line directly under the box that sent it (above the
  inspector's, which sits at the bottom). Behavior as in **Assistant**.

### Assistant (M3, restrained in M4d)

The composer is the entry to a small built-in agent (Eino, `internal/agent`). It acts at once and
reports what it did with an Undo; it asks only when an action would be wrong without the answer.
Screens: `now-thinking`, `now-reply-receipt`, `now-reply-receipts`, `now-reply-sentence`,
`now-question`, `now-no-model` (light and dark), `now-composer-*`.

- **Working.** The avatar pulses (plain with Reduce Motion) and the empty field's
  placeholder reads `Thinking…`. One request at a time: Enter while it works keeps what is typed.
  **Esc cancels** the request (its context is cancelled); with nothing changed yet the text returns
  to the field.
- **The answer is never an echo of the user.** The user's own words are not shown again, and there
  is no divider and no bubble. What shows depends on the turn, always left-aligned with the
  avatar of the box (18 pt) and 8 pt under it:
  - *It changed something*: **one line per task**, 34 pt tall: a green ✓ (an accent bell for a
    reminder), the verb in `text.secondary`, the title in `text` weight 500, the rest in
    `text.secondary`, and **Undo** at the right in `accent` (`meta`, tinted on hover):
    `✓ Added Fix the login redirect loop · Fri · P1   Undo`. The verb is `Added` for a new task,
    `Marked done`, and none for an edit (`✓ Team meeting · Moved to adhd-todo`). What follows the
    title is what the words made of it: a deadline as its day (`Fri`; a time only when one was said,
    `Fri 3:00 PM`; `18:00` is the assistant's default end of day and reads as the day alone), a
    reminder as `Tomorrow 9:00 AM · reminder`, `P1` when the priority is not the default, and the
    project or `Inbox` only when it is not the page the words were typed on (on Mine, a task in the
    Inbox says nothing). Up to **three** lines; more receipts end in `and 1 more` (`meta`-quiet, no
    Undo). Clicking a line opens the task. After Undo the title is struck through in `text.tertiary`
    and the line says `Undone`. In the inspector the same lines sit above its box and leave the
    title out when it is the task shown there.
  - *It changed nothing* but said something: one grey line, the assistant's 12 pt avatar and
    its sentence in `text.secondary`.
  - *A question*: the question in `body`, its 2 to 4 options as small tinted buttons below (the
    style of the From your agents buttons), wrapping when they must. A click sends the option as the
    answer; typing a free one works too (the placeholder says `Type an answer, or pick one below`).
    A question **stays until it is answered**; once answered its buttons are gone.
  - *An error*: neutral words ("Couldn't reach the model. Is the address right and the server
    running?") with **Retry** as a link. No red. It stays.
- **It goes by itself.** The line of a turn that asked and failed nothing shows **6 s**, the last 400 ms
  a fade (opacity only, so Reduce Motion changes nothing), by the app clock; Undo restarts the 6 s.
  It goes at once when the user types in the composer it came from. What is under it slides up
  rather than jumping (not with Reduce Motion). `Esc` closes it at once; a further `Esc` with
  nothing else to close makes the assistant forget the conversation (it forgets after 10 minutes
  idle as well).
- **⌘Z** undoes the assistant's last change that is not undone, newest first, while its line is shown
  or the conversation is live (until the second `Esc`, or the assistant idling out); with the line
  gone it comes back for a moment saying `Undone`. In a composer with text typed, `⌘Z` is the
  field's own. (The task menu's Delete keeps its own Undo toast.)
- **No model connected.** See **Composer**.
- Times in receipts follow the Mine screen's words: `Today`, `Tomorrow`, a weekday, then a date,
  followed by the clock (`9:00 AM`) when there is one to say.

### Project

**Header.** One line: the project's name in `title`, and at the right a quiet `List | Board` control:
two icons (list ☰, board ▥) in a `bg.subtle` well, 30 × 26 pt cells, 6 pt radius, the chosen one
raised on `bg` with a 1 pt shadow; the others in `text.tertiary`. They have no text; the tooltips say
`List` and `Board`, and the accessibility labels `List view` and `Board view` (List is the default).
Nothing else is in the header: **no repository** (it is in the sidebar's project menu and the
inspector's top line), **no `Done 5 · Backlog 2`** (finished tasks are in the list), and **no
milestone menu until the project has a milestone**.

**Milestone filter.** With at least one milestone, quiet `meta` text with a small chevron sits before the
view control: `All milestones ⌄`, or the milestone being filtered by (`v0.2 native ⌄`). Its menu:

1. `All milestones`, a separator, then one item per milestone (checked when it filters the page),
   as before: `v0.2 · due Oct 28`, `v0.1 · closed`;
2. a separator and `New milestone…`;
3. a separator and one submenu per milestone, `Edit “v0.2”`, holding `Rename…`, `Due date…` and
   `Close milestone` (`Reopen milestone` for a closed one).

`New milestone…`, `Rename…` and `Due date…` open the **Milestone sheet**. Closing and reopening act
at once. A project with no milestone has no filter: `New milestone…` is in its sidebar menu.

**List view: two columns** (the Inbox keeps the single list, **Inbox list view** below; the board is unchanged). A project
with at least one task is drawn as **Mine** on the left (tasks with `Source user`) and **Agents** on the
right (`Source agent`) under the header and the composer, as in `docs/design/mockups/v7.html` and
`v7b.html` (screens `project-two-columns-*`, `project-list-*`). With no task at all the project keeps the
centred empty state.

- **Block.** Header, composer and columns share one block as wide as the content allows less the page's
  side padding, up to `theme.BoardMax` (980 pt), centered. Two equal columns 28 pt apart
  (`theme.ColumnsGap`). Below **640 pt of content** (the inspector open on a small window) they stack:
  Mine first, the Agents tray under it, 14 pt apart.
- **Column header.** A 20 pt avatar (`you`; the robot in the accent color for Agents), the name in
  `body.strong` (`Mine` / `我的`, `Agents` / `Agent`), the count of its active tasks (not Done, not
  Backlog) in `text.tertiary`, and a 1 pt line under them.
- **The boundary is a tray** (option 2 of v7b): the Agents column sits on a faint well, `bg.tray`
  (`#F8F8FA` light, `#202023` dark; about half as dark as `bg.subtle`), 14 pt radius, 16 pt inside on the
  left and right, 14 pt under its last row. Mine has no background, and there is no vertical rule.
- **Groups.** Each column groups its own tasks as the list did: Blocked, In review, Doing, To do (empty
  groups are not drawn), then its own **Done** (the newest three and `Show all 5 →`) and its own
  `Backlog 2 — show` line. The two columns open and close their Done and Backlog separately. Rows sit
  26 pt in from their header, as in the Inbox: an Agents row's avatar under the header's name, and the
  titles of both columns a step further in (a Mine row keeps the avatar's room empty); `Show all N →`
  lines up with the titles.
- **Agents column, first group `Waiting for an agent` / `等 Agent 接手`**: tasks the user handed over
  (`HandedTo` set) that no agent has reported on yet. Its glyph is the pixel parcel (10 x 10, drawn
  at 20 pt, in `theme/pixel.go`). A row: the same accent robot as the column header (the task is any
  agent's, so no agent's own sprite), the title, the age at the right, and under it how it is picked
  up, `Tell any agent “do #44” to pick it up` / `在任意 Agent 里说「做 #44」就会接手` (the task's id).
  An agent's first report moves it to Doing.
- **Mine rows**: no avatar (the column says whose), the title in `body.strong` (one line), and at the
  right a `P0` or `P1` badge, then the due label. Finished rows: the title in `text.secondary` and the day.
- **Agents rows**: the agent's 18 pt avatar, the title (one line) with the live dot and the age or stale
  label at the right, and under it what the agent last said, **up to two lines** in `text.secondary`,
  led by `#12 · ` when the task has a pull request (a blocked task says what it waits for). **No branch**:
  the inspector has it. No agent name either: the avatar says it.
- **An empty column** keeps its header and one quiet `text.tertiary` line. Mine: `Nothing for you here
  — write one above` / `这里还没有你要做的事 —— 在上面写一句就行`. Agents: `No agent has reported in this
  project yet` / `还没有 Agent 在这个项目里汇报` and the accent link `Install MCP →` / `安装 MCP →`, which
  opens Settings > Agents. Screens `project-empty-agents-*`.
- **Keyboard order**: every Mine row, then every Agents row (waiting rows first).

#### Handing a task to an agent

Drag a row of Mine onto the Agents tray (screens `handoff-drag-*`, `handoff-over-*`,
`handoff-landed-*`). **Nothing is started and nothing is copied**: handing a task over only moves it to
`Waiting for an agent`, marked `handed_to = "agent"` (any agent, `domain.AnyAgent`); an agent finds it
when told its number, through the MCP.

- **Lift and carry.** The row drags like a board card: it lifts (a shadow, a 2 degree tilt, about 250 pt
  wide) and leaves an empty dashed slot. Only unfinished tasks of the user's drag. The user's `you`
  sprite (24 pt) carries it: it stands under the left end of the lifted row, its head tucked 3 pt
  under the card as if holding it up, and goes wherever the pointer takes the row. mygo paints the
  dragged row itself as a copy of its element tree at 75 % opacity, so the sprite is a child of the row
  that exists only while it is dragged. While the pointer moves the sprite walks: a bob of up to 2 pt,
  four steps a second, from the app clock; it stands still 120 ms after the pointer stops. Frames are
  asked for every 16 ms only while it walks.
- **Over the tray.** The tray gets a 7 % accent wash and a 1.5 pt accent ring at 35 %, and a first line in
  accent reads `Drop to hand to an agent` / `松开，交给 Agent` (no agent is named: a hand-off goes to
  any agent). The robot in the tray's header hops up and reaches: one short
  hop (`theme.HopAt` from and to the same spot, 10 pt high, scaled to the header's 20 pt), again every
  0.8 s while the row is over the tray.
- **Drop.** The task becomes the agent's, To do, in `Waiting for an agent`, with no travelling in
  between. The header robot squashes once as it catches it (the landing of the hop, 250 ms), and the
  new row keeps an 8 % accent wash that fades over 2 s. A toast says `Handed to an agent` /
  `已交给 Agent`, with **Undo** for six seconds, and so does ⌘Z (the assistant's own ⌘Z wins while
  it has a change to undo): the task is taken back with its old status.
- **Take back.** Dragging an agent's unfinished row onto Mine calls `Store.TakeBack`, with no sprite;
  the toast says `Yours now` / `现在归你了`, and Undo (or ⌘Z) restores Source and the hand-off as they
  were (`Store.RestoreHandOff`).
- **Reduce Motion.** The sprite still carries the row, but does not walk; the agent neither hops nor
  squashes. The wash fades as above.

**Inbox list view** below; the board is unchanged). A project
with at least one task is drawn as **Mine** on the left (tasks with `Source user`) and **Agents** on the
right (`Source agent`) under the header and the composer, as in `docs/design/mockups/v7.html` and
`v7b.html` (screens `project-two-columns-*`, `project-list-*`). With no task at all the project keeps the
centred empty state.

- **Block.** Header, composer and columns share one block as wide as the content allows less the page's
  side padding, up to `theme.BoardMax` (980 pt), centered. Two equal columns 28 pt apart
  (`theme.ColumnsGap`). Below **640 pt of content** (the inspector open on a small window) they stack:
  Mine first, the Agents tray under it, 14 pt apart.
- **Column header.** A 20 pt avatar (`you`; the robot in the accent color for Agents), the name in
  `body.strong` (`Mine` / `我的`, `Agents` / `Agent`), the count of its active tasks (not Done, not
  Backlog) in `text.tertiary`, and a 1 pt line under them.
- **The boundary is a tray** (option 2 of v7b): the Agents column sits on a faint well, `bg.tray`
  (`#F8F8FA` light, `#202023` dark; about half as dark as `bg.subtle`), 14 pt radius, 16 pt inside on the
  left and right, 14 pt under its last row. Mine has no background, and there is no vertical rule.
- **Groups.** Each column groups its own tasks as the list did: Blocked, In review, Doing, To do (empty
  groups are not drawn), then its own **Done** (the newest three and `Show all 5 →`) and its own
  `Backlog 2 — show` line. The two columns open and close their Done and Backlog separately. Rows sit
  26 pt in from their header, as in the Inbox: an Agents row's avatar under the header's name, and the
  titles of both columns a step further in (a Mine row keeps the avatar's room empty); `Show all N →`
  lines up with the titles.
- **Agents column, first group `Waiting for an agent` / `等 Agent 接手`**: tasks the user handed over
  (`HandedTo` set) that no agent has reported on yet. Its glyph is the pixel parcel (10 x 10, drawn
  at 20 pt, in `theme/pixel.go`). A row: the same accent robot as the column header (the task is any
  agent's, so no agent's own sprite), the title, the age at the right, and under it how it is picked
  up, `Tell any agent “do #44” to pick it up` / `在任意 Agent 里说「做 #44」就会接手` (the task's id).
  An agent's first report moves it to Doing.
- **Mine rows**: no avatar (the column says whose), the title in `body.strong` (one line), and at the
  right a `P0` or `P1` badge, then the due label. Finished rows: the title in `text.secondary` and the day.
- **Agents rows**: the agent's 18 pt avatar, the title (one line) with the live dot and the age or stale
  label at the right, and under it what the agent last said, **up to two lines** in `text.secondary`,
  led by `#12 · ` when the task has a pull request (a blocked task says what it waits for). **No branch**:
  the inspector has it. No agent name either: the avatar says it.
- **An empty column** keeps its header and one quiet `text.tertiary` line. Mine: `Nothing for you here
  — write one above` / `这里还没有你要做的事 —— 在上面写一句就行`. Agents: `No agent has reported in this
  project yet` / `还没有 Agent 在这个项目里汇报` and the accent link `Install MCP →` / `安装 MCP →`, which
  opens Settings > Agents. Screens `project-empty-agents-*`.
- **Keyboard order**: every Mine row, then every Agents row (waiting rows first).

#### Handing a task to an agent

Drag a row of Mine onto the Agents tray (screens `handoff-*`, storyboard in v7b).

- **Lift.** The row drags like a board card: it lifts (a shadow, a 2 degree tilt, about 250 pt wide) and
  leaves an empty dashed slot. Only unfinished tasks of the user's drag. While it is over the tray, the
  tray gets a 7 % accent wash and a 1.5 pt accent ring at 35 %, and a first line in accent reads `Drop to
  hand to Claude Code` / `松开，交给 Claude Code` (the agent that last reported in the project; the store
  picks the real one).
- **Drop.** The view words a prompt in the app's language (`Please do task #44 in adhd-todo: <title>`,
  the notes when there are any, and how to call `report_progress`) and calls `Store.HandOff`. The task
  becomes the agent's, To do, in `Waiting for an agent`. When the store started the agent in Terminal,
  a toast says `Handed to Claude Code — opened in Terminal` / `已交给 Claude Code，已在终端里打开`;
  when it could not, the prompt is copied and the toast says `Prompt copied — paste it to any agent` /
  `已复制提示词，粘贴给任意 Agent`. The toast has **Undo** for six seconds, and so does ⌘Z (the
  assistant's own ⌘Z wins while it has a change to undo): the task is taken back with its old status. It
  cannot close the Terminal window.
- **Take back.** Dragging an agent's unfinished row onto Mine calls `Store.TakeBack`, with no animation;
  the toast says `Yours now` / `现在归你了`, and Undo (or ⌘Z) restores Source and the hand-off as they
  were (`Store.RestoreHandOff`), starting nothing.
- **The motion** (`theme.HandOffAt`, about 1.3 s, a function of the app's clock like the hop; frames are
  asked for every 16 ms only while it plays). The row folds into the parcel (0 to 250 ms); the user's
  sprite hops down from the Mine header to catch it (100 to 400 ms), carries it to the gap between the
  columns (520 to 820 ms) while the agent's sprite hops from the Agents header to meet it (540 to
  820 ms); the parcel changes hands and the agent squashes as it catches (820 to 930 ms); both hop home
  (960 to 1200 ms) and the parcel opens into the new row (1000 to 1200 ms), which keeps an 8 %
  accent wash that fades over 2 s. While the user's sprite is away its header avatar is not drawn. The
  hops are `theme.HopAt` stretched to each leg, so the arc, the landing squash and the settle are the
  hop's. **Reduce Motion**: nothing hops; the row fades out where it was and in on the right (160 ms),
  and the wash fades as above. Leaving the screen ends the motion.

**Inbox list view** (one column; a project's list before M7b). Groups in this order: Blocked, In review, Doing, To do. Group header: glyph +
name (`body.strong`) + count in `text.tertiary`. Rows sit 26 pt in, their avatar under the
header's name (no glyph per row; the group says the status); `Show all N →` under Done lines up
with the rows' titles. Row: the title (one line, `body.strong`); for an agent's task a
quiet second line in `text.secondary`, `Claude Code · feat/native-ui · #12 · what it last said` (the
agent's name, the branch, the pull request, then the report; for a blocked task what it waits for);
at the right a `P0` or `P1` badge when it has one (see **Glyphs**), then the due label, or the age,
or the stale label. There is no branch pill beside the title any more. **Empty groups are not drawn.**
The composer sits above the groups (see **Composer**): it is how a task is added here.
After To do comes **Done**, a group like the others and **expanded**: glyph + `Done` + count,
its rows newest first in `text.secondary`, with who finished them and the day at the right (`Claude
Code  Yesterday`, `You  Oct 12`), three rows and then a `Show all 4 →` link in `accent` that expands
the group in place. Last line: `Backlog 2 — show` in `text.tertiary` (folded: Backlog stays out of
the way); clicking it opens the group. With no backlog the line is not drawn.

**Board view.** Four fixed lanes, left to right: **To do, Doing, In review, Done**. A blocked task sits
in Doing, **first**, with the red blocked glyph at the start of its meta line. Backlog and Cancelled
tasks are not on the board (the list shows them). Done shows its **newest 5** and, past them, `Show
all 12` in `accent`, which opens the lane in place. Screens `project-board-*`.

- **Lanes have no background.** A lane is its header (status glyph 18 pt, name in `body.strong`,
  count in `text.tertiary`, 32 pt tall) and its cards 8 pt apart; lanes are 20 pt apart. While a card
  is dragged over a lane it gets a faint accent wash (7 %) reaching 8 pt past the cards. An empty lane
  says what it is for in `meta` `text.tertiary`: `Drag here to start` (Doing), `Nothing to review` (In
  review), `Nothing yet` (To do, Done).
- **Cards**: `bg` with a 10 pt radius, a 1 pt `border` ring and a 1 pt soft shadow; 12 pt padding, 4 pt
  between the two lines. The title in `body.strong` (up to three lines), then **one meta line**
  (13 / 18): the blocked mark (12 pt) for a blocked card, the `P0` / `P1` badge, the branch in
  `mono` (ellipsized), and at the right the due label, the age or the stale label. A card with
  nothing to say has no meta line. **No status glyph on a card**: its lane says it (so the way to finish
  your own task is the task menu, `Space`, or dragging it to Done). Done cards are flat `bg.subtle`
  with no ring, the title in `text.secondary` weight 400.
- **No `+` in a lane header**: the input above adds tasks. Dragging a card to another lane changes its
  status as before (a blocked card dropped on Doing stays blocked); there are no drop targets for
  hidden statuses.
- **Edges.** The header, the composer and the four lanes share their left and right edges: one
  centered block as wide as the window allows less the page's side padding (48 pt), but no wider than
  980 pt. When even 176 pt lanes do not fit (the inspector open on a small window), the block fills
  the room and the lanes scroll sideways.

Decisions made while building it (2026-10-14):

- **One word per status**, everywhere (list groups, board lanes, the inspector, the menu strips
  and their accessibility labels, receipts, Chinese too): Backlog, To do, Doing, Blocked, In review,
  Done, Cancelled. The stale label is `3d stale` in the list and on a card, with no `!`.
- **Review without a PR** still goes to From your agents (`Claude Code finished and is waiting for
  you`, action Review). A review task of my own does not: it waits on someone else.
- **Later** shows dated tasks due within 7 days; later ones appear in their project or the Inbox only.
- **Reminders** stay in Today (an agent task's in From your agents) until marked done; the notification fires once, the row does not
  go away.
- **Tight rows** (the inspector open on a small window): Mine's rows drop the project name below 600 pt of
  content, and the branch, pull request and age below 460 pt. From your agents rows with two buttons put
  them under the summary below 560 pt, and every such row does below 420 pt (the 860 pt window
  with the inspector open leaves 282 pt), so the title keeps its width. A finished row drops who
  finished it below 460 pt and what they said below 600 pt.
- A notification click opens the task in the inspector over Mine.

### Inbox

One column of groups (Blocked, In review, Doing, To do, Done, Backlog), as a project's list was before
M7b: an agent's row has the agent's name, branch, pull request and what it said in a quiet second line.
No view control and no subtitle under the title; tasks with no project.

### Inspector (360 pt, slides in 160 ms)

Top line (the bar with the close button, `meta` `text.secondary`): where the task lives, `adhd-todo ·
QingYunA/adhd-todo · #7` (the project, its repository when it has one, the task's number; `Inbox ·
#7` for an Inbox task). A long name is ellipsized before the number. Under it: large editable title; property list (Status, **From**, Priority, **Project**,
Milestone, Due, Remind, Branch, Pull request) with left labels, each with its line icon, in `meta` secondary and values
editable in place; Notes; `ACTIVITY` timeline newest first. Save on blur or Enter. Esc closes.

- The **Status**, **Priority**, **Project** and **Remind** values open the pieces of the **Task
  menu** (below) under the value, not a system menu: the status strip, the priority strip, the
  `Move to…` list (`Inbox` and the projects, the current one checked) and the reminder presets.
  Moving a task to another project clears its milestone (milestones belong to one project); the
  Milestone row follows the project.
- The body (title, properties, notes, activity, delete) is its own scroll area that takes the height
  the pinned stack leaves; the **answer and composer** are pinned at the bottom under a hairline and
  never overlap it. The answer is the one line of **Assistant**, its left edge on the properties' (24 pt),
  leaving out the title the pane already shows; the composer is the same quiet box (see **Composer**,
  without `⌘K`, placeholder `Change anything about this task`). `Ask about this task…` (⌘J) focuses it.
  Values the assistant set in the last 10 s carry a small accent `✦` after the value. Priority reads as
  text (`P1  High`), with no bars glyph.
- Below ACTIVITY, a quiet `Delete task…` in `text.secondary` (no red, no box). It deletes
  at once like the menu's Delete (with the Undo toast) and closes the pane.

### Task menu

Our own popover, not the system's, so it can show the status glyphs and priority as one-click
strips. The same menu on every surface that shows the task (list rows, Mine rows, search and
Completed rows, board cards), opened by:

- a **right-click** on a row or card (at the pointer);
- a click on the **status glyph** of an agent's task, or of a finished task (at the glyph);
- **S**, **P**, **M**, **R** on the selected row (below the row), and `⌘J`, `⌘⌫` (see Keyboard).

It is 268 pt wide, 12 pt radius, opaque `bg` (the mockup's blur is not available in mygo; at 96 % the
rows behind showed through), a 1 pt `border` and a soft shadow; it moves into the window, or above
the point it hangs from, when it would leave it. The Move to… page, which shrinks as the filter
narrows it, is placed once by the room its whole list needs (below the point, else above it, else
as low as the window allows) so it does not jump while typing. `Esc` or a press outside closes it (the press goes on to what is
under it, so a right-click on another row moves the menu there). Reference: frame 3 of
`docs/design/mockups/v2.html`.

Main page, top to bottom (10 pt padding):

1. `Status` label in `section` style and a **strip** of the status glyphs: backlog, to do, doing,
   blocked, review, done, cancelled. One click sets the status and closes the menu. Hover
   tints a button (`bg.hover`); the current status is tinted `bg.selected`. There is no tooltip (it
   covered the label): the label row names the choice, `Status · Doing` (the hovered or
   keyboard-highlighted one, else the current one; the name in `text.tertiary`).
   (The mockup draws six, without cancelled; the seventh keeps cancelling a task one click away.)
2. `Priority` and a strip `P0 P1 P2 P3` (P0 in `status.blocked`, the current one tinted); its label
   reads `Priority · P1 High` the same way.
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
7. `Open in…` (`IconTerminal`), only for an agent's task: replaced by a page titled `Open in…` with a
   back chevron and four choices, `The agent's app`, `Terminal`, `Editor`, `Finder` (see **Going to
   the agent** under Mine). A click opens that place once and closes the menu; it does not change the
   Settings choice. A choice with nowhere to go (no folder, or no app and no folder) is drawn in
   `text.tertiary`, with no hover, and does nothing;
8. `Ask about this task…` (`⌘J`): opens the inspector with its composer focused;
9. a hairline and `Delete` (`⌘⌫`) in `status.blocked`, the only red.

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
newest task in them. Group header: `#` mark (a tray for the Inbox), project name in `body.strong`, count in `text.tertiary`.
Row: status glyph (a button, like on Mine), title, branch pill and `#PR`, and at the right the age of
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
subtitle `Done and cancelled, newest first`. Mine's `Show all in Completed` and a list's
`Show all` lead the eye here. Day header in `body.strong` with the count in `text.tertiary`: `Today`, `Yesterday`,
then a date (`Oct 12`, `Oct 12, 2025` in another year), written by `app/words`. Row: the status glyph
(done check or cancelled slash, a button: choosing another status reopens the task), title, project
name, and at the right who finished it (`Claude Code`, `You`) and the time of day.

- **Empty**: `Nothing finished yet` and `Tasks you mark done or cancel collect here, newest first.`
- 100 tasks are listed; `Show older` at the end loads 100 more.
- The same keys as any list: `↑ ↓ ↵ Space Esc`, `⌘3` opens it.

### Calendar

The `Calendar` sidebar item (`日历`, `IconCalendar`), between Inbox and Completed; `⌘4` opens it.
Reference: `v4.html` frame 1, screens `calendar-*.png`. A month of tasks by the day that matters.

- **Header**: the month as the page title (`October 2026`, `2026 年 10 月`, written by `app/words`),
  and at the right a quiet segmented control `‹ Today ›` in the well of the list and board toggle
  (`bg.subtle`, 8 pt radius, 26 pt segments), Today raised. `‹ ›` show the previous and next month and
  move the selected day along (the same day number, the last day of a shorter month); Today returns
  to this month and selects today. Under it the composer, then the grid.
- **Grid**: 7 columns, the weeks the month needs (four to six), weeks starting on the system's first
  weekday (Monday for zh-Hans, Sunday for en when the system names none). Weekday names in `meta`,
  `text.tertiary`, over the grid. Cells are separated by 1 pt hairlines (`border`) inside an
  `RadiusCard` frame; Saturday and Sunday on a faint fill (`CalendarWeekend`); days of other months
  in `text.tertiary`; the first of a month reads `Nov 1` / `11 月 1 日`. Today's number sits in an
  accent pill (`accent`, `accent.text`); the selected day has a 2 pt ring in `accent` at 60 %. The
  grid is at most `CalendarMax` (1240 pt) wide and takes the window's remaining height, never less
  than `CalendarMinCell` (76 pt) a cell; below that the page scrolls.
- **Items** (`meta`, 20 pt, 2 pt apart, ellipsized): a task starts with its author's 12 pt avatar
  (`theme.You` for what I wrote, the assistant included; `theme.AgentAvatar(name)` for what an agent
  wrote, by its latest report, `agent` when it has none). A task *due* that day shows its title; a
  *reminder* shows its time in `text.tertiary` first (a small clock instead in a cell too narrow for
  both, and nothing in a narrower one); a task *finished* that day shows its title in `text.tertiary`
  struck through; a *milestone* due that day is an accent-tinted chip with `IconFlag`. A task due and
  reminded the same day shows once, as the reminder. An open task whose due time has passed has its
  title in the blocked red. Cancelled tasks and closed milestones are not shown. Order within a day:
  milestones, open items by time, finished items.
- **More**: a cell lists at most three items; when there are more (or fewer lines fit the cell's
  height) the rest is `2 more` / `还有 2 项` in `text.tertiary` on the last line. It is a link: pointer
  cursor, `text.secondary` and underlined on hover, a11y `Show all 16 on Oct 15` / `显示 10 月 15 日的全部
  16 项`. A click opens the **day card** and takes the click alone (the cell's own click does not fire).
- **Add**: a hovered cell shows a 12 pt `IconPlus` (`text.tertiary`, `text.secondary` on a `bg.hover`
  20 pt square when hovered) at the right of the day number, floating over the cell so nothing
  moves; a11y and tooltip `Add to Oct 15` / `添加到 10 月 15 日`. A click opens the day card with its
  input focused. A **double click** on a cell does the same (the first click selects the day).
- **Day card** (`app/views/calendar_day.go`): one day in full, over the grid in an overlay. Its top
  left sits on the cell's top left, so it covers the cell and grows outward; near the right or bottom
  edge of the window it grows left or up instead and stays 8 pt inside the window. Width
  `max(1.6 × cell, 280 pt)` on the 4 pt grid; height follows its content up to 420 pt, then the list
  scrolls. A panel like the sheets: `bg`, `RadiusPanel`, 1 pt `border`, the sheets' shadow, 12 pt
  padding. Top to bottom: the header (`Thursday, Oct 15` / `10 月 15 日 星期四` in `body.strong`, the
  count `5 items` / `5 项` in `text.tertiary`, a 24 pt `IconClose` button at the right); every item of
  the day, 28 pt rows drawn like the cell's items (avatar, a reminder's time, the title; finished items
  struck through, milestones as the accent chip), a click opens the task in the inspector and closes
  the card; at the foot an input in the sheets' well (`Add to this day…` / `添加到这天…`). Enter adds a
  plain task by me, due that day at 18:00, clears the input and keeps the card open, the new item
  joining the list; empty text does nothing. **Closing**: `Esc`, a click outside (it does nothing
  else: it does not select the day under it), the close button, a change of month or `Today`, and
  leaving the page. While the card is open the window's shortcuts wait, so the arrows do not move the
  day. **Opening** is four pixel steps, no tween (`theme.DayCardAt`): 55 %, 70 %, 85 %, then full size
  from 150 ms, done at 200 ms, scaling the card's width and height from the corner it grows from and
  clipping the contents; opacity is 1 throughout. A frame is asked for every 16 ms only while it
  plays; with Reduce Motion the card is full size at once. Closing has no animation. Screens
  `calendar-day-card-{light,dark,zh-light}`.
- **Selecting**: a click on a day selects it (a click on a grey day of another month keeps the
  month). The composer's placeholder becomes `Add to Fri, Oct 16, or ask me anything` / `添加到 10 月
  16 日周五，或者问我任何事`. The assistant's scope carries the day (`PlaceCalendar`, `agent.ScopeCalendar`):
  a task with no time words is due that day at 18:00 local, and a day named in the words wins. With
  no model connected a plain task is added to the Inbox with that day at 18:00 as its due date.
- **Items**: a click opens the task in the inspector (it does not select the day); a right click opens
  the task menu. Drag an item onto another day: a due item moves its due date there and a reminder
  item its reminder, both keeping their time of day; the target day tints with the accent while the
  item is over it. Finished items do not drag.
- **Keyboard** (no text input focused): `← →` the previous and next day, `↑ ↓` the same day a week
  earlier or later (leaving the grid turns the page), `⌘←` `⌘→` the previous and next month,
  `⌘4` opens the page, `⌘K` the composer. Items have no keyboard focus of their own: the task menu
  and `↵` act on the highlighted task of other pages only.
- **Data**: the snapshot holds every task including all finished ones, so no range query is needed;
  `data.BuildCalendar` places them (see `app/data/calendar.go`).

### Calendar

The `Calendar` sidebar item (`日历`, `IconCalendar`), between Inbox and Completed; `⌘4` opens it.
Reference: `v4.html` frame 1, screens `calendar-*.png`. A month of tasks by the day that matters.

- **Header**: the month as the page title (`October 2026`, `2026 年 10 月`, written by `app/words`),
  and at the right a quiet segmented control `‹ Today ›` in the well of the list and board toggle
  (`bg.subtle`, 8 pt radius, 26 pt segments), Today raised. `‹ ›` show the previous and next month and
  move the selected day along (the same day number, the last day of a shorter month); Today returns
  to this month and selects today. Under it the composer, then the grid.
- **Grid**: 7 columns, the weeks the month needs (four to six), weeks starting on the system's first
  weekday (Monday for zh-Hans, Sunday for en when the system names none). Weekday names in `meta`,
  `text.tertiary`, over the grid. Cells are separated by 1 pt hairlines (`border`) inside an
  `RadiusCard` frame; Saturday and Sunday on a faint fill (`CalendarWeekend`); days of other months
  in `text.tertiary`; the first of a month reads `Nov 1` / `11 月 1 日`. Today's number sits in an
  accent pill (`accent`, `accent.text`); the selected day has a 2 pt ring in `accent` at 60 %. The
  grid is at most `CalendarMax` (1240 pt) wide and takes the window's remaining height, never less
  than `CalendarMinCell` (76 pt) a cell; below that the page scrolls.
- **Items** (`meta`, 20 pt, 2 pt apart, ellipsized): a task starts with its author's 12 pt avatar
  (`theme.You` for what I wrote, the assistant included; `theme.AgentAvatar(name)` for what an agent
  wrote, by its latest report, `agent` when it has none). A task *due* that day shows its title; a
  *reminder* shows its time in `text.tertiary` first (a small clock instead in a cell too narrow for
  both, and nothing in a narrower one); a task *finished* that day shows its title in `text.tertiary`
  struck through; a *milestone* due that day is an accent-tinted chip with `IconFlag`. A task due and
  reminded the same day shows once, as the reminder. An open task whose due time has passed has its
  title in the blocked red. Cancelled tasks and closed milestones are not shown. Order within a day:
  milestones, open items by time, finished items.
- **More**: a cell lists at most three items; when there are more (or fewer lines fit the cell's
  height) the rest is `2 more` / `还有 2 项` in `text.tertiary` on the last line.
- **Selecting**: a click on a day selects it (a click on a grey day of another month keeps the
  month). The composer's placeholder becomes `Add to Fri, Oct 16, or ask me anything` / `添加到 10 月
  16 日周五，或者问我任何事`. The assistant's scope carries the day (`PlaceCalendar`, `agent.ScopeCalendar`):
  a task with no time words is due that day at 18:00 local, and a day named in the words wins. With
  no model connected a plain task is added to the Inbox with that day at 18:00 as its due date.
- **Items**: a click opens the task in the inspector (it does not select the day); a right click opens
  the task menu. Drag an item onto another day: a due item moves its due date there and a reminder
  item its reminder, both keeping their time of day; the target day tints with the accent while the
  item is over it. Finished items do not drag.
- **Keyboard** (no text input focused): `← →` the previous and next day, `↑ ↓` the same day a week
  earlier or later (leaving the grid turns the page), `⌘←` `⌘→` the previous and next month,
  `⌘4` opens the page, `⌘K` the composer. Items have no keyboard focus of their own: the task menu
  and `↵` act on the highlighted task of other pages only.
- **Data**: the snapshot holds every task including all finished ones, so no range query is needed;
  `data.BuildCalendar` places them (see `app/data/calendar.go`).

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

`⌘K` the page's composer · `⌘Z` the assistant's last change (see **Assistant**) · `⌘,` Settings · `⌘F` search · `↑ ↓` move selection through rows on the current screen · `↵` open the
selected row in the inspector · `Space` toggles done for your own tasks · `Esc` closes the
inspector, then clears the selection, then the search · `⌘1` Mine, `⌘2` Agents, `⌘3` Inbox, `⌘4` Calendar, `⌘5` Completed.

With a row selected: `S` status, `P` priority, `M` move to, `R` remind: each opens the Task menu on
that piece, where `←` `→` or `↑` `↓` move, `↵` chooses and `Esc` closes; `⌘J` asks the assistant
about the task (opens the inspector with its composer focused); `⌘⌫` deletes it, with Undo.

While a text input has the keyboard its keys are its own (`Element.OnTextKey`), and the window's
arrows, `↵`, `Space` and `Esc` stay out of the way:

| Input | `↵` | `Esc` | `↓` |
|---|---|---|---|
| Composer | sends (`⇧↵` breaks the line); on an empty field opens the highlighted row | cancels a running request, else closes the answer, else leaves the field | moves into the rows (Now); `⌘Z` on an empty field undoes the assistant's last change |
| Inspector title, branch | saves | puts the stored text back and leaves the field | |
| Inspector notes | breaks a line (saves on blur) | puts the stored text back and leaves the field | |
| Search field | opens the highlighted result (the first when none) | clears the text and leaves the field | moves into the results |
| Rename project (sidebar) | saves | puts the old name back | |
| New project, milestone sheets | creates / saves | closes the sheet | |

### Settings

A window of its own (M4c; the in-window sheet is gone). `⌘,` (the app menu's `Settings…` item) and the
sidebar gear open it; there is one instance, and a second request brings it to the front, on the pane
asked for. **One fixed size, 680 × 520 pt of content**, not resizable: a pane taller than that scrolls
under the toolbar, and a tab change never resizes the window (M5; resizing to each pane's height was
the flicker). Each pane has its own scroll key. Reference: `v4.html` frames 5 to 7, and `v5.html` frames 5 and 6 (logos, Agents); screens
`settings-general-*`, `settings-assistant-*` (at rest), `settings-assistant-new-*` (nothing set up),
`settings-change-*`, `settings-change-error-*`, `settings-custom-*`, `settings-agents-*`,
`settings-github-*`.

- **Toolbar panes**, as in System Settings: General (`IconGeneral`), Assistant (`IconStar`), Agents
  (`IconTerminal`), GitHub (`IconOctocat`), each an icon above a
  label (`meta`), 78 pt wide; the open one gets a 6 % black (white in dark) 8 pt well and an `accent` icon.
  The window title is the open pane's name. The top of the window is **one band** (the toolbar
  fill, a hairline under it): the title bar is hidden, the traffic lights sit in the band's first
  row, the open pane's name is centered there in `section` style, and the pane icons sit under it.
- **Grouped lists.** Pane background `#F5F5F7` (dark `#1C1C1E`); groups are white (dark `#2C2C2E`)
  rounded 10 pt blocks with a 0.5 pt hairline; rows are at least 44 pt, the label (`body`) and an
  optional `meta` line left, the control right; hairlines between rows; an optional group label
  above the block in `section` style (13 / 18, 600, `text.secondary`).
- **General**: Language (a pop-up: System / English / 简体中文); **Archived projects** (a row per
  archived project with a **Restore** link, or the empty line); **Data folder** (the path in `meta`, written
  from the home folder as `~/Library/Application Support/adhd-todo`; **Show in Finder** at the right
  reveals the real path); About (`adhd-todo 0.3.0`, `Open source, MIT license`, a GitHub
  link).
- **Assistant**, at rest (M5): the label **In use** and one card (a group): the provider's **logo**
  (38 pt, `theme.ProviderLogo`, see **Provider logos** below), its
  name (`body` 500), then `model · ● Connected · 312 ms` in `meta` (the model in `text.secondary`, the
  dot and words in the done green), and **Change** at the right. Before any test the line reads
  `Not tested yet` with a **Test** link; a test that failed shows its friendly error in `meta` under
  the line, in the blocked red, with **Test again**. Under the card, **Also set up**: a group with a
  row per other configured provider (a 28 pt logo, name, model, and for Ollama and Magpie their
  note), **Use** at the right; it makes that provider the one in use and tests it. A provider counts
  as set up when it is in use, has a stored key, or (Ollama, Magpie, Custom without a key) was saved
  once. With nothing set up the pane opens on the grid at once, without **Done**.
- **Assistant > Change**: the label **Pick a provider** with **Done** at the right (back to rest), and
  a grid of 12 tiles, 6 × 2, 76 pt tall, 8 pt apart: Anthropic, OpenAI, DeepSeek, Kimi, 智谱 GLM,
  通义千问, Gemini, OpenRouter, 硅基流动, Ollama, Magpie, Custom. A tile is a 32 pt logo and the name
  (`meta`); a 7 pt green dot at the top right when the provider is set up; a 2 pt `accent` ring when
  picked (a 1 pt hairline ring otherwise). The provider in use is picked when Change opens. Under the
  grid, in one group, only: **API key** (a secure field, placeholder `Paste your Kimi key`, a stored
  key shows dots and an empty field keeps it; **Get a key ↗** at the right opens the provider's key
  page), and **Model** (a pop-up as wide as the field, with **Save and test**, the accent button, at
  its right). The model pop-up lists the preset's suggestions, then the models the provider says it
  has (`GET {base}/models`, or Anthropic's `/v1/models`), up to 24, with `Other…` last, which turns it
  into a text field; the list is asked in the background when a provider with a stored key (or none
  needed) is picked, and again when a typed key is left; if it fails the suggestions stand in
  silently. **Save and test** stores the key and the model, sends a 1-token request, and on success
  makes the provider the one in use and returns to rest; on failure the friendly error shows under
  the fields (the blocked red, `meta`) and the key stays stored, so a retry is one click. Enter in the
  key or model field is the same as the button. **Custom** adds **Base URL** and **Protocol**
  (OpenAI-compatible | Anthropic API) above the key, and its model is always typed. Magpie and Ollama
  have no key row. A missing key, base URL or model is said under the fields without a request.
- **Provider logos** (M6): the official marks of lobe-icons (MIT, `@lobehub/icons-static-svg` 1.95.1),
  vendored as SVG under `app/theme/logos/` with their LICENSE and embedded in the binary, so nothing is
  fetched. `theme.ProviderLogo(c, presetID, size)` draws one `size` pt square: Anthropic (`claude-color`),
  DeepSeek, 智谱, 通义千问, Gemini, OpenRouter and 硅基流动 in their own colors (`ui.Image`); OpenAI,
  Kimi and Ollama only have one color, so they use `ui.Icon` and follow the text color (white in dark
  mode; the caller sets it with `TextColor`, Settings uses `text`); Magpie is the pixel robot in teal;
  Custom is a plus in `text.secondary`. Sizes: `LogoCard` 38, `LogoTile` 32, `LogoRow` 28, `LogoChip` 16
  (for the composer's model chip). The provider logo is never a colored tile with letters any more.
- **Agents** (M6, `v5.html` frame 6): the label **Agents on this Mac**, one plain `meta` line under
  it ("Installing the MCP adds adhd-todo's MCP server to that agent, so it reports its progress here.
  It does not install or start the agent."), then ten agents in **two columns**, each column a white
  group, read left to right, top to bottom: Claude Code, Codex, Gemini CLI, Cursor, opencode,
  Copilot (VS Code), Qwen Code, Kimi CLI, Windsurf, Zed. A row (at least 44 pt): the 24 pt pixel
  avatar (`theme.AgentAvatar`), the name (`body`), one `meta` line, and at the right one of
  - `✓ MCP installed` (done green): adhd-todo is in the agent's MCP config, or the agent has reported;
  - **Install MCP** (the accent button): the agent is on this Mac (its command is on PATH, or its
    config folder is there) and has no adhd-todo yet. While it runs the button reads `Installing…`;
  - **Get it ↗** (a link to the agent's site): the agent is not on this Mac.
  The line under the name is `N reports today` when the agent reported today, else `No reports
  today` when it ever reported, else `Found · gemini` / `Found · ~/.cursor` (what was found), else
  `Not on this Mac`. The agent that reported is matched by `agentName` (`claude-code` is Claude Code).
  An install that fails says why in the row, in `meta` and the blocked red, in plain words (the last
  line the agent's own command printed, "adhd-core isn't at ~/.local/bin/adhd-core", a timeout).
  A config file with comments (Zed's `settings.json`) is **never rewritten**: a `Add this to
  ~/.config/zed/settings.json:` line and the snippet in the mono well with **Copy** appear under the
  columns. Under them one group with one row, **Open agent work in** / `去看看时打开`, with the `meta`
  line `Brings the agent's app forward. It cannot open the exact session.` and a pop-up at the right:
  `The agent's app` (default) / `Terminal` / `Editor` / `Finder`. It is where Review and Answer on a
  handed-over row go (see **Going to the agent** under Mine); the choice is kept in `prefs.json` next
  to the database. Screen `settings-agents-open-with-*` (scrolled to the bottom). At the bottom, the old footer: `Or run this once in a terminal, for Claude Code` and the
  `claude mcp add` command in a mono well with **Copy**. "Install MCP" is the only word for this; it
  is never called connecting. Detection (`internal/agentsetup`) reads files and looks on PATH (plus
  `/opt/homebrew/bin`, `/usr/local/bin`, `~/.local/bin` and the like, which a Dock-launched app's PATH
  lacks) every 30 s and after an install; **nothing is installed without the button**.
  - Commands, run with the absolute `~/.local/bin/adhd-core` and a 30 s timeout: `claude mcp add --scope
    user adhd-todo -- <core> mcp`; `codex mcp add adhd-todo -- <core> mcp`; `gemini mcp add -s user
    adhd-todo <core> mcp`; `qwen mcp add --scope user adhd-todo <core> mcp`; `kimi mcp add --transport
    stdio adhd-todo -- <core> mcp`; `code --add-mcp '{"name":"adhd-todo","command":"<core>","args":["mcp"]}'`.
  - Files (copied first to `<file>.adhd-todo.bak`, key order, number text and indentation kept, created
    when missing): Cursor `~/.cursor/mcp.json` `mcpServers`; opencode `~/.config/opencode/opencode.json`
    `mcp` (`type: local`, `command: [core, "mcp"]`); Windsurf `~/.codeium/windsurf/mcp_config.json`
    `mcpServers`; Zed `~/.config/zed/settings.json` `context_servers`.
  - Installed is read from `~/.claude.json`, `~/.codex/config.toml` (`[mcp_servers.adhd-todo]`),
    `~/.gemini/settings.json`, `~/.qwen/settings.json`, `~/.kimi/mcp.json`, the files above, and VS
    Code's user `mcp.json` (`servers`).
- **Avatars of agents** (M6): besides Claude Code, Codex, Cursor and Gemini CLI, a dark rounded square
  with a gray center for opencode and a dark head with white goggles and purple pupils for Copilot
  (both are lightened in dark mode); Qwen Code, Kimi CLI, Windsurf, Zed and Cline are the robot in their
  brand color. `agentName` and `theme.AgentAvatar` agree on the names agents report under
  (`Visual Studio Code` is Copilot (VS Code), `qwen-code-mcp-client` and `qwen-cli-mcp-client-*` are Qwen
  Code); any other name gets a robot colored by the name.
- **GitHub** (M4a): the pull request poller's state, from the `gh` CLI the user is already signed in
  to (the app owns no token). One line and a **Check now** link:
  - *Connected*: `Connected as QingYunA · checked 2 min ago`; while a check runs the link reads
    `Checking…` and is inert; before the first check ends the line says `Checking GitHub…`.
  - *No `gh`*: `The GitHub CLI isn't installed. Install it to follow your pull requests.` and the
    command `brew install gh` in a mono well.
  - *Signed out*: `You're not signed in to GitHub. Sign in to follow your pull requests.` and
    `gh auth login` in the mono well.
  - *Offline*: `Can't reach GitHub. Trying again soon.` Never red.
  Polling runs at launch, every 5 minutes, and when the main window comes back to the front if the
  last check is older than a minute; at most 4 `gh` calls at once. With no poller the pane says so.
- A report from an agent in an archived project's repository restores the project too.
- A test says `Connected · 640 ms` or the error in the words of **Assistant**; nothing is tested
  unless the user asks (Test, Use, Save and test).
- Keys live in the system keychain (service `adhd-todo`), never in the database, the provider file or
  the logs. The active provider and per-provider overrides are in `models.json` next to the database.

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
- [ ] Nothing repeats what is next to it: no label the title says, no hint a control shows, no glyph the group
      or lane names.
- [ ] One title line, one input, one list; the rest appears on hover, in a menu or in the inspector.
- [ ] Text is aligned to a shared left edge in each section.
- [ ] Empty, loading and error states exist and read well.
- [ ] Hit targets are at least 24 px; rows at least 32 px.
