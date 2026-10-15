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
- Radii: 6 (pills, inputs), 9 (rows), 10 (board cards), 12 (composer, Needs you container, popovers).
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

Priority: a small outlined text badge, `P0` in `status.blocked` (1 pt ring at 35 %) and `P1` in
`text.secondary` (`border` ring), 5 pt radius, `badge` type. **Only P0 and P1 are drawn**, on rows
and cards, before the due label or the age; P2 is the default and P3 means "whenever", so both
show nothing and only deliberate priorities draw the eye. There is no bars glyph any more: the
inspector and the task menu say all four levels as text (`P1  High`, the strip `P0 P1 P2 P3`).

Clicking the status glyph of **my own** task marks it done at once (one click, no menu); marking
done plays a 220 ms fill-and-check animation (a plain fill with Reduce Motion), then the item moves
to Done. Clicking the glyph of an **agent's** task, or of any finished task, opens the **Task menu**
(below), so an agent's work is never completed by accident and a finished task can be reopened.

## Avatars and icons (M5)

Reference: `docs/design/mockups/v4.html` (frames 2 to 4). Pixel art is for **avatars and empty
states** only; every other icon stays a 1.4 pt line icon in the color of its text. The sprites live
in `app/theme/pixel.go` (12 x 12 pixels each, drawn as rectangles, so they stay sharp at any size).

- **Who wrote a task.** `Source == user` is `theme.You`, the pixel person (a task the assistant wrote
  for me counts as mine). `Source == agent` is `theme.AgentAvatar(name)`, `name` being the agent of
  the task's latest report (`agent` when it has none): Claude Code, Codex, Cursor and Gemini CLI have
  their own, any other agent a robot tinted by its name. The assistant is `theme.Assistant`, a
  purple ghost with a crown.
- **List rows** (a project, the Inbox, the Done group): an 18 pt avatar in the column of the group
  glyph, vertically centered on the row; the title stays on the group's title (the avatar's slot
  gives back the difference between the row's 12 pt gap and the header's 8 pt). Rows that carry the
  status glyph (Now, Done today, Completed, search) keep the glyph: it is the way to complete or reopen.
- **Board cards**: a 12 pt avatar first in the meta line, before the blocked mark, the badge, the
  branch and the age. The line is always drawn.
- **Inspector**: a **From** row right under Status: the 18 pt avatar and the agent's name as people
  write it (`Claude Code`, `Codex`, `Cursor`, `Gemini CLI`, `GitHub`), or `You`. Every property label
  has a 14 pt line icon in `text.tertiary` before it (Status `IconStatus`, From `IconAgent`, Priority
  `IconFlag`, Project `IconFolder`, Milestone `IconDiamond`, Due `IconCalendar`, Remind `IconBell`,
  Branch `IconBranch`, Pull request `IconPR`); the label and its icon share the 96 pt label column.
- **Activity**: each report starts with its agent's 18 pt avatar (GitHub's own events with its mark)
  on the thin timeline; the name is in `meta` `text.secondary`, red for the newest report when it
  is a blocked one.
- **Assistant**: the composer's mark is the assistant's 18 pt avatar (it pulses while the assistant
  works, as the sparkle did); a one-line answer that changed nothing starts with its 12 pt avatar.
  Receipts keep their green check.
- **Sidebar**: Now `IconNow`, Inbox `IconInbox`, Completed `IconDone`, 16 pt in `text.secondary`;
  a project is `IconFolder` tinted `theme.ProjectColor(name)` (the color follows the name, the user
  does not choose it).
- **The drawn task menu** (and its Move to and reminder pages) puts a 15 pt line icon before each
  item: Move to `IconMove`, Remind me `IconClock`, Open PR `IconArrowOut`, Ask `IconSparkle`, the
  reminder presets `IconClock` / `IconCalendar`, No reminder `IconClose`, Custom `IconPlus`, a
  project `IconFolder` in its color (white on the highlighted row), Delete `IconTrash` in
  `status.blocked` with its text. The native context menus are text only.
