# M4d: restraint — one title, one input, one list

User feedback on M4c (2026-10-15, screenshot of the project board in Chinese): "这个界面，我觉得很混乱，
我们的产品，在保证功能情况下，交互要简单，直接，好看". Approved direction: `docs/design/mockups/v3.html`
(render `v3.png`). Every screen shows one title line, one input, one list; everything else appears
when it is needed (hover, right-click, the inspector). Functions stay; only their surfaces move.

## Header (projects, Inbox, Now, Completed)

- Projects: the name in `title`, and at the right one small icon segmented control, list ☰ / board
  ▥ (30 × 26 pt cells, icons only, tooltips "List" / "Board").
- Removed from the header: the repo slug, the "Done 5 · Backlog 2" text, and the milestone menu
  when the project has no milestones. With milestones, the filter stays as quiet text before the
  view control ("v0.3 ⌄" / "All milestones ⌄"); "New milestone…" moves to the project's sidebar
  context menu (and stays in the filter menu when it is shown).
- The repo slug moves to the project's sidebar context menu as a first item "Open QingYunA/adhd-todo on
  GitHub ↗" (only with a slug) and to the inspector's top line ("adhd-todo · QingYunA/adhd-todo · #7").
- Now keeps its date in `text.tertiary`; Inbox keeps no subtitle.

## The input

- At rest: one line, 50 pt tall, 12 pt radius, `bg.subtle`, **no border, no shadow, no chip**; a
  small accent ✦ at the left; placeholder in `input` (17/25) `text.tertiary`; `⌘K` in
  `text.tertiary` at the right.
  - Placeholders: Now "Tell me anything — I'll turn it into tasks" / "告诉我任何事，我来变成任务";
    project and Inbox "Add a task, or ask me anything" / "添加任务，或者问我任何事"; inspector
    "Change anything about this task" / "说说要怎么改这个任务".
- Focused or with text: background `bg`, 1 pt accent border at 50 % and a 3 pt ring at 10 %;
  grows with its text up to 6 lines; the ✦ and ⌘K give way to a 28 pt round send button at the
  right (accent with text, hidden when empty). No hint text at all.
- No model connected: the same field; Enter adds a plain task here (as now). The only extra is the
  existing quiet "Set up the assistant" line under the field while it is focused and empty.

## The answer

- The user's own words are no longer echoed. No divider.
- A turn that changed something shows **one line** under the input: green ✓, "Added **title** ·
  Fri · P1" (or "Moved …", "Marked done …", several changes: "Added 2 tasks"), and "Undo" at the
  right in `accent`. It fades out after 6 s (fade only with Reduce Motion), or when the user types
  again. Several receipts in one turn: one line per task, at most 3, then "and 2 more".
- A turn that changed nothing: one grey line with a small ✦ and the assistant's sentence; fades the
  same way.
- A question stays until answered: the question in `body`, the options as tinted buttons below.
- An error stays with "Retry".
- ⌘Z undoes the assistant's last change while its line is shown or the conversation is live
  (and the menu's Delete keeps its own Undo toast).
- The inspector's input uses the same one-line answers above itself.

## Lists

- Priority: rows and cards show only P0 and P1, as a small outlined text badge ("P0" in
  `status.blocked`, "P1" in `text.secondary`); P2 and P3 show nothing. The bars glyph is removed
  from rows and cards (the inspector and the task menu keep showing all four levels as text).
- Groups as now; Done expanded (3 rows, then "Show all N"); Backlog folded on the last line
  "Backlog 2 — show".

## Board

- Four fixed lanes: To do, Doing, In review, Done. Blocked cards sit in Doing, first, with the red
  blocked glyph in their meta line. Backlog and cancelled tasks are not on the board (the list
  shows them). Done shows the newest 5 and "Show all N".
- Lanes have **no background**; a lane is its header (glyph, name, count) and its cards, 20 pt
  apart. An empty lane says, in `text.tertiary`, "Drag here to start" (Doing), "Nothing to review"
  (In review), "Nothing yet" (To do, Done).
- Cards: white, 10 pt radius, a 1 pt `border` ring and a 1 pt soft shadow; title in `body.strong`;
  one meta line: P0/P1 badge, branch (mono, ellipsized), and at the right the due label or the age.
  No status glyph on cards (the lane says it). Done cards: `bg.subtle`, no ring, title in
  `text.secondary` weight 400.
- No "+" in lane headers: the input above adds tasks. Dragging between lanes changes status as now.
- The board shares the content block's left and right edges with the header and the input.

## Everything else

- Apply the same restraint wherever it applies: no duplicated labels, no hints that repeat what a
  control already says, no boxes around things that space can separate.
- Keep: all functions (menus, search, Completed, settings window, inspector values, milestones).

## Done when

- The surfaces match v3.png in light/dark, English/Chinese; ui-spec updated; screens regenerated and
  reviewed; tests, vet, gofmt, `mygo i18n check` clean; MCP unchanged.
