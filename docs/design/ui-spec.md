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
7. **Pictures are pixels; text and layout are native.** See **Pixel language**.
8. **Few entries, few steps.** Each action has one place to start it; a new button must replace
   one or earn its place against the question above. Rarely used parts start folded.
9. **Hand it to an agent.** What an agent can do for me (filing, splitting, dating, organizing)
   is offered through the assistant or a hand-off, not as another form or setting.
10. **Add features with restraint.** A feature that adds a control, a setting or a screen needs a
    reason the existing ones cannot carry.

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

## Pixel language

Decided 2026-10-17 (M6b made the icons pixels, v8 the day card, v9 the busy indicator). Every new
piece of UI follows it; a mockup that breaks it is redrawn before it is built.

- **Every picture is pixel art on a 12 x 12 grid.** Status glyphs, interface icons, avatars,
  empty-state art, busy and progress indicators, the stop mark: filled squares only, no strokes,
  no curves, no anti-aliased edges. They are drawn only at whole multiples of the grid (12, 18, 24,
  36 pt) so no pixel is split.
- **No system indicators or symbol fonts.** No `ui.Spinner`, no `NSProgressIndicator`, no SF
  Symbols, no emoji, no text glyph standing in for an icon (`✦`, `✓`). A missing picture is added as
  a pixelarticon or drawn as a sprite in `app/theme`.
- **Pixel art moves in frames.** A picture that moves switches whole pixels on and off on a fixed
  beat; it does not rotate, scale or ease. Two beats (v10): a **stroke being drawn** steps once a
  60 Hz refresh, 17 ms, a few pixels a frame, so it reads as one pen stroke (the Accept stamp: 7
  frames, about 120 ms); a **loop** steps slowly, 100 ms, so it reads as a rhythm and not a flicker
  (the busy indicator). No frame is shorter than 17 ms. With Reduce Motion a stroke shows its last
  frame at once and a loop holds its first.
  A third beat (v18): something that **travels across a scene** (a thrown good, a fall, a hop)
  moves every 60 Hz refresh and snaps to whole cells, so the flight is smooth and the picture stays
  pixels; its pose, spin and other changes keep the 100 ms loop beat.
- **Text and layout stay native.** The system font, and smooth layout motion (rows sliding, a panel
  opening, a fade): 160 ms ease-out and 220 ms spring, as above. The art is pixels; the typography and
  the layout are not.