- **Empty states** (Now, Inbox, a project, Completed, search with no hits): `theme.DuoIcon` at 60 pt
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

**Sidebar.** A search field first (see **Search**), then `Now` (count = items in Needs you + Agents
at work + Yours), `Inbox` (count) and `Completed` (no count: it only grows), then `Projects` (sentence
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
  project on screen goes to Now.
- The repository is not on the project's page: it is in this menu and in the inspector's top line.

### Now (home)

Header: `Now` in `title` + the date in `text.tertiary`, nothing else.

**Composer.** The one input is the **Composer** (see **Composer** below), the first thing on the page.
`⌘K` focuses it from anywhere. With a model connected, Enter sends the text to the built-in
assistant (see **Assistant**). With none, Enter creates a plain task (an Inbox task on Now) and
shows a quiet "Added to Inbox" line under the box for 1.5 s.

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
   never makes Now "not clear": **Nothing needs you** looks at the first three. Who finished a task:
   the agent that filed a report at the moment it closed (a report within five seconds of the
   completion, or later), else me.

**Nothing needs you.** When the first three sections are empty (Done today may still show below it): the
empty state under the composer (see **Avatars and icons**), `Nothing needs you right now` and in
secondary `Your agents are at work. Whatever happens shows up here.` (zh `现在没有要你处理的事` /
`Agent 们在干活，有动静会出现在这里`), with a "How to connect" link (opens Settings).

### Composer

One component, four places: the top of **Now**, the top of **every project** (under the header, above
the list or the board), the top of the **Inbox**, and the bottom of the **inspector**. It replaces the
old ask field and the inline "+ Add a task" inputs; the Completed page and the search results have
none. Screens `composer-*`, `project-composer-*`, `inspector-composer-*`. It is **one line**: the
page's title already says where a task goes, so the input has no label, no chip and no hint text.

- **At rest.** One line, 50 pt tall, 12 pt radius, `bg.subtle`, **no border, no shadow**; a 16 pt
  assistant avatar (18 pt) at the left (it pulses while the assistant works); the placeholder in `input` (17 / 25)
  `text.tertiary`; `⌘K` in `meta` `text.tertiary` at the right (not in the inspector's). The
  placeholder is short: Now `Tell me anything — I'll turn it into tasks`; a project and the Inbox
  `Add a task, or ask me anything`; the inspector `Change anything about this task`.
- **Focused or with text.** The background becomes `bg`, with a 1 pt `accent` border at 50 % and a 3 pt
  ring at 10 %. The text area grows with its text, 1 to 6 lines, then it scrolls; the avatar and `⌘K` give
  way to a 28 pt round **send** button at the right, bottom-aligned (`accent` with text, absent while
  the field is empty; neutral while the assistant is still working on the last request). The box
  keeps its 50 pt from rest to one line of text, so focusing it moves nothing. Empty and focused it
  shows only the cursor and the placeholder. No `↵ to send` or `⇧↵ new line` text: the button is the
  hint.
- **Keys.** `↵` sends, `⇧↵` breaks the line, `Esc` cancels a running request, else closes the answer,
  else leaves the field. `↓` on a page walks into the rows. `↵` on an empty field opens the highlighted
  row. `⌘Z` in an empty composer undoes the assistant's last change (see **Assistant**).
- **No model connected.** The same field; `↵` adds a plain task in the place of the page (a project,
  the Inbox; Now → Inbox) and a quiet `✓ Added to Inbox` line shows 1.5 s under the field. The only
  extra is the line `Set up the assistant` (a link to Settings > Assistant) under the field while it
  is focused and empty. The inspector's composer is not drawn without a model.
- **Scope.** The composer tells the assistant where it is. On Now: nothing (new tasks go to the
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
    project or `Inbox` only when it is not the page the words were typed on (on Now, a task in the
    Inbox says nothing). Up to **three** lines; more receipts end in `and 1 more` (`meta`-quiet, no
    Undo). Clicking a line opens the task. After Undo the title is struck through in `text.tertiary`
    and the line says `Undone`. In the inspector the same lines sit above its box and leave the
    title out when it is the task shown there.
  - *It changed nothing* but said something: one grey line, the assistant's 12 pt avatar and
    its sentence in `text.secondary`.
  - *A question*: the question in `body`, its 2 to 4 options as small tinted buttons below (the
    style of the Needs you actions), wrapping when they must. A click sends the option as the
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
- Times in receipts follow the Now screen's words: `Today`, `Tomorrow`, a weekday, then a date,
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

**List view.** Groups in this order: Blocked, In review, Doing, To do. Group header: glyph +
name (`body.strong`) + count in `text.tertiary`. Rows indent under the header text (no glyph
per row; the group says the status). Row: the title (one line, `body.strong`); for an agent's task a
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
  (13 / 18): the blocked mark (14 pt) for a blocked card, the `P0` / `P1` badge, the branch in
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
- **Review without a PR** still goes to Needs you (summary "Ready for your review", action Open).
  A review task of my own does not: it waits on someone else.
- **Yours** shows dated tasks due within 7 days (or overdue); later ones appear in their project or
  the Inbox only.
- **Reminders** stay in Needs you until marked done; the notification fires once, the row does not
  go away.
- **Tight rows** (the inspector open on a small window): Now's rows drop the project name below 600 pt of
  content, and the branch, pull request and age below 460 pt. Needs you rows with two buttons put
  them under the summary below 560 pt, and every Needs you row does below 420 pt (the 860 pt window
  with the inspector open leaves 282 pt), so the title keeps its width. A finished row drops who
  finished it below 460 pt and what they said below 600 pt.
- A notification click opens the task in the inspector over Now.

### Inbox

Same list view as a project, without the view control and with no subtitle under its title; tasks
with no project.

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
strips. The same menu on every surface that shows the task (list rows, Now rows, search and
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
newest task in them. Group header: `#` mark (a tray for the Inbox), project name in `body.strong`, count in `text.tertiary`.
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

`⌘K` the page's composer · `⌘Z` the assistant's last change (see **Assistant**) · `⌘,` Settings · `⌘F` search · `↑ ↓` move selection through rows on the current screen · `↵` open the
selected row in the inspector · `Space` toggles done for your own tasks · `Esc` closes the
inspector, then clears the selection, then the search · `⌘1` Now, `⌘2` Inbox, `⌘3` Completed.

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
asked for. 680 pt wide, as tall as its pane's content, not resizable. Reference: `v2.html` frame 5;
screens `settings-general-*`, `settings-assistant-*`, `settings-agents-*`, `settings-github-*`.

- **Toolbar panes**, as in System Settings: General, Assistant, Agents, GitHub, each an icon above a
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
- **Assistant**: **Provider** (a pop-up of the presets), **Model** (a pop-up of the preset's
  suggestions plus `Other…`, which turns the row into a text field; a preset without suggestions shows
  the field), **API key** (a secure field, `Kept in your Keychain` underneath, hidden for presets with a
  fixed key: Magpie, Ollama), a **status row** (`● Works · 640 ms`, or the friendly error in
  `text.secondary`, or `Not tested yet`) with a **Test** link; the group **Advanced** holds **Base
  URL**. Choosing a provider activates it; saving a key tests it at once, so Test is rarely needed.
  Models, base URLs and keys save on Enter or blur.
- **Agents**: `Connect Claude Code`, one line on what agents report, the registration command in a
  mono well with **Copy**.
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
- **Test** sends a 1-token request with what is typed, saved or not, and says `Works · 640 ms` or the
  error in the words of **Assistant** (`text.secondary`; the dot is the only color).
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
