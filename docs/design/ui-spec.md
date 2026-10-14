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
| Raycast | The panel: one large borderless input, a list with a keyboard-highlighted row, a footer of key hints. | Command-palette complexity. |
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
6. **Fast is a feature.** Panel input is focused the instant it opens. No spinner for local data.

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
| `input` | 17 / 22 | 400 | panel input |
| `body` | 13 / 18 | 400 | rows, cards, fields |
| `body.strong` | 13 / 18 | 500 | task titles |
| `meta` | 12 / 16 | 400 | summaries, ages, field labels |
| `section` | 11 / 14 | 600, tracking +0.4, uppercase | section headers |
| `mono` | 11.5 / 16 | 400 | branch, `#42` |

### Space, shape, motion

- 4 pt grid. Common gaps 4, 8, 12, 16, 24.
- Radii: 6 (pills, inputs), 8 (cards, rows), 12 (panel, popovers).
- Shadows only on floating things: panel, drag ghost, popovers.
- Motion 160 ms ease-out for hover, selection and drawer; 220 ms spring for the panel
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

### 1. Quick panel (tray flyout, 440 x 560)

Liquid Glass background (glass plugin) with a 12 pt radius; fall back to vibrancy.

```
┌──────────────────────────────────────────────┐
│  Add a task…                                 │  input, 17 pt, no border, 16 px padding
├──────────────────────────────────────────────┤  hairline
│  AGENTS                              3 active │  section + count
│  adhd-todo                                    │  project, meta, secondary
│  ◐ Port core to Go   feat/go-core       2m   │  body.strong · mono pill · meta age
│     Ported the store and migrations           │  meta, secondary, 1 line, ellipsis
│  ● Fix tray race     fix/tray     ! 3d stale  │  blocked glyph; stale label in `stale`
│                                               │
│  MINE                                         │
│  ○ Reply to design feedback      in 1h ⏰     │  remind/due time on the right
│  ○ Book dentist                  tomorrow     │
├──────────────────────────────────────────────┤
│  ↵ add   ↑↓ select   ⌘↵ open   esc close     │  footer hints, tertiary
└──────────────────────────────────────────────┘
```

- Rows are 44 px for two-line agent rows, 32 px for one-line rows. Keyboard highlight uses
  `bg.selected` with an 8 pt radius inset 6 px from the edges.
- Agents section lists tasks with `source = agent` in doing / blocked / review, grouped by
  project, blocked first. Mine lists `source = user` tasks in todo / doing, sorted by
  remind/due time, then priority.
- Typing filters both sections live; Enter with no match creates a task in the inbox and
  shows an inline "Added to Inbox" confirmation for 1.5 s. (M3 replaces Enter with the agent.)
- Empty Agents: "No agent is reporting yet." and one line of help: "Coding agents report here
  through the adhd-todo MCP server." with a "How to connect" link that opens the README.
- Empty Mine: "Nothing on your plate. Type above to add a task."

### 2. Main window (1100 x 720, min 800 x 520)

Unified title bar. Three columns: sidebar, board, inspector (only when a task is open).

```
┌─────────────┬───────────────────────────────────────────────┬──────────────┐
│ ● ● ●       │ adhd-todo   QingYunA/adhd-todo      Milestone ▾│              │
│             ├───────────────────────────────────────────────┤  Inspector   │
│ Inbox     2 │ ○ Todo 3     ◐ Doing 2    ● Blocked 1   ◎ ... │  360 px       │
│             │ ┌──────────┐ ┌──────────┐                     │              │
│ PROJECTS    │ │ Title    │ │ Title    │                     │              │
│ adhd-todo 5 │ │ ▮▮ feat/x│ │ #42      │                     │              │
│ docs      2 │ └──────────┘ └──────────┘                     │              │
│             │                                               │              │
│ Settings    │                                               │              │
└─────────────┴───────────────────────────────────────────────┴──────────────┘
```

**Sidebar (220 px, sidebar material).** Inbox, then `PROJECTS` with each project and its
count of active tasks. A small blocked dot appears after a project that has a blocked task.
Selected item uses `bg.selected`.

**Board.** Columns: Backlog, Todo, Doing, Blocked, Review, Done. Column header: glyph, name,
count; no column backgrounds except a subtle `bg.subtle` well while dragging over it. Columns
are 248 px with 12 px gaps; the board scrolls horizontally when narrower than six columns.
Done shows the 10 most recent and a "Show all" link.

**Card.** `bg` with `border` hairline and 8 pt radius, 12 px padding. Line 1: title
(`body.strong`, up to 3 lines). Line 2, only if any exist: priority bars (if not P2), branch
pill, `#PR`, agent age ("2m"). Hover: border goes to `text.tertiary`. Drag: card lifts with a
shadow and 2° tilt; drop target column shows the well; on drop the card settles with the
spring. Stale cards show the stale label on line 2.

**Inspector (360 px, slides in 160 ms).** Title as a large editable field (`title` token).
Then a compact property list with left labels (`meta`, secondary) and right values that edit
in place: Status, Priority, Milestone, Due, Remind, Branch, Pull request (opens the browser).
Notes: multiline, placeholder "Add notes…". Then `ACTIVITY`: the report timeline, newest
first — a 6 px dot on a thin vertical line, agent name and relative time in `meta`, summary in
`body`, next step in secondary with a "Next:" prefix. Edits save on blur or Enter; no Save
button. Esc closes the inspector.

**Settings.** A small sheet, not a sidebar destination: "Connect Claude Code" with the copyable
registration command, data folder path, and an About line. No shortcut settings.

### Window chrome, tray, Dock

- Tray icon: a 16 pt template glyph — a circle with a small check, outline style. When any
  agent task is blocked, the glyph shows a filled dot.
- Dock icon visible. Clicking it shows the main window.

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