- **Category icons are 12 x 12 pictures** (M9, v24): kept in the database as 144 characters (`.`
  empty, `#` the category's own color, `a` to `l` the 12 palette colors), drawn as one path of 2-unit
  squares per color (`theme.CategoryIcon`, with `ui.Image`, which keeps the palette fills), only at
  12, 18, 24 (editor previews), 36 and 144 (the import's result) pt. The library icons of the picker are pixelarticons reduced to that grid
  (`scripts/category-icons.sh`); an icon that does not reduce cleanly carries a hand-drawn grid.
  The color swatches are 12 x 12 pixel circles drawn at 18 pt. See **Categories (M9)**.
- **Floating panels may be pixel panels.** The day card's shape (v8 B): two-step corners, a 2 pt
  solid edge, a hard 4 pt shadow with no blur.
- **Provider marks use an 18 x 18 grid** (v11): their official shapes do not
  survive 12 x 12, and the user must recognize them. They are not redrawn: each official SVG is
  sampled onto the grid by `docs/design/tools/sample_marks.py` (a cell is lit where the mark covers
  it, in the mark's own color) into `app/theme/provider_marks.go`. Drawn only at 18 pt (1 pt a pixel)
  and 36 pt (2 pt). Anthropic is the Claude Code crab of the sidebar, the same sprite in both places.
- **Harness marks are the second exception, on a 24 x 24 and an 18 x 18 grid** (v14): the official
  marks of the coding agents do not survive 12 x 12 either, and the user again has to recognize
  them. Settings > Agents draws them in the official colours, light and dark: 24 x 24 (drawn at
  24 pt) in the rows of **On this Mac**, 18 x 18 (drawn at 18 pt) in the compact rows of **More**.
  They are the lobe-icons / Iconify drawings the approved mockup embeds
  (`docs/design/mockups/v14.html`, its ICONS); `docs/design/tools/harness_marks.py` quantises each
  light and dark pixel mark to at most 24 colours and writes `app/theme/harness_marks.go`.
  `theme.HarnessMark(c, id, size)` picks the grid and the appearance, and draws rectangles like
  every other sprite (never `ui.Image`, which is anti-aliased at Retina scale). Claude Code is the
  one exception inside the exception: it keeps the app's own 12 x 12 crab, the same sprite at 24 pt,
  because it is the mascot and not the official starburst.
- **The Summary scene has its own grids** (v18): actors 24 x 24 and goods 16 x 16, so a figure can
  lift, throw and hop and a good reads at a glance; they carry a dark outline, three or four shades
  lit from the top left, and a one-cell shadow. The whole scene is drawn at 3 pt a cell. Only the
  scene uses these grids; avatars and icons stay 12 x 12. Like every character and object, they are
  files (see **Sprites are files**).

The busy indicator (`theme.PixelSpinner`, reference `docs/design/mockups/v9.html`): eight 2 x 2
blocks round the 12 x 12 grid, clockwise from the top (5,1), (8,2), (9,5), (8,8), (5,9), (2,8),
(1,5), (2,2). One block is lit at full opacity and steps clockwise every 100 ms (800 ms a turn);
the two behind it are at 60 % and 35 %, the rest at 25 %, all in the color of the text beside it.
The stop mark is a solid 6 x 6 square in the middle of the same grid (9 pt at 18 pt).

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
The avatars and empty-state art are sprites, PNG files under `app/theme/sprites/avatars/` (12 x 12
pixels each, drawn as rectangles, so they stay sharp at any size; see **Sprites are files**).

**Interface icons** are pixelarticons (MIT, `pixelarticons@2.4.2`; a 24-unit viewBox whose strokes
are mostly 2 units wide, so most sit on a 12 x 12 pixel grid; some use 1-unit coordinates and do not,
which is why the category picker reduces them, see **Categories (M9)**), vendored under `app/theme/pixel-icons/` with their
LICENSE and a README line, embedded with `go:embed` and tinted with the color of the surrounding
text (`fill="currentColor"`, no strokes). Provider logos are pixel marks too (see **Pixel language**). Icons
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
| `IconBox` | `checkbox` | `IconFlip` | `flip-horizontal-2` |
| `IconArrowLeft` | `arrow-left` | `IconArrowDown` | `arrow-down` |
| `IconMinus` | `minus` | | |
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
| `IconNow` | `circle` | `IconFile` | `file-text` |

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
  `IconFlag`, Category `IconList`, Project `IconFolder`, Milestone `IconDiamond`, Due `IconCalendar`, Remind `IconBell`,
  Branch `IconBranch`, Pull request `IconPR`); the label and its icon share the 96 pt label column.
- **Activity**: each report starts with its agent's 18 pt avatar (GitHub's own events with its mark)
  on the thin timeline; the name is in `meta` `text.secondary`, red for the newest report when it
  is a blocked one.
- **Assistant**: the composer's mark is the assistant's 18 pt avatar (it pulses while the assistant
  works, as the sparkle did); a one-line answer that changed nothing starts with its 12 pt avatar.
  Receipts keep their green check.
- **Sidebar**: Mine is the pixel you (`theme.You`) and Agents the robot in the accent color
  (`theme.AccentBot`), both 18 pt, the size of the icons of the other rows, so the labels stay in line;
  a sub-row under them sits one step in: a 12 pt slot flush with the parent's label, holding the
  category's 12 x 12 icon in the category's own color, the agent's avatar
  (`theme.AvatarSmall`), or nothing (Unsorted), then its label, so the sub-rows' labels line up
  with each other;
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

### Sprites are files

Decided 2026-10-17 (#67): the characters and objects of the app will change, so their pictures live
in files and not in Go. Every sprite that stands for someone or something is a PNG under
`app/theme/sprites/`, embedded with `go:embed`. No pixel grid of a character or an object is
written in Go. To change a picture, replace its file with one of the same name and size; nothing
else changes.

- **One pixel of the file is one cell of the grid.** A fully transparent pixel is empty; any other
  pixel is drawn in its own color, alpha kept. Draw them in any pixel editor (Aseprite, Pixelorama)
  and export PNG at 1x.
- **Where they go:**
  - `avatars/<name>.png`, 12 x 12: you, the assistant, every agent (`claude`, `codex`, `cursor`,
    `gemini`, `opencode`, `copilot`, ...) and the empty-state art (`parcel`, ...).
  - `avatars/<name>.dark.png`, optional, the same size: drawn instead in dark mode, for a picture
    that is itself dark and would vanish on a dark window.
  - `actors/<who>/<pose>.png` (24 x 24), `goods/<kind>.png` (16 x 16 cells and the shadow cell, so
    17 x 17), `props/` (the bin and its lids, the gold star) and `scenes/<sky>.png` (the 264 x 80
    backdrops: `day`, `dusk`, `night`): the pictures of the Summary scene (see **Summary**). They are
    drawn as bitmaps, not rectangle by rectangle, and the README lists each file and its size.
- **The code names files, not pixels.** Each avatar kind maps to a file name in `app/theme`. A test
  loads every file the code names and fails on a missing file, a wrong size, or a file under
  `sprites/` that nothing names.
- `app/theme/sprites/README.md` lists every file, its size and where it is drawn.
- Not files: the status glyphs, the busy indicator and the Accept stamp (shapes and frames that
  carry meaning, in Go); interface icons (pixelarticons SVGs); provider marks (sampled from the
  official SVGs); harness marks (the official marks of Settings > Agents, generated into
  `app/theme/harness_marks.go`, see **Pixel language**).

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
tasks at work, never their subtasks), each with sub-rows (below), `Inbox` (count), `Calendar`, `Completed` (no count: it
only grows) and `Summary` (no count), `⌘1` to `⌘6` in that order, then `Projects` (sentence
case, `section` style) with a quiet `+` at the right of its header: one row per project (a folder icon tinted with the project's color, name only),
**on one line, ellipsized** (`answer-me-with-ht…`), its sub-projects under it (see **Sub-projects**); a 6 pt red dot at the right when
it has a blocked task, otherwise the count of active tasks in `text.tertiary`. A project with no active
task shows no count. Settings is a gear button at the bottom. Rows are 32 pt tall in `body` 14 pt
(the sidebar is one step smaller than the page); selected row: `bg.selected`, 7 pt radius.

**Sub-rows.** Under `Mine`: every category by its position, empty ones too (Work / `工作` and Life /
`生活` first until the user moves them, then the user's own), and `Unsorted` / `未分类` last, only
while it has rows (or is the open page). Under `Agents`: one row per agent with a task at work
or finished today, named as the Agents screen names it (`claude-code` and `Claude Code` are one
`Claude Code` row), in the order of that screen; an agent with no name is `Agent`. A sub-row is a
normal 32 pt sidebar row (selection, hover) whose content is indented 28 pt (an icon and the gap),
so it reads as part of its parent. Its count is
the parent's count for that slice, in `text.tertiary`: the rows asking for attention (From your
agents plus Today) for a slice of Mine, the tasks at work for an agent. The red unseen badge stays
on `Mine` alone. A click opens the parent's screen showing only that slice: the same sections,
built from the tasks of the slice, so their counts and the "Show all" links are the slice's; the
page title is the slice's name (`Work`, `Claude Code`); the composer of a Mine slice files new tasks
in its category (see **Category**). `⌘1` and `⌘2` still go to the whole of Mine and Agents. A
subtask goes where its parent goes. The slice is `nav.cat` (a category's id, or `catUnsorted`) on
Mine and `nav.sub` (an agent's name) on Agents; a page whose category was deleted falls back to Mine.
Right-click: see **Categories (M9)**.

**Category.** A task is in one category or none (`tasks.category_id`, NULL for none; migration 012;
before M9 only `work` and `life`, migration 007).
The category it counts under is its own, else Work when its project is a repository, else none
(Unsorted): a category set by hand or by an agent always wins. The assistant classifies the tasks it
writes from the words and leaves it out when unsure or when the task goes into a repository's project.
A task typed in the composer of a category's page gets that category, with or without a model.

**Sub-projects.** A project can hang under another one (`projects.parent_id`, migration 013), as deep
as the user likes. Its row is indented 28 pt per step, so a sub-project's folder sits where its
parent's label is, and the projects under a parent follow it in name order; the row still shows the
project's own active tasks, never those of its sub-projects. A sub-project is made with `New
sub-project…` in its parent's menu (below): the **New project** sheet opens titled `New sub-project`,
with `Under adhd-todo` under the title, and the project it makes hangs under that one and opens.
Archiving a project archives its sub-projects with it, so no row is left under a project that is not
shown. Restoring it brings back the project and the sub-projects archived by that same action; what
was archived earlier stays archived, and the project above is left as it is. A sub-project whose
parent is still archived reads as a top-level project, and comes back under its parent when that one
is restored. Deleting a project leaves its sub-projects standing as top-level projects, as deleting a
task leaves its subtasks. A sub-project is made by hand only: it is a project like any other, so it
takes a repository from the sheet if the user gives it one.

- **`+` on Projects** opens the **New project** sheet (below). It is a 24 pt target in `text.tertiary`,
  `text.secondary` on hover.
- **Right-click a project**: `Open QingYunA/adhd-todo on GitHub` (only with a repository: it opens
  github.com in the browser), `New milestone…` (the project's only way to its first milestone),
  `New sub-project…` (the same sheet, with this project as the parent), a
  separator, `Rename`, `Archive`, a separator, `Delete…`. Rename edits the name in the
  row itself: the label becomes an input with the cursor at the end, Enter saves, Esc and a click away put
  the old name back. Archive acts at once, with the project's sub-projects (the project is in Settings >
  General > Archived projects, one click from coming back); Delete… asks first (**Delete confirmation**).
  Archiving or deleting the project on screen goes to Mine.
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
     the task to done and the row leaves as in **Accepting** below. When the latest report carries
     links (see **Links from agents**), **Open page** / `打开页面` comes first and opens its first link:
     **Open page**, **Open PR**, **Accept** with a pull request, **Open page**, **Accept** without;
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

**Accepting** (v10 B, `docs/design/mockups/v10.html`). The store says done at once; the row stays
for the animation, about 420 ms, and nothing jumps:

1. **Stamp, 0–120 ms.** The Accept button turns `status.done` (its tint at 16 %, the text in
   `status.done`) and the check of the `done` glyph is drawn in it before the word, 12 pt, left to
   right in 7 frames of 17 ms (two pixels more a frame; see **Pixel language**).
2. **Hold, 80 ms**, so the eye sees it.
3. **Fold, 220 ms.** The row fades (180 ms) and its height closes to nothing, ease-out cubic (the day
   card's curve); the hairline above it closes with it, the box of From your agents shrinks and the
   rows under it slide up. When it was the last row, the whole section (header and box) folds
   instead. At the same time the task opens at the top of Done today (height and opacity, 220 ms).

With Reduce Motion: the stamp shows its last frame at once, the row fades 180 ms and is then gone,
and the Done today row appears without motion. Every other completion that takes a row out of an
active list (the status glyph, `Space`, the task menu) folds the row the same way after its glyph
fill, instead of vanishing at the end of the fade.

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
  `text.tertiary`; with a model connected the **model chip** at the right; `⌘K` in `meta`
`text.tertiary` after it (not in the inspector's). The
  placeholder is short: Mine `Tell me anything — I'll turn it into tasks`, Agents `Ask what your agents are doing`; a project and the Inbox
  `Add a task, or ask me anything`; the inspector `Change anything about this task`.
- **Focused or with text.** The background becomes `bg`, with a 1 pt `accent` border at 50 % and a 3 pt
  ring at 10 %. The text area grows with its text, 1 to 6 lines, then it scrolls; the avatar and `⌘K` give
  way to a 28 pt round **send** button at the right, bottom-aligned (`accent` with text, absent while
  the field is empty; a neutral **stop** button while the assistant works on a request from this box,
  see **Assistant**). Without a
  model the box keeps its 50 pt from rest to one line of text; with one, the second row below adds
  its height once there is text. Focusing it moves nothing either way. Empty and focused it
  shows only the cursor, the placeholder, the `+` (below) and the model chip at the right. No `↵ to send` or `⇧↵ new line` text: the button is the
  hint.
- **Second row (M6, with a model connected).** While the box holds text or an image, while the
  assistant works on a request sent from it, and while that request's answer shows, a 28 pt row shows
  under the text: `+` (18 pt plus icon, `text.secondary`, round hover well) at the left, then the
  **model chip**, then the send (or stop) button, which moves down into this row. Otherwise the box
  is one line and the chip sits at the right of it (v9 B, so the model in use is always in sight).
  Focusing an *empty* box still moves nothing: it shows the cursor and the placeholder, with the `+`
  in the avatar's place so images can be added at once. (A box that grows when it is focused would
  shift the page under a press that blurs it, and the click would be lost.) Without a model there is
  no second row, no `+` and no chip.
- **Model chip.** 24 pt tall, 7 pt radius, `bg.subtle`: the provider's pixel mark (18), `Provider · model`
  in `meta` `text.secondary`, a 12 pt chevron. On a one-line box with no room for its words (the
  inspector's, and a page's under 720 pt of content) it is the logo and the chevron alone, its words in
  a tooltip, so the placeholder reads whole; the second row always has the words, ellipsized.
  On the one-line box its well is a step darker than the box's `bg.subtle` (`bg.hover` over it).
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
  (8 pt radius, hairline border, contain-fit on `bg.subtle`, so the whole picture shows and two
  similar screenshots can be told apart) in a row above the text, with an 18 pt round `x` at its
  top-right corner that removes it. At most 4; a fifth, a file that is not a PNG, JPEG, GIF or
  WebP image (a HEIC photo, a PDF), one still over 5 MB after shrinking, or an unreadable file
  shows a quiet `compact` `text.secondary` line under the box for 4 s and is not taken. Images are
  shrunk to at most 1600 px on the long side and re-encoded (a JPEG stays a JPEG, the rest PNG; a
  noisy PNG over 5 MB goes out as JPEG), off the UI thread. They are sent with the text (`↵` or the
  send button; images alone can be sent too). The model gets the shrunk copies; the tasks the turn
  creates or changes keep the images (see **Task images**). A turn that fails keeps its images for
  **Retry**, and a stopped turn puts them back in the box.
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

- **Working (v9 B).** Sending empties the field as before, so the next thought can be typed, but the
  box keeps its second row and the send button becomes a neutral **stop** button (the pixel stop
  mark; a click is `Esc`). The answer line under the box shows the pixel busy indicator (see **Pixel
  language**) in the avatar's place, `Thinking` and `· 3s` (`meta`; the seconds in `text.tertiary`,
  counted from the send), and `Esc to cancel` in `text.tertiary` at the right. After 15 s the words
  read `Taking longer than usual`. The answer replaces this line in place, so from the send to the
  answer nothing under the box moves. The box's avatar, when the box is at rest, still pulses (plain
  with Reduce Motion). One request at a time: Enter while it works keeps what is typed.
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
  lines up with the titles. While the pointer is over an open row of either column, the row shows its own
  status glyph under the header's glyph, as in the Inbox.
- **Agents column, first group `Waiting for an agent` / `等 Agent 接手`**: tasks the user handed over
  (`HandedTo` set) that no agent has reported on yet. Its glyph is the pixel parcel (`avatars/parcel.png`, 12 x 12
  with a pixel of room all round, drawn at 24 pt). A row: the same accent robot as the column header (the task is any
  agent's, so no agent's own sprite), the title, the age at the right, and under it how it is picked
  up, `Tell any agent “do #44” to pick it up` / `在任意 Agent 里说「做 #44」就会接手` (the task's id).
  An agent's first report moves it to Doing.
- **Mine rows**: no avatar (the column says whose), the title in `body.strong` (one line), and at the
  right a `P0` or `P1` badge, then the due label. Finished rows: the title in `text.secondary` and the day.
- **Agents rows**: the agent's 18 pt avatar, the title (one line) with the live dot and the age or stale
  label at the right, and under it what the agent last said, **up to two lines** in `text.secondary`,
  led by `#12 · ` when the task has a pull request (a blocked task says what it waits for). **No branch**:
  the inspector has it. No agent name either: the avatar says it.
- **Acting on an Agents row (v10).** While a row is hovered or keyboard-selected it takes `bg.hover`
  and the age at the right gives way to the same tinted buttons as the row's From your agents row
  on Mine: in review, **Open PR** + **Accept** (with a pull request) or **Review** + **Accept**
  (without); blocked, **Answer**. Other rows show no buttons (a click opens the inspector, as
  before). While the row's task menu is open the menu has the choices and the row shows its age. Accept leaves as in **Accepting** (Mine): the row folds out of In review (the group's
  header folds with it when it was the last) and opens at the top of the column's Done. The same
  holds in a project's single list and in the Inbox list.
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
- **Taken.** An agent takes a task by reporting on it with its number; the first report moves it out
  of `Waiting for an agent`. Reading it with `get_task` only tells the agent so, and changes nothing.
  One piece of work that covers several handed tasks reports on each (the MCP instructions say so).
  A task finished by any other way (by hand, a merged pull request) leaves `Waiting for an agent` too:
  done or cancelled clears the hand-off.
- **Reduce Motion.** The sprite still carries the row, but does not walk; the agent neither hops nor
  squashes. The wash fades as above.

**Inbox list view** (one column; a project's list before M7b). Groups in this order: Blocked, In review, Doing, To do. Group header: glyph +
name (`body.strong`) + count in `text.tertiary`. Rows sit 26 pt in, their avatar under the
header's name (the group says the status, so a row has no glyph at rest; while the pointer is over
it or in the glyph column beside it, a row shows its own status glyph under the header's glyph, a button that takes no room: clicking
it marks my own task done, and opens the task menu for an agent's task, as `glyphButton` does
elsewhere; it stays while the row completes, and select mode shows none); `Show all N →` under Done lines up
with the rows' titles. Row: the title (one line, `body.strong`); for an agent's task a
quiet second line in `text.secondary`, `Claude Code · feat/native-ui · #12 · what it last said` (the
agent's name, the branch, the pull request, then the report; for a blocked task what it waits for;
a report that only repeats the title, as on a task made by its first report, is left out);
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
- **Acting on a card (v10).** A hovered or keyboard-selected agent card in In review (or blocked in
  Doing) shows the same buttons as its Agents row in place of the age at the right of its meta
  line. Accept stamps the button as in **Accepting**, then the card leaves with the board's exit
  (`cardMotion`: it shrinks and folds) and enters at the top of Done.
  In a lane narrower than 200 pt two buttons do not fit beside the meta line: they take a line of
  their own under it.
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
#7` for an Inbox task). A long name is ellipsized before the number. Under it: large editable title; property list (Status, **From**, Priority, **Category**, **Project**,
Milestone, Due, Remind, Branch, Pull request) with left labels, each with its line icon, in `meta` secondary and values
editable in place (a Branch longer than the pane is ellipsized like every long value, and a click turns it into the editable text); the task's **images** (see **Task images**); Notes; `ACTIVITY` timeline newest first. Save on blur or Enter. Esc closes.

- The **Status**, **Priority**, **Category**, **Project** and **Remind** values open the pieces of the
  **Task menu** (below) under the value, not a system menu: the status strip, the priority strip, the
  Category page (a filter field and the categories; the row shows only the category that was set,
  after its 18 pt icon in the category's color, and `None` with no icon in `text.tertiary`, for a
  task in a repository's project too), the `Move to…` list (`Inbox` and the projects, the current one checked) and the reminder presets.
  Moving a task to another project clears its milestone (milestones belong to one project); the
  Milestone row follows the project.
- The body (title, properties, notes, activity, delete) is its own scroll area that takes the height
  the pinned stack leaves; the **answer and composer** are pinned at the bottom under a hairline and
  never overlap it. The answer is the one line of **Assistant**, its left edge on the properties' (24 pt),
  leaving out the title the pane already shows; the composer is the same quiet box (see **Composer**,
  without `⌘K`, placeholder `Change anything about this task`). `Ask about this task…` (⌘J) focuses it.
  Values the assistant set in the last 10 s carry a small accent mark after the value: the pixel `sparkle` (`IconSparkle`, 12 pt) on a `bg.selected` pill. Priority reads as
  text (`P1  High`), with no bars glyph.
- Below ACTIVITY, a quiet `Delete task…` in `text.secondary` (no red, no box). It deletes
  at once like the menu's Delete (with the Undo toast) and closes the pane.

### Task images

v13 (`docs/design/mockups/v13.html`): an image is part of a task, so the user can look at it again.

- **From the composer.** The images sent with a request are attached to the tasks the turn creates or
  changes. The assistant's task tools take the indexes of the images that belong to a task; a task the
  model gives none for gets every image of the turn (attach too many rather than lose one). The receipt
  line says so: `Added 2 tasks, with their images` (`已添加 2 个任务，图片也挂上了`). A request from the
  inspector's composer attaches its images to that task.
- **In the inspector.** Between the properties and Notes: tiles of 96 x 72 pt, three a row 12 pt apart,
  8 pt radius, hairline, the whole picture on `bg.subtle` (contain), and a last dashed tile with a
  pixel `+` that opens the file panel. With no image, one quiet line `Add images…` (`添加图片…`) in
  `text.tertiary`, as `Add notes…`. Image files dropped on the inspector are attached (the same types
  and limits as the composer's). A hovered tile shows an 18 pt round pixel `x` at its top right that
  detaches the image, with the Undo toast; a click on the tile opens the viewer.
- **In lists.** A row of a task with images (Mine, a project's columns and list, the Inbox, search,
  Completed) shows its first image as a 32 x 24 pt thumbnail (4 pt radius, hairline, contain on
  `bg.subtle`) just before the due label or the age; with more than one, a `+2` badge (9 pt, white on
  55 % black) at its bottom right. A row without images is unchanged.
- **On board cards.** The first image as a band across the top of the card, 96 pt tall, contain on
  `bg.subtle`, a hairline under it, the `+2` badge at its bottom right when there are more.
- **The viewer.** A click on any thumbnail opens the image over the window: the window dims (90 %
  near-black), the image fits the window whole, a bottom bar says `name · 1 / 2 · 1200 × 900` and
  offers **Open in Preview** (`在预览中打开` and the external-link icon, the file in macOS Preview). Round pixel `chevron-left`
  / `chevron-right` and `close` buttons; `←` `→` step through the task's images, `Esc` or a click
  outside closes.
- **Storage.** The original file is kept (up to 20 MB; a larger one keeps the composer's shrunk copy)
  under the data folder in `images/`, named by the hash of its content so one picture is stored once,
  with a 256 px thumbnail beside it; lists, cards and tiles read only thumbnails. A table records
  which task has which image (its name, size and when it was added). Deleting a task or detaching an
  image removes the link; a file no task links to any more is deleted. The model still gets the
  1600 px copy.
- **Screens.** `inspector-images`, `inspector-images-none`, `inspector-images-hover`, `mine-pictures`,
  `project-list-pictures`, `project-board-pictures`, `viewer`.
- **Not yet.** An agent attaching a screenshot through the MCP; images in an export.

### Links from agents

v19 (`docs/design/mockups/v19.html`, #76). An agent can attach pages to a report: an HTML report it
wrote, a design draft, a document. The app shows them as links and opens them in the default
browser; it never renders, downloads or caches them.

- **MCP.** `report_progress` takes `links`: up to 5 of `{url, title}`, `title` optional (cut to 80
  characters, ending in `…`). Only `https://`, `http://` and `file://` with an absolute path are kept. A
  bad link never fails the report: the report is filed, the bad link is dropped, and the result
  lists it under `dropped_links` with the reason. `get_task` returns each report's links. The
  server instructions add one sentence: when you made something that can be opened (a report page,
  a design draft), attach it with `links`.
- **Stored on the report** (migration 011, `reports.links`, a JSON array), not on the task: the
  timeline says which step made it.
- **Inspector, Activity.** Under a report's text, one chip per link, wrapping: the 12 pt pixel file
  icon in accent and the title (`meta`, ellipsized), on `bg.subtle`, 26 pt tall, 7 pt radius.
  Without a title: the file name for `file://`, the host and last path segment for `https://`
  (`github.com › pull/19`). Hover: an accent wash, the text in accent, and a tip with the full
  address (`~/…` for a file in the home folder). A `file://` link whose file is gone (checked when
  drawn) is drawn in `text.tertiary`; clicking it shows the tip `File not found` /
  `找不到这个文件` and nothing else.
- **From your agents** and the Agents rows and board cards that act like them: **Open page** as
  above, only for a task in review (a blocked task waits for an answer, not a look). Tight rows
  move the buttons as for two.
- **Summary export**: a task line ends with `[title](url)` for each link of its reports.
- **Screens.** `inspector-links`, `inspector-links-hover`, `inspector-links-missing`, `mine-open-page`,
  `project-board-open-page`.

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
3. One row, `Category` with the 18 pt icon of the task's category in its color (`IconList` when it
   has none), its name as the hint at the right and a `›`: it opens the **Category page**, which is
   the `Move to…` page for categories (v24 frame 2): a filter field `Find a category`, a row per
   category with its icon (the current one checked), `None` (unsorts), a hairline and `New
   category…`, which opens the category sheet and gives the new category to the task. The row takes
   no letter key;
4. a hairline;
5. `Move to…` (key hint `M`): the menu's body is replaced by a filter field and a list of the
   projects and `Inbox` (the current one checked). Typing filters, `↑ ↓` move, `↵` moves the task,
   `Esc` goes back to the main page;
6. `Remind me…` (`R`): replaced by presets, each with the moment it means at the right (the time
   alone for today and tomorrow, `Mon 9:00 AM` for the others): `Later today` (6:00 PM, only while
   that is over half an hour away), `Tomorrow` (9:00 AM), `Next Monday` (9:00 AM, left out on a
   Sunday, when it is tomorrow), `This weekend` (Saturday 10:00 AM, not from Saturday 9:30 or on
   Sunday), `No reminder`, and `Custom…`, which shows a calendar with a time of day, Clear and
   `Set reminder`;
7. `Open PR`, only when the task has a pull request;
8. `Open in…` (`IconTerminal`), only for an agent's task: replaced by a page titled `Open in…` with a
   back chevron and four choices, `The agent's app`, `Terminal`, `Editor`, `Finder` (see **Going to
   the agent** under Mine). A click opens that place once and closes the menu; it does not change the
   Settings choice. A choice with nowhere to go (no folder, or no app and no folder) is drawn in
   `text.tertiary`, with no hover, and does nothing;
9. `Ask about this task…` (`⌘J`): opens the inspector with its composer focused;
10. a hairline and `Delete` (`⌘⌫`) in `status.blocked`, the only red.

**Delete does not ask.** The task goes and a toast says `Deleted “title”` with **Undo** for six
seconds; Undo puts the task back with its id, fields and reports. (A project keeps its confirmation: it
holds many tasks.) The inspector's values open the same pieces alone: Status shows the status strip,
Priority the priority strip, Category the Category page, Project the `Move to…` list, Remind the presets.

A project's menu is in the Sidebar and stays a system menu.

### Selecting several tasks

v20 (`docs/design/mockups/v20.html`, #75). Every task list can hold several tasks at once and act on
them together: Mine, a project (list, two columns, board cards), the Inbox, Agents, Completed and
search results. Calendar and Summary do not select.

- **Select button.** At the right of the page title, `Select` / `选择` (the pixel checkbox icon,
  quiet like the title row's other buttons) enters select mode: each row's status glyph becomes a
  pixel checkbox, a click ticks or unticks the row instead of opening the inspector, and the title
  row reads `2 selected` / `已选 2 件` · `Select all` / `全选` · `Done` / `完成`. `Done`, `Esc`, or
  finishing a bulk action leaves select mode. The first time the user ⌘-clicks in a list, the
  Select button flashes once (two 100 ms frames), so the button is found.
- **Without the mode**: `⌘`-click adds or removes a row, `⇧`-click selects from the last clicked row
  to this one, `⇧↑ ⇧↓` extend from the keyboard row, `⌘A` selects every row of the page. A plain
  click opens the inspector and makes that row the only one selected, as before. `Esc` clears the
  selection first, then does what it did.
- **Look.** Selected rows take `bg.selected`; neighbours join into one block (no rounded corners
  between them). The keyboard row also has a 1.5 pt accent ring at 55 %. Selected rows are read as
  "selected".
- **The bulk bar.** With 2 selected (1 in select mode), a pixel panel floats at the bottom centre of
  the list (the day card's shape: two-step corners, a 2 pt solid edge, a hard 4 pt shadow, no blur):
  `3 selected` / `3 件`, **Done** (`⌘↵`), **Move to…** (a project or the Inbox), **Priority** (P0 to
  P3), **Hand to an agent** (`⌘⇧A`), **Delete** (in the blocked red, after a confirmation `Delete 3
  tasks?` / `删除 3 件任务？`), and `×` that clears the selection. When the window is narrow the
  buttons drop their words, keeping their icons, the count and `×`. It is a toolbar: Tab reaches it.
- **Each action is one change with one Undo** (`⌘Z`, or the toast's **Undo**). Done and cancelled
  rows can be selected (to delete or reopen them); **Hand to an agent** takes only the user's own
  unfinished tasks and says how many it skipped (`Handed 2 to an agent · 1 skipped`). Handed from
  Mine or the Inbox, each task waits in its own project's `Waiting for an agent` (an Inbox task
  waits in the Inbox's Agents).
- **Task menu.** Right-clicking a selected row opens the menu for all of them: a header `3 tasks` /
  `3 件任务` and the bar's actions plus **Cancel**. Right-clicking a row that is not selected acts on
  that row only and keeps the selection.
- **Dragging a stack** (a project's two columns): dragging a selected row takes every selected row,
  in list order; the others fade where they were. The user's sprite carries a stack, at most three
  cards fanned (-4°, -2°, +2°), a count badge at its top right (accent, white `badge` text). Over
  the tray the first line reads `Drop to hand to an agent (3)` / `松开，交给 Agent（3 件）`.
  Dropped, the rows land in `Waiting for an agent` in their order, each with the 8 % accent wash
  fading over 2 s; the header robot squashes once per task, 120 ms apart. Dragging a row that is not
  selected takes only that row and clears the selection. On a board, dragging selected cards to
  another lane changes all their statuses.
- **The hand-off toast** (one task or many): `Handed 3 tasks to an agent` / `已交给 Agent：3 件`,
  **Copy for the agent** / `复制给 Agent` and **Undo**. Copy puts `Do adhd-todo tasks #31, #32 and
  #33.` (`Do adhd-todo task #12.` for one) on the clipboard and reads `Copied` / `已复制` for 1.5 s.

Decisions made while building it (2026-10-18; screens `select-*`):

- **One selection per page**, in `Main.multi` (`app/selection`, a pure module with its own tests): cleared when the page changes or the search starts or
  ends, and held to the rows the page draws. While it is empty the keyboard row is the only selected
  row, as before; a plain click or a plain arrow leaves it empty. A `⌘`-click with nothing ticked
  counts the keyboard row as the first one. `Esc` clears the selection, and leaves select mode,
  before it does anything else.
- **Select button** is the first of the title row's controls, before the milestone menu and the
  List | Board switch; not drawn on an empty page. The flash is remembered per page for the run of
  the app, and skipped with Reduce Motion. In select mode a row without a status glyph (a list, the
  columns, Agents, a board card) shows the checkbox where its author's avatar was (12 pt on a card);
  `Space` ticks the keyboard row.
- **Done** takes every selected task that is not done, an agent's too (cancelled ones reopen as
  done); there is no bulk reopen. **Move to** leaves subtasks with their parent and tasks already
  there. Each toast counts the tasks that really changed. A bulk **Done** plays the same fold as
  `Space`. Undo puts back the status, priority, project, milestone, owner and hand-off of every task,
  and of the open subtasks that closed with a parent (the reports that closing filed stay).
- **The bar** is a row with the toolbar role and a focus group, not `ui.Toolbar` (which reserves room
  for an overflow button). Below 640 pt of content it keeps icons; below 330 pt its count is the
  number alone. Its pages for Move to… and Priority are the task menu's, opened above the bar.
  While it is up, every list it floats over keeps its height (38 pt) and its 22 pt margin under its
  last row (the page's scroller, and each lane of a board), so the last row scrolls clear of it.
- **A stack in the hand** carries the user's unfinished selected rows only (the badge counts what
  will be handed); the others fade to 50 %. The cards behind are drawn by the carried row itself (a
  row can only paint over its own background, so it paints all its cards), 3 pt lower each.
  Handing several tasks stamps the first of them the newest, so `Waiting for an agent` reads in the
  order the rows were in.
  One rule for every drag, `Selection.Stack`: the selected rows in page order that the target takes, or
  only the dragged row when it is not selected or fewer than two are taken. The tray takes the user's
  unfinished rows; a board lane takes every selected card.
- **Toasts** are built by the views (`toast.go`) in mygo's own look, so a hand-off toast can have
  two buttons; they fade out but no longer fade in. Copy puts the numbers in ascending order.

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

### Summary

The end-of-day ledger (#67). References: `docs/design/mockups/v18.html` (the scene, decided),
`v14b.html` (rules and export), `v15.html` (the assistant's paragraph, the end-of-day reminder);
`v16`/`v17` are the drafts that were turned down. The page is the day (or the week) as a
Stardew-Valley-style shipping: everyone who finished something throws it into a shipping bin, then
a ledger counts it up.

**Where.** A sidebar row `Summary` (`总结`) under Completed, icon `bulletlist`, `⌘6`. Header: the
title, a `Today | This week` segment (`今天 | 本周`), `‹ date ›` paging back (a day or a week; `›`
stops at today), and `Export Markdown` (`导出 Markdown`) on the right. Under it, top to bottom: the
scene, the assistant's paragraph, the ledger. The page scrolls; the scene scrolls with it.

**The scene** (pictures in `app/theme/sprites/`, see **Sprites are files**):

- Edge to edge of the content column, 240 pt tall, one cell 3 pt: a grid 80 cells tall. It is laid
  out for 264 cells; a wider column extends the sky and the ground, a narrower one crops both sides
  equally (the actors and the bin stay inside the middle 220 cells).
- Three layers behind, nothing tall: the sky (an ordered-dither gradient), two rows of low hills,
  then grass and a path. Props: a fence behind the path, a lamp post at the right end. Nothing
  stands behind an actor, the bin or a stack.
- **The ground is Kenney's** (v23, the mixed version; Kenney Tiny Town, CC0, the tiles used and its
  License.txt in `third_party/kenney-tiny-town/`): in the day and dusk scenes the grass with its tufts
  and flowers, the path with its grassy edge, two trees at the left edge and mushrooms at the foot of
  the lamp post are its tiles, tinted to the scene's own grass; the sky, the hills, the fence and the
  lamp post stay ours. `docs/design/tools/scenes-v23.py` writes `scenes/day.png` and
  `scenes/dusk.png` from `docs/design/tools/scene-base/`. The night (the week) is left as it was: its
  stacks stand across the whole ground.
- **Sky by time**: day (blue with clouds) before 17:00, dusk (v18) from 17:00, night from 20:00, by
  the local time the page opens. A past day is drawn at dusk; the week at night.
- **Actors**: everyone who finished something in the period, 24 x 24 cells
  (`actors/<who>/<pose>.png`, poses `idle`, `lift`, `throw`, `hop`; a missing pose falls back to
  `idle`; an agent with no folder is the robot, `actors/bot/`, tinted like its avatar). You first,
  then the agents by count. Up to three stand left of the bin and three right of it (v22); a
  seventh agent and beyond throw from off the right edge.
- **Agents with a figure of their own** (v22): `claude` (the crab), `codex`, `gemini`, `cursor`,
  `copilot`, `opencode`. Each grows out of its official mark (the harness marks of Settings >
  Agents), never out of what its name suggests: Cursor is the cube with its triangular facet, not a
  mouse pointer; Gemini is the four-colour sparkle, red up, yellow left, green down, blue right and
  at its heart; Copilot is the head with its goggles pushed up over a pale face; OpenCode is the
  thick upright frame with a pale panel for a face. `docs/design/tools/actors-v22.py` draws the four
  new ones (poses `idle`, `idle2`, `lift`, `throw`, `hop`). The same rule holds for the avatars:
  `avatars/cursor.png` is the cube (with `cursor.dark.png`), not a pointer.
- **Goods**: each finished task is a good, 16 x 16 cells (`goods/<kind>.png`): `gem` a feature,
  `jar` a bug fix, `gear` a refactor or migration, `book` docs or design, `flask` tests, `mail` a
  reply, `coin` money or a bill, `scroll` a plan or research, `crate` anything else. The assistant
  picks the kind when it writes the paragraph (below) and it is stored; without a model, words of
  the title and the PR pick it, else `crate`. A task whose PR merged is **starred**.
- **The bin**: 40 x 16 cells (`props/bin.png`, lid frames `props/lid-*.png`), centred.

**The day, played once when the page opens** (the frames of v18):

1. The lid opens in 3 frames and stays open.
2. In the order the tasks were finished, 480 ms apart, each actor throws its good: you lift it over
   your head and throw; the crab raises both claws and flings it; Codex crouches and shoots it from
   its `>_`. Other agents use their own `lift`/`throw` poses. The good spins in the air; a starred
   one trails gold dust.
3. As a good lands the bin dips a cell, `+1` rises above it in pixel digits, and a starred good
   puts up a gold star. Goods heap in the bin's mouth: 7 drawn at most, the rest only counted.
4. When the last lands, everyone hops together once, fireflies come out at the right, and the
   actors then sway in place (a cell, every 450 ms) while the page is open.
5. The ledger's rows light up one by one, each count running up.

**The week**: night; one stack a day, Monday to Sunday under day labels. The week's goods fall from
the sky one by one (110 ms apart) and bounce once on landing; a stack is two goods wide. A day with
nothing is bare grass. Last, whoever finished most hops beside the busiest day's stack, once every
1.4 s.

- **Hover** a good in the bin or a stack: a small dark tip with its title and its last report line.
  **Click** opens the task in the inspector. **Click** the sky replays the scene.
- **Motion**: positions (a flight, a fall, a hop) move every refresh and snap to whole cells; poses,
  the spin of a good, the lid and `+1` step every 100 ms. The scene plays once each time the page
  opens or the period changes, then holds its last frame. With Reduce Motion it draws the last frame
  at once and nothing sways.

**The assistant's paragraph**: the assistant's avatar (36 pt) and a pixel speech bubble (a 3 pt
square-cornered frame with a stepped tail), header `Today · written by the assistant`
(`今天 · 助手写的`) and `Rewrite` (`重写`) on the right.

- Two or three sentences for a day, three to five for a week: what you and the agents finished,
  which PRs went in, what is still waiting. It only sees the period's finished tasks, their
  reports and PR titles, not code. It does not cheer, preach or invent.
- Written once the first time a period is opened, with the model of the composer, and stored; later
  visits show it at once. `Rewrite` writes it again. While today (this week) is not over and more
  has been finished since it was written, a dot beside `Rewrite` says so.
- It types in at 30 ms a character behind a blinking block caret; a stored paragraph shows whole.
- No model set: the paragraph is not shown; everything else is.

**The ledger**: one row per person, in the scene's order: avatar (18 pt), name, a dotted leader,
`2 PRs merged` (`合并 2 个 PR`) when there are any, the count in `title` weight, `件`, a chevron. A
row opens to that person's tasks (title, last report line with the PR number, project chip, time;
click opens the inspector). Last row: `Shipped today` / `Shipped this week` (`今天出货` /
`本周出货`) and the total.

**What counts** (v14b): done tasks only, cancelled ones do not. Who finished one is
`data.CompletedBy` (an agent's report within 5 s of completion, else you); a task done by a merged
PR counts for the PR's agent. Periods are local days; a week starts on the system's first weekday.

- **Empty**: no actors and no throws, just the scene with the lid shut, and the paragraph's place
  reads `Nothing finished yet today` (`今天还没有做完的事`); Export is dimmed.

**Export Markdown**: a sheet with the Markdown and `Copy` / `Save as…`. The paragraph first, then a
section per person with a line per task (`title (project, PR #12): last report`), then what is
still open (`3 in progress, 1 blocked`).

**The end-of-day reminder**: Settings > General > Notifications gets `End-of-day reminder`
(`收工提醒`, default on) and its time (default 18:30). Monday to Friday at that time, one
notification: `Day's done` (`收工了`) / `7 finished today, 3 PRs merged. See today's shipping?`; on
Friday `This week: 25 finished. The weekly summary is ready.`; a day with nothing finished `That's
it for today. See you tomorrow.` (never "0"). A click opens Summary. Not sent while the window shows
Summary; off with the Notifications switch too.

**Sound** (set A "wood box" of `docs/design/mockups/v21.html`; the other two sets there were
turned down). The scene makes a few quiet sounds while it plays, and nothing anywhere else in the app.

| When | Sound | File |
| --- | --- | --- |
| The lid opens | a soft wooden knock, sliding down | `lid.wav` |
| A good lands in the bin | a short wooden tock; each one a step higher on the major pentatonic (0, 2, 4, 7, 9, 12, 14, 16, 19 semitones) | `land.wav` |
| A starred good lands | a faint high ping over the tock | `star.wav` |
| Everyone hops (the last good landed) | three marimba notes going up (C, E, G), about 0.6 s | `done.wav` |
| The week: a good lands on its stack | a faint brushed tap, pitch varied a little | `tick.wav` |

- Restraint: a `land` sounds only 150 ms or more after the last one that sounded, and 8 at most; a
  star only on a land that sounds; a `tick` at most every 220 ms; the week ends on `done` 180 ms
  after its last good lands. The ledger counting up and the paragraph typing are silent.
- When: once each time the scene plays (the page opens, the period changes, the sky is clicked),
  scheduled from the plan's times (`theme.DayPlan`, `theme.WeekPlan`), not from frames. Leaving the
  page stops what is still to come. Not while the window is not key. With Reduce Motion (the last
  frame drawn at once) only `done`.
- Off when Settings > General > **Sound effects** (`音效`, a switch row under Notifications, `meta`
  `Quiet sounds while the day is shipped in Summary` / `总结页出货时的轻微音效`, on by default, kept in
  `prefs.json`) is off, or when macOS's "Play user interface sound effects" is off
  (`com.apple.sound.uiaudio.enabled` = 0 in the global domain; a missing key is on).
- Files: `app/theme/sounds/*.wav` (go:embed), mono 48 kHz 16-bit, replaceable as the sprites are; a
  missing or bad file is silence, not an error. They are written by `app/theme/sounds/gen.go`
  (`go run`), which synthesises set A of v21 at the level of its volume slider at 50% (peaks about
  -18 dBFS; the week's `tick` about -30 dBFS, louder than v21 so that it is heard), with a fixed noise seed so a rerun writes the same bytes. The pitch steps are the
  player's speed (`2^(semitones/12)`), so there is one `land.wav`.

**As built** (what the spec above leaves to the code; screens `summary-*.png`):

- The scene is a pure function of its plan and the time (`theme.DayPlan`, `theme.WeekPlan`:
  `Frame(t, calm)`), drawn by `SceneFrame.Draw` from the sprite files as pixelated bitmaps at 3 pt a
  cell, so a frame is a few dozen images and not 21 000 rectangles. Positions snap to whole cells
  every refresh; the page asks for a frame every 16 ms while something travels and every 100 ms while
  it rests (sway, fireflies). A good in the air is its 16 x 16 body turned a quarter turn every
  100 ms (the file's shadow cell is left off while it flies).
- People stand at x = 26, 66 (left of the bin) and 198, 160 (right of it) when there are four or
  fewer; with five or six, at 24, 52, 216, 188, 80, 160 in that order (three each side, 28 cells
  apart). A person right of the bin faces it (its picture is turned round), except those whose
  picture is a mark with a set direction: Codex (`>_`), Cursor (its facet), Gemini (its colours) and
  OpenCode. A seventh person and beyond are not drawn: their goods come in from past the right
  edge. A good is held up over the head of Gemini (12, 2), Cursor (12, -2), Copilot (12, 2) and
  OpenCode (12, 2), in cells from the top left of the figure. A pose with no file is
  `idle`; `idle2` is swapped in on the sway's beat. Throws are 480 ms apart, squeezed to 120 ms at
  the least so that a long day plays in about eight seconds.
- A week stack draws four rows of two goods (eight); more are counted in pixel digits (`+4`) above
  it. The lead (whoever finished most) stands in the empty day after the busiest one, else the one
  before it, else beside the stack itself.
- The ledger's count is the body size at the weight of a title, with a unit (`tasks`, `件`). A past
  day or week says `Shipped that day` / `Shipped that week`; a period with nothing says so in the
  paragraph's place (`Nothing was finished that day`, `... that week`). Still open counts in
  progress, in review and blocked.
- The paragraph is asked for as soon as a period with something finished is opened, off the UI
  thread, in the language of the window; while it is being written the first time the bubble holds
  the caret, a rewrite keeps the old text and shows the busy indicator where Rewrite is. A failure
  says `Couldn't write it. Try Rewrite.` and is not retried by itself. The goods change kind as the
  kinds arrive.
- The reminder comes up to two hours after its time (a Mac that slept through 18:30 says it at 19:00;
  one opened at ten at night says nothing), once a day (the date is kept in `prefs.json`), and is
  left out, for the day, while the window shows Summary. The time is a pop-up of the evening in half
  hours in Settings. Notification ids `end-of-day/day` and `end-of-day/week` open Summary.
- Sounds: the cues are a timer each, set once when the scene starts (`theme.DayPlan.SoundCues`,
  `WeekPlan.SoundCues`), and goods are taken in the order they land, not the order they are
  thrown. The macOS setting is read once per play off the UI thread; the window's focus is asked
  before each cue, so a sound is dropped when the window loses focus mid-scene. `Sound effects` is
  a group of its own under Notifications. `star.wav` carries its 40 ms lead in the file.

**Stored**: a `summaries` table (migration 010): period (`day`/`week`), its first date, the
paragraph, the model, the goods kinds by task id (JSON), created_at. Rewriting replaces the row.

### Recent changes

What changed in your projects lately (#110), read the way a changelog reads: not a review queue.
Reference: `docs/design/mockups/v25.html` (decided); an earlier draft grouped by review state was
turned down. Reviewing, checks and who does what stay with the agents; this page only helps you understand
what was shipped and what is being worked on, and the assistant does the reading for you.

**Off by default.** Settings > General gets a group **Pages** (`页面`) with one switch row, **Recent
changes** (`最近变化`), `meta` `See what each project merged lately, each pull request and issue in a
sentence. Needs gh signed in.` (`按项目看最近合并的 PR 和 issue，助手用一句话说清每条做了什么。需要已登录的
gh。`), off; kept in `prefs.json`. While it is off nothing below exists: no sidebar row, no `⌘7`, no
entry on project pages, nothing fetched.

**Where.** A sidebar row `Recent changes` (`最近变化`) under Summary, icon `git-branch`
(pixelarticons), no count, `⌘7`. Header: the title, a repository pop-up (`All repositories` /
`全部仓库` ▾), a `7 days | 30 days` segment (`7 天 | 30 天`), and `Updated 2 min ago` (`2 分钟前更新`)
at the right, which refreshes on click. With the switch on, a project page's title bar has a
`Recent changes` link that opens this page with that repository picked.

**Which repositories.** Those of the projects (their `repo_slug`), archived ones left out; every pull
request and issue in them, not only yours. The pop-up lists `All repositories`, then each repository
with its count for the period; one with nothing in the period is dimmed but can be picked. Picking one
shows only its section. The repository and the period are remembered. Clicking a section's project
name picks that repository.

**A section per project**, those with something in the period, the one changed last first. Its head:
folder icon, project name (`body` 600), the full repository in `meta`, and at the right `9 merged · 2
open · 2 issues` (`合并 9 · 在做 2 · 问题 2`). Then, in order:

- **The assistant's lines**: the assistant's avatar (24 pt) and the pixel speech bubble of Summary
  (smaller, 2 pt frame), header `These 7 days · written by the assistant` (`这 7 天 · 助手写的`) and
  `Rewrite` (`重写`): two or three sentences on what the project gained, what was fixed and what is
  still open. Written once per project and period, stored; a dot beside Rewrite when more was merged
  since (as in Summary).
- **New** (`新功能`) and **Fixed** (`修复`): the merged pull requests, newest first. Each line: the good
  of Summary as its icon (`gem` new, `jar` fixed, 18 pt), one sentence, and under it in `meta` who did
  it (the agent's avatar and name when a task has the PR's URL, else the GitHub author), when (`today`,
  `2 days ago`) and `#41`.
- **Other** (`其他`): docs, refactors, dependencies, versions, screenshots, folded as one line `3 more:
  docs, version, screenshots` (`其他 3 项：文档、版本号、截图`); a click unfolds them, one line each.
- **Open** (`在做`): the open pull requests, drafts too: a green pull-request icon, one sentence, who and
  `open for 2 hours` (`开了 2 小时`). No checks, no review state.
- **Issues** (`问题`): issues opened or closed in the period: an issue icon (a ring with a dot; green
  open, purple closed), one sentence, `New · you · yesterday · #58` (`新问题 · 你 · 昨天 · #58`) or
  `Solved · fixed by #41 · today · #56` (`已解决 · 由 #41 修好 · 今天 · #56`) when gh names the pull
  request that closed it.

An empty group is left out. A repository gh cannot read says so in its section in one line with a
command to copy; the others show as usual.

**One sentence per item.** The assistant reads a pull request's title and description (an issue's title
and the first 2000 characters of its body) and writes, in the window's language, one plain sentence of
at most 30 characters in Chinese or 12 words in English saying what it does for the user (`周报总数是两位数
时，不再被截成省略号`, not `fix(summary): ...`), and puts it in New, Fixed or Other. Written once per
item and kept by its URL and `updatedAt`, so it is written again only when the title or description
changed. No model: the title without a conventional-commit prefix (`feat(scope): `), New for `feat`,
Fixed for `fix`, Other for the rest.

**Click an item** to unfold **the assistant's explanation** under it, three points: **What** (做了什么),
**Why** (为什么), **Where** (动了哪), from the description and the names of the changed files (never the
code), with `Read the description and the changed files` (`读了 PR 描述和改动的文件`) in `meta` above.
Written the first time it is opened and kept. Under it `Open on GitHub` (`在 GitHub 打开`, with the
pixel external-link icon) and, when a task has the URL, `Task #121` which opens it in the inspector.
Issues unfold the same way (What, Why, and the pull request that fixed it when there is one).

**Data**, through the `gh` the user is signed in to, per repository: `gh pr list --state merged
--search "merged:>=<date>"`, `gh pr list --state open`, `gh issue list --state all --search
"updated:>=<date>"` (fields: number, title, body, url, author, dates, draft, head branch, labels, and
the changed files' names for the explanation). Fetched when the page opens and every 10 minutes while it
is shown and the window is key; never while the switch is off.

**Stored**: not in the database. The sentences, categories, the per-project lines and the explanations
are a cache file in the data folder (`recent-changes.json`), keyed by URL and `updatedAt` (lines by
repository and period); deleting it only costs the assistant's time.

**Empty**: no project has anything in the period: `Nothing changed in the last 7 days` (`最近 7 天没有变
化`) with the assistant's avatar. gh missing or signed out: the page says what to run (`brew install gh`
/ `gh auth login`) with Copy.

**As built** (what the spec above leaves to the code; screens `recent-*.png`):

- Fetching goes through `ghpoll.Runner` (the one the poller uses, `ExecRunner` in the app): per repository
  `gh pr list --state merged`, `--state open` and `gh issue list --state all`, one after the other, 100 at
  most each, four repositories at a time, off the UI thread (`data.GHRecent`, `data.FetchRepos`; the demo
  and the tests use `data.FakeRecent`). The period is 7 or 30 calendar days ending today, from local
  midnight; the search date is the UTC date of that midnight, a superset, and the exact moment cuts the
  rest. An issue counts when it was opened or closed in the period; an open pull request counts however
  old it is. `fixed by #N` is the first of the issue's `closedByPullRequestsReferences`. A pull request is
  by the agent of the task whose `pr_url` is its URL, else by its author: `You` (`you` for an issue) when
  that is the account gh is signed in to, with the robot for an app, else the login alone.
- The page reads when it opens, on a click on `Updated …`, when the period changes, and every ten
  minutes while it is shown and the window is key (`App.WindowKey`); the answer of an older read is
  dropped. The repository and the period are in `prefs.json` (`recentRepo`, `recentDays`) beside the
  switch (`recentChanges`); a remembered repository that is no project's is All repositories.
- The assistant is asked, for the sections on screen, off the UI thread and once for each read:
  sentences, category and topic for the items that have none kept (12 items a request, language of the
  window), then the lines of the section, then, when an item is unfolded for the first time, its
  explanation. A failure is not retried by itself: sentences fall back to titles until the next read, the
  lines show `Couldn't write it. Try Rewrite.`, an explanation shows a Retry. The three are
  `Assistant.DescribeRecent`, `WriteRecentLines` and `ExplainRecent` (`internal/agent/recent.go`).
- `recent-changes.json` holds the sentences keyed by `<language> <url>` with the `updatedAt` they were
  written for, the explanations in the same entry (so a changed item loses both), and the lines keyed by
  `<language> <days> <repo>` with when they were written (a dot beside Rewrite when something merged
  after). An item not updated for 90 days is dropped when the file is next written.
- With no model the titles stand in (`FallbackNote`: the conventional-commit prefix off; feat New, fix
  Fixed, the rest Other, with no topics, so the fold reads `3 more`), there are no lines, and an
  unfolded item offers `Open on GitHub` and `Task #n` only.
- The pop-up lists the projects by name with their count (dimmed at 0, still pickable); a project name in
  a section picks its repository. The good beside a line is drawn at one point a cell (17 pt in the 18 pt
  slot) to keep its pixels square; a pull request in Other has the good its title points to.
- gh missing or signed out shows as the page's state only when every repository failed so; one that
  failed alone says so in its section (a `gh repo view owner/name` to copy when GitHub refused it). In
  Settings the switch's second line turns to what to run when the switch is on and gh cannot be used.
- Not done: the typing-in of the assistant's lines (they show whole), more than the first pull request
  that closed an issue, and a state of its own for GitHub being offline besides the section line.

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
  scrolls. A **pixel panel** (`theme.DrawPixelPanel`, `v8.html` B): `bg`, corners stepped in twice by
  2 pt, a 2 pt edge in `PixelEdge` (`bg` a fifth of the way to `text`), and a hard shadow 4 pt right
  and down, not blurred (black at 10 %, 50 % in dark mode); 12 pt padding inside the edge. Top to bottom: the header (`Thursday, Oct 15` / `10 月 15 日 星期四` in `body.strong`, the
  count `5 items` / `5 项` in `text.tertiary`, a 24 pt `IconClose` button at the right); every item of
  the day, 28 pt rows drawn like the cell's items (avatar, a reminder's time, the title; finished items
  struck through, milestones as the accent chip), a click opens the task in the inspector and closes
  the card; at the foot an input in the sheets' well, its corners stepped once (`theme.DrawPixelWell`) (`Add to this day…` / `添加到这天…`). Enter adds a
  plain task by me, due that day at 18:00, clears the input and keeps the card open, the new item
  joining the list; empty text does nothing. **Closing**: `Esc`, a click outside (it does nothing
  else: it does not select the day under it), the close button, a change of month or `Today`, and
  leaving the page. While the card is open the window's shortcuts wait, so the arrows do not move the
  day. **Opening** is a pop-out (`theme.DayCardAt`): the card is laid out at its full size
  and position from the first frame and a visual transform grows it from the corner of the cell it
  grows from, so nothing is resized or clipped. Over 140 ms its scale goes from the cell's width over
  the card's (clamped to 0.5 to 0.95) to 1 along an ease-out cubic, and its opacity goes 0 to 1 with
  an ease-out over the first 100 ms. A frame is asked for every 16 ms only while it plays; with
  Reduce Motion the card is full size and opaque at once. Closing has no animation. Screens
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

### Sheets and confirmations

Small sheets on the dimmed window, built like Settings (12 pt radius, 24 pt padding) but 400 pt wide.
`Esc` and the Cancel button close them; the primary button is the accent one.

- **New project**: title `New project`; field `Name` (focused); field `GitHub repository` (optional,
  mono, placeholder `owner/repo`) with the line `Agent reports from this repository land in the
  project.` Create is off until the name has text. A pasted GitHub address is read as `owner/repo`.
  Problems read under the field in `text.secondary`, never red: `Use the form owner/repo, like
  QingYunA/adhd-todo.` and `“adhd-todo” already uses this repository.` Enter in either field creates;
  the new project opens. Opened from `New sub-project…` in a project's menu, the title is `New
  sub-project`, a line under it reads `Under adhd-todo`, and the project it makes is that project's
  sub-project; there is no other difference and no extra field.
- **Category sheet**: `New category` / `Edit category` (400 pt); see **Categories (M9)**.
- **Milestone sheet**: `New milestone` (Title, Due date optional), `Rename milestone` (Title) and
  `Due date` (a calendar and a Clear link). Done is the primary button; Enter saves.
- **Delete confirmation** (a project, a task): title `Delete “name”?`; body in `text.secondary`.
  For a project: `This removes its 14 tasks, their reports and its milestones. It can't be undone.
  To keep them, archive it instead.` (`This removes the project and its milestones.` when it has no
  task.) For a task: `Its progress reports go with it. This can't be undone.` Buttons `Cancel`
  (focused, so Enter does not delete) and `Delete` (accent, not red). Esc cancels.

### Categories (M9)

References: `docs/design/mockups/v24.html` (all frames are built: 7a to 7d are the 12 x 12 editor
and the image import). Plan: `docs/plans/m9-custom-categories.md`.

The user makes categories; Work and Life are built in, always there, movable and re-iconable but
neither renamed (their names follow the language) nor deleted. A category has a name (unique in any
case, at most 32 characters; at most 24 categories), a 12 x 12 icon and one of eight colors, the
project colors in order: indigo, yellow, green, red, blue, pink, teal, orange (Work starts as a
blue briefcase, Life as a green house). The icon is drawn in that color wherever the category shows: sidebar sub-rows (12 pt), the
inspector's value, the Category page and the task menu's row (18 pt), the sheet's preview. In dark
mode the colors are the project folders' (`theme.ProjectColor`); there is no second set. On an
accent-filled menu row the icon is white, like a project folder in `Move to…`.

- **Sidebar menus** (system menus, like a project's; v24 frame 1): right-click a category: `New
  category…`, `Edit category…`, `Move up`, `Move down` (each left out at its end of the list),
  and, for the user's own, a separator and `Delete category`. Right-click `Mine` or `Unsorted`: `New
  category…` only.
- **Category sheet** (frames 3 to 5, 8): title `New category` / `Edit category`; `Name` (focused;
  Work and Life show their localized name read-only with the line `Work and Life keep their names.`);
  `Icon`: the chosen icon at 36, 18 and 12 pt in the chosen color, on `bg.subtle`; `Color`: eight 18
  pt pixel circles (a 24 pt target, a ring on the current one; a11y `Color: Red`); a `Search icons`
  field; a 240 pt scroll area with a hairline border: nothing typed, the curated groups (Work, Study,
  Health, Money, Home, Hobby, Travel, People, `meta` headings, localized), typed, the matches as one
  flat grid, or `No icon matches`; 36 pt cells, 9 to a row, the icon at 18 pt in the text color (a11y
  label: the icon's name); the chosen cell has a 2 pt accent ring and a faint accent fill and shows
  its icon in the chosen color. A new category starts on the first curated icon and on the color its
  name hashes to, which follows the name until a swatch is clicked. Create / Save is off while a new
  name is blank; Enter in `Name` creates or saves. Problems read under the field in `text.secondary`,
  never red. After Create the category's page opens, unless the sheet came from the task menu, which
  gives the category to the task instead.
- **Palette.** An icon's pixels are empty, the category's own color (`#`, follows the category) or
  one of 12 fixed colors, the same in light and dark: `a` indigo `#5B5BD6`, `b` yellow `#E0A800`,
  `c` green `#30A46C`, `d` red `#E5484D`, `e` blue `#3B82F6`, `f` pink `#D9488B`, `g` teal `#0EA5A4`,
  `h` orange `#E08A1E` (the project colors, `theme.ProjectColors`), `i` white `#FFFFFF`, `j` gray
  `#8E8E93`, `k` black `#1C1C1E`, `l` brown `#8A5A3B` (`theme.PaletteColors`).
- **Icon modes** (frames 7a to 7d). Under `Icon`, the preview and two buttons, `Draw` and
  `Import image…` (`画`, `导入图片…`). In the editor and the import the buttons sit at the right of
  the `Icon` label and name the other ways: `Library` (`图标库`, back to the search and the grid; the
  picked library icon replaces the drawing), `Draw`, `Import image…`. Create stays off while the icon
  is empty.
- **Editor** (7b). A 12 x 12 grid of 18 pt cells with 1 pt lines in `border`, empty cells in `bg`;
  at its right the 24 / 18 / 12 pt preview, `Clear` and `Invert` (swaps empty and own-color cells;
  palette cells turn empty, they have no opposite), four arrows that shift the drawing one cell
  without wrapping, `Flip` (left to right). Under the grid the pens: the category color first
  (`Category color`, `类别色`), set off by a 1 pt rule in `border` since it can match a palette color,
  then `a` to `l`, 22 pt squares, the picked one with a 2 pt border,
  and the line `Click a cell to draw, drag to keep going. ⌘Z undoes.` Pressing an empty cell or one
  of another color paints it with the pen; pressing one already of the pen's color erases, and the
  drag goes on in that mode, filling the cells between samples. One stroke or button is one step of
  `⌘Z` (while the focus is inside the editor). With the grid focused the arrows move a cell and
  `Space` toggles it.
- **Import** (7c, 7d). A file panel (PNG, JPEG, GIF, WebP, SVG) or one file dropped on the sheet.
  The picture (144 pt, file name under it) → the 12 x 12 result on `bg.subtle` (144 pt), a
  `Threshold` stepper (percent, steps of 4, starts at Otsu's), the 24 / 18 / 12 pt previews and the
  `Outline only` switch. The picture is scaled to 256 px; its ink is the alpha when its edge is
  transparent, else its distance from the median edge color; it is cropped to the ink, padded to a
  square and averaged into 10 x 10 cells in the middle of the 12 x 12 grid; cells with ink at or above
  the threshold are set, each in the nearest palette color (OKLab), and a color covering 80 % of the
  cells becomes `#`. Outline only keeps the cells that touch an empty cell, all as `#`. An SVG must
  be right-angle paths (24-unit view box); it is the icon as it is (no threshold). Errors, under the
  icon: `Couldn't read this image.` (`读不了这张图片。`), `This SVG has curves; export it as PNG
  first.` (`这个 SVG 有曲线，请先导出成 PNG。`).
- **Delete** is immediate (the list is the user's own); a toast `Deleted “Reading” · 5 tasks
  unsorted` (no count when 0) with **Undo** for six seconds, which puts the category back with its
  icon, color, place and tasks (only the ones still unsorted).
- **Chinese** (frame 8): `新建类别…`, `编辑类别…`, `上移`, `下移`, `删除类别`, `名称`, `图标`, `颜色`,
  `搜索图标`, `没有匹配的图标`, `查找类别`, `工作和生活不能改名。`.

### Keyboard

`⌘K` the page's composer · `⌘Z` the assistant's last change (see **Assistant**) · `⌘,` Settings · `⌘F` search · `↑ ↓` move selection through rows on the current screen · `↵` open the
selected row in the inspector · `Space` toggles done for your own tasks · `Esc` closes the
inspector, then clears the selection, then the search · `⌘`-click, `⇧`-click, `⇧↑ ⇧↓` and `⌘A` select several
(see **Selecting several tasks**), `⌘↵` marks them done, `⌘⇧A` hands them to an agent · `⌘1` Mine, `⌘2` Agents, `⌘3` Inbox, `⌘4` Calendar, `⌘5` Completed, `⌘6` Summary, `⌘7` Recent changes (when it is on, see **Recent changes**).

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
| New project, milestone, category sheets | creates / saves | closes the sheet | |

### Notifications

System notifications (v12, `docs/design/mockups/v12.html`). macOS draws them; the app chooses the words,
the sound, the stack and where a click goes. Every kind opens its task in the inspector over Mine, and
the notifications of one project stack together in Notification Center (`Group` = the project; the
Inbox is one stack).

| Kind | When | Title / subtitle / body | Sound |
|---|---|---|---|
| Handed back | an agent's report moves the task to In review | `Claude Code finished: <title>` (`Claude Code 做完了：<title>`) / project / with a PR `PR #12 is ready for your review`, else the agent's last summary | yes |
| Stuck | an agent's report moves the task to Blocked | `Claude Code is stuck: <title>` (`Claude Code 卡住了：<title>`) / project / what it waits for (`next_step`, else `summary`) | yes |
| Checks failing | the GitHub poller finds a PR's checks newly failing (the moment Mine's row appears) | `Checks are failing on PR #12: <title>` (`PR #12 检查没过：<title>`) / project / `Open the PR to see which one failed` | yes |
| Merged | the GitHub poller finds the PR merged and completes the task | `PR #14 merged: <title>` (`PR #14 已合并：<title>`) / project / `The task is done` | **silent** |
| Reminder | unchanged: `remind_at` passes, once; due while the app was closed, at the next launch | `Reminder: <title>` / — / project or Inbox | yes |

The agent is named as people write it (`Claude Code`, `Codex`, ...), as in the inspector's From.

- **Once.** Each report or GitHub event notifies once, when the app first sees it.
- **Not what you did.** A change the user made in the app (a status set by hand) notifies nothing.
- **Not what you are looking at.** While the app is frontmost and its window shows Mine, a new
  handed-back, stuck or pull request event does not notify: its row is already on screen. In the
  background, or on another page, it does. Reminders always notify.
- **No catch-up** for agent and GitHub events: what happened while the app was closed is on Mine at the
  next launch. Only reminders catch up, as before.
- **A burst becomes one.** When a third event arrives within 10 s of the first one still shown, one
  summary notification takes their place: `3 things need you` (`3 件事等你`), body `2 finished, 1 stuck
  · open Mine` (`2 个做完了，1 个卡住了 · 点开去「我的」`), counting each kind that occurred; a click opens
  Mine. Further events in the burst update the summary (same notification ID) until 10 s pass with none.
- **The switch.** Settings > General > Notifications (below). Off, nothing notifies, reminders included.

### Updates (M10)

The app updates itself from GitHub releases with mygo's native updater plugin (`native.Plugin`); the
window is the plugin's own, so the app draws nothing of it. All of it shows only in a build that can
update (`mygo.Updater.Enabled()`: `updates` in `app/mygo.json`); updates are off for now, see
`docs/plans/m10-auto-update.md`.

- **The menu item**: `Check for Updates…` (`检查更新…`) in the app menu right after About. It opens the
  plugin's window at once: release notes of the new version, **Install Update**, **Remind Me Later**,
  **Skip This Version**, a progress bar while it downloads, then a relaunch. When there is none, the
  window says the app is up to date; a failed check says why. The label follows the app's language.
- **The window** is the plugin's native-UI window (no webview), in the language of the app. Background
  checks run once a day, the first 10 s after launch when one is due, and never in a development build.
- **The switch**: Settings > General > Software updates (below).
- `adhd-core` is inside the app and updates with it; see `docs/plans/m10-auto-update.md`.

### Settings

A window of its own (M4c; the in-window sheet is gone). `⌘,` (the app menu's `Settings…` item) and the
sidebar gear open it; there is one instance, and a second request brings it to the front, on the pane
asked for. **One fixed size, 680 × 520 pt of content**, not resizable: a pane taller than that scrolls
under the toolbar, and a tab change never resizes the window (M5; resizing to each pane's height was
the flicker). Each pane has its own scroll key. Reference: `v4.html` frames 5 to 7, and `v5.html` frames 5 and 6 (logos, Agents); screens
`settings-general-*` (`settings-general-denied-*`: macOS refused the notifications), `settings-assistant-*` (at rest), `settings-assistant-new-*` (nothing set up),
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
- **General**: Language (a pop-up: System / English / 简体中文); **Notifications** (v12): a group of one
  row, **System notifications** (`系统通知`) with the `meta` line `Agents done or stuck, failing or merged
  PRs, and reminders` (`Agent 交回或卡住、PR 检查没过或合并、提醒到点时通知你`)
  and a switch at the right, on by default. When the last notification was refused by macOS
  (`mygo.ErrNotificationsDenied`) the `meta` line reads, in `stale`, `macOS does not allow adhd-todo to
  notify` (`macOS 不允许 adhd-todo 发通知，开关不起作用`) and an **Open System Settings** link
  (`打开系统设置`, followed by the pixel external-link icon, as every Settings link that leaves the app) opens the Notifications settings; **Archived projects** (a row per
  archived project with a **Restore** link, or the empty line); **Data folder** (the path in `meta`, written
  from the home folder as `~/Library/Application Support/adhd-todo`; **Show in Finder** at the right
  reveals the real path); About (`adhd-todo 0.3.0`, `Open source, MIT license`, a GitHub
  link). **Software updates** (M10; only in a build that can update): a group of one row between Sound effects and Pages, `Software
  updates` (`软件更新`), `meta` `Checked once a day; you choose when to install` (`每天检查一次，装不装由你决定`),
  and a switch at the right, on by default, which is mygo's updater plugin's "automatic checks": the
  plugin keeps it in its own `updater.json`, not in the prefs, and the row reads it back whenever the
  plugin reports a change (a check, an answer in the update window).
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
  key shows dots and an empty field keeps it; **Get a key** (with the external-link icon) at the right opens the provider's key
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
- **Provider logos** (M6, pixel since v11, `docs/design/mockups/v11.html`): the official marks of
  lobe-icons (MIT, `@lobehub/icons-static-svg` 1.95.1) stay vendored as SVG under `app/theme/logos/`
  with their LICENSE, as the source of the pixel marks; the app draws the 18 x 18 sprites of
  `app/theme/provider_marks.go`, generated from them (see **Pixel language**), so nothing is fetched.
  `theme.ProviderLogo(c, presetID, size)` draws one `size` pt square: DeepSeek, 智谱, 通义千问, Gemini,
  OpenRouter and 硅基流动 in their own colors; OpenAI, Kimi and Ollama only have one color and follow
  the text color (white in dark mode; the caller passes it, Settings uses `text`); Anthropic is the
  Claude Code crab; Magpie is the pixel robot in teal; Custom is the pixel plus in `text.secondary`.
  Sizes, whole pixels only: `LogoCard` 36, `LogoTile` 36, `LogoRow` 18, `LogoChip` 18 (the composer's
  model chip and its menu). The provider logo is never a colored tile with letters any more.
- **Agents** (M6, v14, `docs/design/mockups/v14.html`): the label **Coding agents** (`编程 Agent`),
  one plain `meta` line under it (the M6 wording: what Install MCP does and does not do), then two
  sections, both on the first screen of the 680 x 520 window, before the pane scrolls:
  - **On this Mac** with its count (`On this Mac 6`, `本机已有 6`; the number is quiet, in
    `text.tertiary`, after the section label): every agent whose command is on PATH, whose config
    folder or file is there, or that ever reported. A report keeps an agent here even when this
    look no longer finds its command, because its MCP is installed somewhere. The section is two
    white group columns; rows read left to right and then top to bottom, always in catalog order,
    never sorted by reports (the row must not move under the pointer): Claude Code, Codex, DeepSeek
    Harness, Gemini CLI, Cursor, opencode, Copilot (VS Code), Qwen Code, Kimi CLI, Windsurf, Zed,
    Amp, Kiro, Cline. A row (48 pt in the app; the mockup draws 50, but the app's toolbar is taller,
    so both sections have to fit the first screen): the 24 pt official mark (`theme.HarnessMark`),
    the name (`body`), one `meta` line, and at the right one of `✓ MCP installed` (done green;
    also for an agent that only reported), **Install MCP** (accent; `Installing…` while it runs),
    or nothing.
    The line is `N reports today`, else `No reports today`, else `Found · gemini` / `Found ·
    ~/.dsh` (what was found). An install that fails says why under the line, in `meta` and the
    blocked red, in plain words. With no agent found and none reported, the section is one empty
    row: `No agent found on this Mac yet` / `还没在这台 Mac 上找到 Agent`, with **See how to
    connect** / **去看看怎么接**, which opens the README's connect section; More still lists the
    whole catalog.
  - **More** with its count: every catalog agent not on this Mac, one white block of compact two
    column rows (8 in the design, four rows). A row (31 pt in the app; the mockup draws 32): the
    18 pt official mark, the name in `meta` size and primary ink, and `Get it` / `去下载` and the 12 pt pixel external-link icon at the
    right, which opens the agent's URL. There is no status line and no "Not on this Mac" (the
    group says it), and More never folds.
  Under the two sections, exactly as before: the manual snippet when a config file is not plain
  JSON, or when a Cordis patch is not a patch list adhd-todo can append to; then the **Open agent
  work in** / `去看看时打开` row with its pop-up (`The agent's app` default,
  `Terminal`, `Editor`, `Finder`), and the `claude mcp add` command in a mono well with **Copy**.
  Under it, **Another agent** / `其他 Agent`: one `meta` line and **Copy setup guide** /
  `复制接入说明`, which copies a guide any agent can follow to add adhd-todo itself (`agentsetup`
  Guide, also printed by `adhd-core guide`): the server, how to install it (its own command first,
  else its config file after a backup), how to check it (`get_overview`), how to report, and an
  offer to add the reporting rule to the agent's global instructions after asking the user. The
  manual snippet of a failed install has the same button. (#86)
  The choice is kept in `prefs.json` next to the database. Screens `settings-agents-*`; the
  scrolled one is `settings-agents-open-with-*`. "Install MCP" is the only word for this; it is
  never called connecting. Detection (`internal/agentsetup`) reads files and looks on PATH (plus
  `/opt/homebrew/bin`, `/usr/local/bin`, `~/.local/bin` and the like, which a Dock-launched app's
  PATH lacks) every 30 s and after an install; **nothing is installed without the button**.
  - Commands, run with the absolute `~/.local/bin/adhd-core` and a 30 s timeout: `claude mcp add --scope
    user adhd-todo -- <core> mcp`; `codex mcp add adhd-todo -- <core> mcp`; `gemini mcp add -s user
    adhd-todo <core> mcp`; `qwen mcp add --scope user adhd-todo <core> mcp`; `kimi mcp add --transport
    stdio adhd-todo -- <core> mcp`; `amp mcp add adhd-todo -- <core> mcp`; `code --add-mcp
    '{"name":"adhd-todo","command":"<core>","args":["mcp"]}'`.
  - Files (copied first to `<file>.adhd-todo.bak`, key order, number text and indentation kept, created
    when missing): Cursor `~/.cursor/mcp.json` `mcpServers`; opencode `~/.config/opencode/opencode.json`
    `mcp` (`type: local`, `command: [core, "mcp"]`); Windsurf `~/.codeium/windsurf/mcp_config.json`
    `mcpServers`; Kiro `~/.kiro/settings/mcp.json` `mcpServers`; Cline `~/Library/Application
    Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json`
    `mcpServers`; Zed `~/.config/zed/settings.json` `context_servers`.
  - **DeepSeek Harness is the one that is not JSON.** It takes MCP servers as Cordis patch entries,
    and it is detected by `~/.dsh` alone (the design's `dsh` is not on PATH). Install appends one
    entry to `~/.dsh/cordis.patch.yml`, the home patch layer every profile reads after its own:
    `- insert:` with `@deepseek-ai/dsh-mcp-client`, `serverName: adhd-todo`, `transport: stdio`,
    `command: <core>`, `args: [mcp]`, `failOnStartupError: false` (the plugin's documented minimal
    stdio configuration, with one change: true would make the plugin refuse to activate when
    adhd-core fails to start, and false degrades to no tools, so adhd-todo can never stop DeepSeek
    Harness from starting). A patch file that is not a patch list adhd-todo can append to (a
    mapping, a flow sequence or a list indented from column 0, for example) is not rewritten: the
    pane falls back to the manual snippet, as for a JSON file it cannot safely edit. Installed is
    read from the home layer and every profile's `cordis.patch.yml`.
  - **Amp, Kiro and Cline** join the catalog in v14. Amp is found by its command and installed with
    its own `amp mcp add`; its `~/.config/amp/settings.json` (`amp.mcpServers`) is read for the
    installed state. Kiro and Cline are found by their folder/file and installed by adding the
    server to their MCP JSON. Nothing was guessed: an agent whose method could not be verified is
    listed in More only, never given a button that would fail.
  - Installed is read from `~/.claude.json`, `~/.codex/config.toml` (`[mcp_servers.adhd-todo]`),
    `~/.gemini/settings.json`, `~/.qwen/settings.json`, `~/.kimi/mcp.json`, the files above, VS
    Code's user `mcp.json` (`servers`), `~/.config/amp/settings.json` (`amp.mcpServers`), and
    DeepSeek Harness's patch layers.
- **Avatars of agents** (M6): besides Claude Code, Codex, Cursor and Gemini CLI, a dark rounded square
  with a gray center for opencode and a dark head with white goggles and purple pupils for Copilot
  (both are lightened in dark mode); Qwen Code, Kimi CLI, Windsurf, Zed and Cline are the robot in their
  brand color. `agentName` and `theme.AgentAvatar` agree on the names agents report under
  (`Visual Studio Code` is Copilot (VS Code), `qwen-code-mcp-client` and `qwen-cli-mcp-client-*` are Qwen
  Code); any other name gets a robot colored by the name. These are the avatars of the rest of the
  app: the rows of Settings > Agents draw the harnesses' official marks instead, at 24 and 18 pt
  (see **Pixel language**). The agents added in v14 (DeepSeek Harness, Amp, Kiro) have no 12 x 12
  avatar yet; when it is drawn it is a file `app/theme/sprites/avatars/<name>.png` (see **Sprites
  are files**), and until then elsewhere they get the robot colored by the name; `agentName` maps a
  report to their catalog name so it still lands in the right Settings row.
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
