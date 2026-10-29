# Withdo UI spec

The app answers one question at a glance: **what is happening right now, and what needs me?**
Every screen is judged against that question. The audience is a developer with ADHD who runs
several coding agents in parallel. Visual noise costs them focus, so the UI is quiet by
default and loud only where something needs attention.

Since v31 (2026-10-21) the audience is wider: anyone who wants an agent to keep their to-dos, with
coding agents as one thing it can show and hand work to. The way in is one input. See **Input first
(v31)**; where it disagrees with an older section, it wins until that section is rewritten.

Built with mygo native UI (`github.com/egoist/mygo/ui`, v0.5.1). No webview.

M4d (2026-10-15) made every screen **one title, one input, one list**: what is not needed to read or
act appears on hover, in a right-click menu or in the inspector. The reference for it is
`docs/design/mockups/v3.html` (rendered: `v3.png`); `docs/plans/m4d-restraint.md` is the plan.

## Names in code

Where the name in the UI and the name in code differ (grep for the code name, not the UI one):

| UI | Code | Screens |
|---|---|---|
| Today / `今天` | `Mine`: `page.Mine`, `screenMine`, `NavMine()`, `app/views/mine.go` | `mine-*` |
| Backlog / `待规划` | `domain.StatusBacklog`, message `statusBacklog` | |
| In review | `domain.StatusReview` (`"review"`), message `statusReview` | |
| No project | the Inbox: `data.PlaceInbox`, "Inbox" in `internal/domain` comments | |

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

### Craft rules (2026-10-26)

The owner's rules for every screen, first applied in `docs/design/mockups/v34.html`. Where a token or
a section below disagrees, these win until it is rewritten.

1. **No middle dot as a separator.** Never join pieces of copy with `·`. Use a line break, space,
   weight or color, or say it as one phrase.
2. **No explanatory text in the UI.** A label, a placeholder or the control itself must carry the
   meaning. Hints, captions and "how this works" lines cost attention and dirty the screen.
3. **Copy fits its reader and its place.** Plain words for the audience on that screen, not chatty and
   not invented. Length fits the job: a button is a verb, a header is a noun, a status is a short
   phrase.
4. **Radix Colors only.** Every color is a step of a Radix UI Colors scale (neutral `slate`, accent
   `iris`, status `red`, `amber`, `orange`, `green`, `blue`; dark mode uses the dark scales), used as
   Radix intends:
   - steps 1 and 2 for backgrounds;
   - 3, 4 and 5 for an element's rest, hover and pressed or selected fills;
   - 6, 7 and 8 for borders (subtle, element, focus);
   - 9 and 10 for solid fills;
   - 11 for secondary text and 12 for primary text.
5. **Optical alignment and concentric corners.** Align what the eye compares: text with text, icons
   with glyphs, not boxes with boxes. A rounded thing inside a rounded thing has radius = outer radius
   minus the inset.
6. **Lines are 0.5 pt.** Every border, divider and hairline.
7. **Little motion.** Animate only what explains a change. When something must move: ease-out,
   250 ms as the base.
8. **At most four text sizes, better three; none under 12 pt.** Build structure with weight, color and
   space before adding a size.
9. **Few decisions.** Do not put choices on the page that the user did not come to make. When a
   choice is needed, put it where the user already decides (a menu, the moment of a drag, Settings).
   Rank what a page shows and order it; remove what does not earn its place.
10. **Corners and shadows with purpose.** No rounding or shadow by habit, and never a colored stripe
    on a card's left edge. A radius or shadow must fit what the thing is: floating surfaces get a
    shadow, flat content does not.
11. **No decorative eyebrows.** No small label or uppercase overline above a title unless it adds
    information the title does not. Build hierarchy with size, weight, spacing and white space.

## Input first (v31)

Decided 2026-10-21. Reference: `docs/design/mockups/v31.html` (rendered: `v31.png`). The feature set
stays; the way in changes. People should not fill forms: they say or type one sentence, the
assistant turns it into tasks, and they fix what it got wrong by clicking a chip. Like FocusPomo,
one thing on the home screen is the main character, and finishing something feels good. Unlike it,
there is no timer.

Each decision lists the section it replaces. Each is built by its own pull request, which rewrites
that section in detail. Until then the older text describes the app as it is.

1. **One big input at the top of home.** Home (`Today` / `今天`, the page that was Mine) opens with the date
   and a greeting, then a large input (58 pt, 14 pt radius, `bg` with a hairline and a soft shadow,
   accent ring when focused) with a pixel microphone at its right, and a hint line under it. The
   lists follow: Today, Later, Done today. The Agents panel sits in a narrow right column (300 pt,
   `bg.tray`) titled `What your agents are doing`, with the drop target `Drag a to-do here to hand
   it to an agent` at its foot; dragging a row onto it hands the task over, as today. Replaces the
   bottom-pinned composer of **Mine (home)** and **Composer** on Mine. Built: **Mine (home)**.
2. **Project pages keep a small input; other pages lose theirs.** A project keeps the bottom
   composer, and a sentence typed there goes to that project. Agents and Calendar have no
   composer any more; the inspector keeps its own. Replaces the list of six places in **Composer**. Built: **Composer**.
3. **Applied at once, with Undo.** A sentence is applied as soon as it is sent, never shown as a
   preview to confirm. New rows land with the chips the assistant read (date, reminder, category,
   project, importance) and a light `bg.selected` wash; a toast says what happened (`Added 1, handed
   1 to Claude`) with **Undo** and `⌘Z`. A chip opens the matching piece of the **Task menu**. Keeps
   **Undo** and **Assistant**'s receipts.
4. **Voice goes through the official service.** The microphone (or holding `⌥ Space`) records
   while held or until clicked again (60 s at most). The audio goes to our server, which sends it to a
   cloud speech recognizer (MiMo `mimo-v2.5-asr`, Chinese and English with auto-detection) and
   returns the text. The server holds the recognizer's key; the app never does. The text fills the
   input and is not sent: the person edits it and sends it. The recognizer takes a whole recording, so there are no words while
   speaking: the microphone is a filled accent square and a pixel waveform follows the voice's level
   on the 100 ms loop beat, then `Transcribing…` until the text arrives. Esc cancels. Voice counts
   against the account's quota; signed out, the microphone says `Sign in to use voice`. No text to
   speech. Decided 2026-10-21 over the system recognizers: macOS's `SFSpeechRecognizer` refuses
   while Dictation is off, and one service gives the same result on every platform.
5. **Understanding comes from the official service.** Signing in (Google or email) gives the input
   its model, with a Free and a paid tier. Without signing in, or past the quota, the input still
   adds tasks: local rules read dates and times, `#category`, `!` (important), and split a sentence
   at `，还有` / `, and`; one quiet line under the receipt offers to sign in, never a dialog. Bringing
   one's own API key moves to Settings > Advanced. Replaces "With none, Enter creates a plain task"
   in **Mine (home)** and the setup flow of **Settings** > Assistant.
6. **Menu bar icon and `⌥ Space`.** On macOS a pixel icon in the menu bar opens a popover (mygo
   `NewTray` + `NewFlyout` with `Popover`): the same input, up to five of today's rows (a click opens
   the task in the main window), and one footer line, `2 agents working · 1 stuck` (the stuck number
   in `status.blocked`, the only strong color there) and `Open Withdo`. `⌥ Space` anywhere opens
   it with the input focused; holding it listens. On Windows the tray icon's left click opens the
   same panel. Replaces "No flyout panel, no global shortcut" and "macOS: no tray icon" in
   **Surfaces**.
7. **Four sidebar entries.** `Today`, `Agents`, `Calendar`, `Summary` (back since v36), then Projects and the gear.
   - Inbox goes into Today: a task with no project lands in Today or Later by its date.
   - Completed goes into each list's `Done` fold, which opens the full history; search finds it too.
   - Summary opens from its sidebar row (v36), from `See summary ›` beside Done today, and from the end-of-day notification.
   - The category sub-rows go: a category is a colored chip on the row; clicking it filters the
     list (a dismissible filter bar above it).
   - The agent sub-rows go: the Agents page groups by agent.
   - Recent changes moves to Settings > Advanced (it is off by default already).
   Replaces the sidebar of **Window**, **Inbox**, **Completed** and the sub-rows and folding.
   Built: **Window**, **Mine (home)**, **No project**, **Completed**, **Categories (M9)**.
8. **My tasks have three states and a star.** For the user's own tasks, Status is a segmented
   control: To do, Doing, Done. Backlog and Cancelled stay in the right-click menu. Priority becomes
   one **important** star (12 pt pixel star, `status.review` color when set; set = P1, unset = P2 in
   the database, so agents and the MCP keep P0 to P3). Agent tasks keep all seven statuses and
   their priority badges, since those are the agent telling you what happened.
9. **The inspector by source.** A task of mine shows: title with the star, Status, `When` (due and
   reminder chips), `Where` (category and project · milestone chips), notes and images, and `Hand
   to an agent…`. Branch, Pull request, From and Activity show only on an agent's task. Replaces
   the property list of **Inspector**.
10. **Settings: four panes.** Account (sign-in, quota, devices: the old Sync pane), General
    (language, notifications, menu bar icon, data folder, archived projects, updates, about), Agents
    (with the old GitHub pane), Advanced (own API key with the twelve providers folded, Recent
    changes, end-of-day reminder, sound effects). Replaces the panes of **Settings**.

## Tokens

### Color

Radix UI Colors (v34, Craft rule 4): neutral `slate`, accent `iris`, status `blue`, `red`, `amber`,
`green`, `orange`. Dark mode uses the dark scales. The step is named in brackets.

| Token | Light | Dark | Use |
|---|---|---|---|
| `bg` | `#FFFFFF` | `#111113` (slate 1) | window content |
| `bg.sidebar` | `#F9F9FB` (slate 2) | `#18191B` (slate 2) | sidebar, with a 0.5 pt `border` at its right |
| `bg.subtle` | `#F0F0F3` (slate 3) | `#212225` (slate 3) | wells: search, finished cards, the view control, the idle send button |
| `bg.tray` | `#F9F9FB` (slate 2) | `#18191B` (slate 2) | the Agents tray on Home, the Agents column of a project |
| `bg.hover` | `#F0F0F3` (slate 3) | `#212225` (slate 3) | row hover |
| `bg.selected` | `#E8E8EC` (slate 4) | `#272A2D` (slate 4) | selected row, keyboard highlight, the sidebar's current page, a highlighted menu item |
| `border` | `#D9D9E0` (slate 6) | `#363A3F` (slate 6) | hairlines, always 0.5 pt |
| `border.control` | `#CDCED6` (slate 7) | `#43484E` (slate 7) | the composer's edge, controls |
| `text` | `#1C2024` (slate 12) | `#EDEEF0` (slate 12) | primary |
| `text.secondary` | `#60646C` (slate 11) | `#B0B4BA` (slate 11) | every grey word: metadata, summaries, placeholders, section headers |
| `text.tertiary` | `#8B8D98` (slate 9) | `#696E77` (slate 9) | icons and glyphs only, never words (under 4.5:1) |
| `accent` | `#5B5BD6` (iris 9) | `#5B5BD6` (iris 9) | the send button, focus ring edge (iris 8), the drag target |
| `accent.soft` | `#F0F1FE` (iris 3) | `#202248` (iris 3) | tinted buttons, a just-added row, the focus ring |
| `accent.ink` | `#5753C6` (iris 11) | `#B1A9FF` (iris 11) | links, tinted button text, `Undo` |
| `status.backlog` | `#B9BBC6` (slate 8) | `#5A6169` (slate 8) | |
| `status.todo` | `#8B8D98` (slate 9) | `#696E77` (slate 9) | |
| `status.doing` | `#0090FF` (blue 9) | `#0090FF` (blue 9) | |
| `status.blocked` | `#E5484D` (red 9) | `#E5484D` (red 9) | the only red in the app |
| `status.review` | `#FFC53D` (amber 9) | `#FFC53D` (amber 9) | also the star |
| `status.done` | `#30A46C` (green 9) | `#30A46C` (green 9) | |
| `stale` | `#CC4E00` (orange 11) | `#FF801F` (orange 11) | stale label text |

Category colors and project folder colors take step 9 of their Radix scale for icons and step 11 for
words. Selection is neutral: the accent is kept for actions and links, so it does not compete with
the composer.

Follow the system appearance (light/dark) and the system accent only if the user changed it
from the default; otherwise use `accent`.

### Type

System font (SF Pro), SF Mono for branches and PR numbers. Use at most three sizes in one view,
four on Home where the composer has its own (Craft rule 8).

| Token | Size / line | Weight | Use |
|---|---|---|---|
| `title` | 24 / 32 | 600 | page title (v34; was 26 / 700) |
| `input` | 17 / 24 | 400 | the composer |
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
  24 pt) in the rows of **On this computer**, 18 x 18 (drawn at 18 pt) in the compact rows of **More**.
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

**A task of mine shows the star instead** (v31): the 12 pt pixel `IconImportant` in `status.review`, on the
same places as the badge, when it is P1 or P0 (the star reads P0 as set); never a `P0` or `P1` badge. An
agent's task keeps the badges.

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
| `IconCloud` (M11) | `cloud` | `IconLaptop` / `IconPhone` (M11) | `laptop` / `smartphone` |
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
- **Sidebar**: Today is the pixel you (`theme.You`) and Agents the robot in the accent color
  (`theme.AccentBot`), both 18 pt, the size of the icons of the other rows, so the labels stay in line;
  Calendar `IconCalendar` (and Recent changes `IconBranch`), 18 pt in `text.secondary`;
  a project is `IconFolder` tinted `theme.ProjectColor(name)` (the color follows the name, the user
  does not choose it). A category chip's icon is the category's 12 x 12 icon in its own color; an
  agent's group header on Agents has its 18 pt avatar.
- **The drawn task menu** (and its Move to and reminder pages) puts an 18 pt pixel icon before each
  item: Move to `IconMove`, Remind me `IconClock`, Open PR `IconArrowOut`, Ask `IconSparkle`, the
  reminder presets `IconClock` / `IconCalendar`, No reminder `IconClose`, Custom `IconPlus`, a
  project `IconFolder` in its color (white on the highlighted row), Delete `IconTrash` in
  `status.blocked` with its text. The project and milestone menus are drawn too (below), text only.
- **Empty states** (Today, Agents, a project, the history, search with no hits): `theme.DuoIcon` at 60 pt
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

One window, and a small panel under an icon in the menu bar or the tray (decided 2026-10-21, v31,
`docs/design/mockups/v31.html` section 6; it replaces the 2026-10-14 decision of no flyout panel, no
global shortcut and no macOS tray icon). Closing the window hides it, so reminders, the end of the
day and pull request news keep coming, and the panel works while the window is hidden; Quit quits.
A second launch starts no second process: it brings the window back (one instance on both systems).

- **App icon** (decided 2026-10-27, v40 C, `docs/design/mockups/v40.html`; drafts v37–v39). A pixel
  check on the navy square (`#1E293B`), on a 24 x 24 grid with a brush three cells square: the short
  stroke in iris (`#B1A9FF`, the color of you) is the person deciding what to do, the long stroke in
  white (`#F8FAFC`) is the agent getting it done, and the vertex belongs to the long stroke.
  `docs/design/tools/app_icon.py` draws `app/resources/icon.png` (keeping the shape of the square,
  28 px a cell) and the Android foreground (3 dp a cell); `docs/design/tools/tray_icon.py` draws the
  12 x 12 tray picture with the same two colors.
- **Menu bar icon (macOS).** The app icon redrawn on the 12 x 12 grid (`app/theme/sprites/tray.png`)
  as a template image: the dark cells black and the check mark cut out (`theme.TrayTemplateIcon`,
  36 x 36 px, which the menu bar draws 18 pt tall), so it follows the menu bar's light or dark
  appearance. Tooltip `Withdo`. A click opens the panel; the icon has no menu (the Dock has
  `Settings…` and Quit).
- **The panel** (`app/views/popover.go`, `app/panel.go`). mygo `NewFlyout` anchored to the icon
  (`Tray`, `PlacementBottom`, `Focusable`) with `Popover` on macOS: an `NSPopover` draws the material,
  the corners, the shadow and the arrow, so the content draws no background there
  (`ui.Context.Vibrancy`); on Windows it draws `bg` and the system shadow. It is native UI like the
  window, 360 pt wide and as tall as its content (12 pt padding), top to bottom:
  1. the **composer** of Mine (`composerBox`, `theme.Input`), focused as it opens: the same input,
     reply, receipts, Undo and plain-task confirmation as Mine's, filing into Mine
     (`data.PlaceNow`). It is its own `Main`, so its draft and reply are its own, and a draft survives
     closing the panel. Esc cancels a running request, then closes the reply, then closes the panel.
  2. under it `Today` / `今天` with its count and up to five of today's rows from Mine (the Today
     section, in its order): status glyph (ticks the task off), title, due label. A click opens the
     task in the main window (brought to the front, inspector open) and closes the panel. Nothing is
     edited in the panel. With no row of today the header is not drawn.
  3. a hairline and one footer line: `2 agents working · 1 stuck` / `2 个 Agent 在干活 · 1 个卡住`
     at the left, `text.tertiary`, the stuck count in `status.blocked`, the only strong color there
     (working = agent tasks in doing, stuck = blocked; a part with a zero count is left out, and with
     both zero the left is empty), and `Open Withdo` / `打开 Withdo` at the right as a link,
     which brings the main window forward and closes the panel.
  Esc or a click away closes it; the icon's own click toggles it.
- **Global shortcut `⌥ Space`** (`Alt+Space` on Windows), `mygo.GlobalShortcut`, registered when the
  app is ready: toggles the panel with the composer focused, wherever the user is. If another app
  holds the keys the registration fails, the app logs it and carries on; the icon still works.
- **Windows** has no Dock, so the app has a tray icon there (decided 2026-10-20, v28,
  `docs/design/mockups/v28.html`): the same sprite in color (shown 2x as 24 x 24 px), tooltip
  `Withdo`. A left click opens the panel under the icon (above it with the taskbar at the bottom
  of the screen: mygo flips it); a right click opens its menu: `Open Withdo` / `打开 Withdo`,
  `Settings…` / `设置…`, a separator, `Exit` / `退出`. The menu bar's File menu holds `Settings…`
  (Ctrl+,) and `Exit`, since Windows has no app menu.
- **Not in the panel (yet):** the microphone and holding the shortcut to listen are decision 4 of
  **Input first**, built by their own pull request.

The high-fidelity reference for this section is
`docs/design/mockups/now.html` (rendered: `now.png`). Match it.

### Window (1100 x 720, min 860 x 540)

Unified title bar. Columns: sidebar (216 pt) | content | inspector (360 pt, only when a task
is open). Content is a centered column, max 780 pt wide, 48 pt side padding, so lists read
like a document, not a spreadsheet.

**Sidebar.** A search field first (see **Search**), then four entries, `⌘1` to `⌘3` for the first three:
`Today` / `今天` (the page that was Mine: a plain count of the rows from your agents plus Today; when the
user has not seen what agents handed over, a 18 pt red badge with their number instead, `status.blocked`
with white `badge` text), `Agents` / `Agent` (count = tasks at work, never their subtasks) and
`Calendar` / `日历` (no count) and `Summary` / `总结` (v36: `chart` pixelarticon, no count, no key; it opens the Summary page on today). Then `Projects` (sentence case, `section` style) with a quiet `+` at the
right of its header: one row per project (a folder icon tinted with the project's color, name only),
**on one line, ellipsized** (`answer-me-with-ht…`), its sub-projects under it (see **Sub-projects**); a
6 pt red dot at the right when it has a blocked task, otherwise the count of active tasks in
`text.tertiary`. A project with no active task shows no count. Settings is a gear button at the bottom.
Rows are 32 pt tall in `body` 14 pt (the sidebar is one step smaller than the page); selected row:
`bg.selected`, 7 pt radius. With **Recent changes** switched on (Settings > Advanced) its row follows
Calendar, `⌘4`.

There are no other rows, no sub-rows and no folding chevrons (v31; `Summary` returned in v36). What the old rows were:

- **Inbox** is Today: a task of mine with no project shows in Today or Later by its date (see **No
  project**).
- **Completed** is the `Show all` link at the end of every list's Done section (see **Completed**).
- **Summary** opens from `See today` beside Done today on Today, and from the end-of-day notification
  (see **Summary**).
- **Categories** are a colored chip on a row; clicking it filters the list (see **Category**). They
  are made, edited, moved and deleted in Settings > General > Categories (see **Categories (M9)**).
- **Agents** are the headers of the Agents page, one group of rows per agent (see **Agents**).

**Category.** A task is in one category or none (`tasks.category_id`, NULL for none; migration 012;
before M9 only `work` and `life`, migration 007).
The category it counts under is its own, else Work when its project is a repository, else none
(Unsorted): a category set by hand or by an agent always wins. The assistant classifies the tasks it
writes from the words and leaves it out when unsure or when the task goes into a repository's project.
A task typed in the big input of Today while a chip's category narrows it (the filter bar below) gets
that category, with or without a model.

**The chip and the filter bar.** A row of mine whose category is not none ends its title with a chip
(v31): the category's 12 x 12 icon in its color and its name, in `meta` `text.secondary`, on
`bg.subtle`, 6 pt radius, 2 pt above and below the text and 7 pt at the sides, tooltip and a11y label
`Show only Work` / `只看 工作`. It shows on Today and, for a category the user or the assistant gave, on a
project's list and columns (the Work a repository's project implies is not repeated on every row of
its page); not on an agent's row, and not under 460 pt of content, where the pills go too. Clicking
it narrows the page to that category: the same sections built from the tasks that count under it
(for Today by the same builder as the page, so counts, `Show all` and the Backlog fold are the
slice's; for a project by its list filter). Above the lists a **filter bar** appears: a pill on
`bg.selected` with the icon, `Showing Work` / `只看 工作` in `meta` and a 20 pt `close` button (tooltip
`Clear filter` / `清除筛选`). The `x` and `Esc` (after the inspector, before the selection) clear it;
going to another page clears it too; deleting the category does. The filter is `Main.catFilter`; it
is not kept. Screens `mine-filter-*` (Work, and `mine-filter-life-*`), `agents-codex-*` (two groups), `project-history-*`.


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
  the old name back. Double-clicking the project's title on its page renames it the same way, in place
  (the title becomes an input; the other pages' titles do nothing). Archive acts at once, with the project's sub-projects (the project is in Settings >
  General > Archived projects, one click from coming back); Delete… asks first (**Delete confirmation**).
  Archiving or deleting the project on screen goes to Today.
- The menu is drawn in the window (mygo `PopoverMenu`) with the theme's colors, like the Task menu, not
  the system's; it opens at the pointer, and arrows, Enter and Esc work in it.
- The repository is not on the project's page: it is in this menu and in the inspector's top line.

### Mine (home)

*Decision 1 of **Input first (v31)** is built here; items 3 and 5 still replace parts of it.*

**v34 (2026-10-26), reference `docs/design/mockups/v34.html`.** Approved by the owner; it wins over
the paragraphs below wherever they disagree, until they are rewritten.

- **The page reads top to bottom:** the date as the title, the composer, then two columns: the
  lists, and the Agents tray at their right.
  - **Title.** The date, left-aligned at 24 / 600: `10月26日 星期一` / `Monday, October 26`.
    There is no greeting and no hint line.
  - **Left edge.** The title, the section headers and the status glyphs share one left edge,
    14 pt in from the column.
  - **No `Select` control.** `⌘`-click and `Shift`-click still select several rows.
- **The composer is the page's main surface** (the owner: "everything starts in this box"):
  - **Size and place.** It spans the whole content width, over both columns. It is two rows tall:
    16 pt above the text, a 20 pt gap, then a row of controls, 8 pt from the edge.
  - **Radius and surface.** Radius 16. It is the page's only shadowed surface: a 0.5 pt `slate 7`
    edge and `0 12 32 -12` at 18 % black.
  - **Text.** 17 / 24; no other text on the page uses this size.
  - **Left.** The 24 pt pixel assistant avatar.
  - **Placeholder.** One real example, not an explanation: `明天 9 点给房东打电话，再让 Claude
    修一下登录的 bug` / `Call the landlord tomorrow at 9, and have Claude fix the login bug`.
    One errand of mine, one hand-off.
  - **Control row.** Image and microphone (32 pt, radius 8, `slate 11`), aligned under the text.
    At the right: the key `⌥ Space` in 13 pt `slate 11`, and the send button (32 pt, radius 8).
    The send button is `slate 3` while empty and `iris 9` with a white arrow once there is text.
    It is the only solid button on the page.
  - **Focus.** The edge turns `iris 8` with a 4 pt `iris 3` ring.
  - **No model chip.** The model is chosen in Settings.
- **Answers under the composer.** The receipt and every other line the composer says back sit
  under it, 12 pt below, in 13 pt `slate 11`, with `Undo` / `撤销` in `iris 11`. Example:
  `已添加 1 件，交给 Claude Code 1 件`. A row the input just made gets an `iris 3` fill that fades
  out after 2 s (250 ms ease-out).
- **Live preview (later, a separate change).** While typing, the composer lists what it will
  make under the text, each with its glyph or agent avatar and where it goes (`明天 9:00`,
  `交给 Claude Code`). It needs the model to parse while the user types, so it ships after the rest.
- **The lists, by time only:**
  - **Section headers.** 13 / 600 `slate 11`, with the count in regular weight.
  - **Today.** Unchanged.
  - **Next** / `接下来`. Replaces Later. Only my tasks dated within the next 7 days, by date.
    Nothing undated, no cap, no `Show all`.
  - **No date** / `没定时间`. A fold, closed by default, with its count. It holds my undated
    tasks **with no project**, including those in the backlog. It replaces the Backlog fold.
    When the input adds an undated task, the fold opens and the receipt names it:
    `已添加到没定时间`.
  - **Done today** / `今天完成`. At the right of its header, `看总结 ›` / `See summary ›` (`iris 11`,
    a 12 pt pixel chevron after the words, v36) opens the Summary; `See today` and `Show all` are gone. A row an agent finished shows the agent's
    avatar and what it did (`PR #335 已合并`) before the time.
  - **Undated tasks of a project** are not on Home. They stay on their project page. To bring
    one in, use **Do today** in the task menu, or say it to the composer.
  - **Dated beyond 7 days.** On the Calendar; the `After next week` fold is gone. A task
    shows in Next by itself once it is within 7 days.
- **Rows.** 40 pt tall, radius 8.
  - Doing is shown by its glyph alone; the row does not say `进行中`.
  - A category is its name in its own color (`green 11`, `blue 11`), with no fill and no dot.
  - The star sits before the date.
- **The Agents tray:**
  - **Tray.** `slate 2`, radius 12, 4 pt inset, a 0.5 pt `slate 6` edge. Its header is the bot
    avatar, `Agent` and the count.
  - **Cards.** One per row, on `bg`, radius 8 (12 − 4), a 0.5 pt `slate 6` edge, 4 pt apart.
  - **Idle (v36, reference `docs/design/mockups/v36.html`, "推荐 A").** With no agent task the header
    shows `空闲` / `Idle` in 13 `slate 11` where the count would be, and under it one card in the
    style of the others (14 pt above, 12 pt at the sides and below). Left: the robot asleep,
    `avatars/bot-asleep.png` at 36 pt (3 pt a cell, the gray template with its eyes shut), and two
    pixel `z` letters (5 x 5 cells at 2 pt, `slate 9`) at the upper right of its head. They rise 4 pt
    in steps (2 pt every 500 ms) over a 2 s loop, shown for the middle two steps only, the second
    half a loop behind; with Reduce Motion both stand still. Right: `都在歇着` / `All resting` at 15 / 500, and
    under it the 14 pt avatar of the agent that reported last and `Claude Code · 12m` (13 `slate 11`;
    left out when no agent ever reported). A 0.5 pt `slate 5` hairline 12 pt below, then
    `把一件事拖到这里交给它们` / `Drag a task here to hand it off`, `拖到这里` / `Drag a task here`
    in 13 / 500 `slate 12`, the rest in `slate 11`. While a row is dragged the drop target takes
    the card's place.
  - **Copy.** Ages say `4 天前` / `17 分钟前`; stale ones are in `orange 11`. Reasons are short:
    `等你回复：…`, `PR #330 待审查`.
  - **One tinted button per card** (`iris 3` fill, `iris 11` text, radius 6): `回复`,
    `打开 PR`. A second action is a text action in `slate 11`: `标为完成` (it was `收下` / Accept).
  - **The drop target shows only while a row is dragged.** It is an `iris 2` well with a
    0.5 pt `iris 8` edge, radius 8, saying `交给 Agent`.


Home is the page `Today` / `今天` in the sidebar (it was `Mine` / `我的`; the code and the screens
`mine-*` keep the old name). It opens with one big input, not a title. Reference: `docs/design/mockups/v31.html` (the proposal frame), screens `mine-*`.

**The hero.** At the top of the content column, scrolling with the page (it is not pinned):
1. the day in `meta` `text.secondary`, centered: `Thursday, October 23` / `10月23日星期四` (`words.Clock.HeroDate`,
   the long month);
2. the greeting in the page's `title` size, centered: `What do you want to do today?` / `今天想做点什么？`;
3. the **big input**, 18 pt under the greeting, centered in at most 720 pt (`theme.HeroMax`): the
   **Composer** itself (see **Composer**, `composerBoxAs` with `hero`), not another control: the same
   field, local rules or assistant, model chip, `+` images, pixel microphone, send and stop, answer
   line and receipts. In hero size it is 58 pt tall at rest (`theme.HeroHeight`), 14 pt radius
   (`theme.HeroRadius`), `bg` with a hairline `border` and a soft shadow (0 2 10, `shadow` at 8 %); focused,
   holding text or working, the border is `accent` at 50 % with the 3 pt ring at 10 % (the shadow gives
   way to the ring; nothing else moves, the steady-look rule of **Composer**). The text is in `input`
   (17 / 25), the theme has no larger style. It has no `⌘K` at its right: the hint says the keys;
4. the **hint line**, 10 pt under the box, centered, in `meta` `text.tertiary`: `Return to add · ⌥ Space from any
   app · One sentence can hold several things` / `回车添加 · 在任何 App 里按 ⌥ Space · 一句话里可以有好几件事`
   (`Alt+Space` on Windows). Holding to talk is not built, so the hint does not say it. What the input
   says back shows **in the hint's place, under the box**, and the hint returns when it goes: the
   answer, the receipt with Undo, the voice line, a picture problem, and the `Set up the assistant`
   line (`inputLineLive` says whether one is showing). The page slides under it as the line comes and
   goes (`belowComposer`).

The `Select` control stays at the top right of the page, on a 36 pt row of its own over the hero. `⌘K` focuses the big input from anywhere (and from a page with no composer it goes to
Home first, see **Composer**); `↓` from the input walks into the rows, and `↑` from the first row
comes back to it. On Mine the composer is no longer pinned at the bottom: there is no dock
(`dockHeight` is 0). The menu bar panel keeps its own composer.

**Two columns, or one.** Under the hero the page has the lists (Today, Later, Done today, below)
and, when there is room, the **Agents panel** at their right.
- The panel is 300 pt wide (`theme.AgentsPanelWidth`), 24 pt right of the lists (`theme.MineGap`), on
  `bg.tray`, 14 pt radius like the project tray, 14 pt above and 16 pt around its content. It is titled
  with the 18 pt bot avatar, `What your agents are doing` / `Agent 在做什么` in `body strong`, and the number
  of rows at the right in `meta` `text.tertiary`. Under it are the rows of the section **From your agents**
  below, the same row views and buttons (Answer, Open PR, Accept, Not now ...), laid out as in a narrow
  window: 268 pt, buttons under the summary, no project name; the age (`25m`, `3d`, in `status.stale` orange when
  stale, as in the inline section) sits at the right of the title line, which ellipsizes before it. Nothing handed over:
  one quiet line, `No agent is working on anything` / `没有 Agent 在干活`. At its foot, 14 pt under the
  last row, is the **drop target**: a box with a dashed 1 pt `border`, 10 pt radius, 12 pt arrow icon and
  `Drag a to-do here to hand it to an agent` / `把左边的待办拖到这里，交给 Agent` in `meta` `text.tertiary`,
  shown also when the panel is empty. Dragging one of my rows over the panel tints it with
  the accent (as the tray of a project) and the box turns accent; dropping it hands the task over exactly
  as the tray of a project does (see **Handing a task to an agent**: the task becomes the agent's, a toast
  with Undo, a stack of selected rows goes together). An agent's row, or a finished one, does not hand over.
  Today and Later rows of Home drag only while the panel is shown (they lift and leave a dashed slot
  as in a project).
- **Breakpoint.** The panel is drawn when the content area (the window less the sidebar and the open
  inspector) is at least **880 pt** wide (`theme.MineSplitBelow` = 2 x 24 pad + 508 for the lists + 24
  gap + 300 panel). The default 1100 pt window has 883 pt of content, so the panel shows; with the
  inspector open it does not. The page's side padding is 48 pt when the 1104 pt block (780 lists + gap +
  panel) fits, and gives down to 24 pt (`theme.MinePad`) before the panel goes. Narrower, the panel is
  not drawn and **From your agents** shows inline at the top of the lists, as before, and
  the lists keep the 780 pt column of the other pages.

Then up to four sections (the first one is the Agents panel, in a wide window). **A section with nothing
in it is not drawn at all** (the panel is: it always shows, empty with its drop target). Subtasks never get
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
2. **Today** / `今天` — the user's tasks (a user task that is blocked, doing or **in review**, too): due or
   reminded today and overdue first by time, then what is in progress. My own task in review waits
   for my look, so it is here and in the count, with a small `To review` / `待审查` tag (the grey
   `bg.subtle` pill of the Agents page) before the priority and the due label. Row: the status glyph (it
   ticks the task off), title, project name, `P0`/`P1` badge, due label at the right ("2:30 PM",
   "Today"). A category chip shows only for a category set on the task (by me or the assistant): the
   Work a repository's project implies is not drawn (see **Categories (M9)**).
3. **Later** / `之后` — the user's tasks due in the next 7 days by date, then undated by priority
   (at most 7; a `Show all` link after lifts the cap in place). Tasks of mine with no project are
   here and in Today like any other, by their date (see **No project**). Ones due further away are on
   the Calendar and in their project; the ones **with no project** end Later in a fold `After next
   week 2` / `下周以后 2` (the fold line of Backlog, closed by default; opened, its rows are by due
   date). Agents' tasks and tasks with a project are not in it.
4. **Done today** / `今天做完` — what was finished since local midnight, by anyone, newest first:
   the `done` glyph, the title in `text.secondary` at weight 400, and at the right who did it, then
   the time of day: `Claude Code · PR #245 merged` or `You`, then `2:05 PM`. At most five. At the
   right of its header is `See today` / `看看今天` with a pixel chevron (the **Summary** opens), and at
   its end `Show all` / `显示全部` opens the full history of everything finished (see **Completed**).
   It does not count toward the sidebar and never makes Today "not clear".
5. **Backlog** — a fold line at the very end, `Backlog 1 — show` as a project's list has it, for the
   tasks of mine parked in the backlog that no project holds (a project's page folds its own; an
   agent's are on Agents). Opened, its
   rows are Today's rows, and the arrow keys walk them. Who finished a task: the agent that filed a report at the moment it closed (within
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
three sections are empty (Done today may still show): the empty state under the header,
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
- **Finder**: the folder. On Windows the choice is **File Explorer** (`文件资源管理器`), and the folder
  opens there.
On Windows, which has no `open`, each place is started by its command with the folder as its working
folder (the way VS Code's external terminal and GitHub Desktop do it, so a path is never quoted for
`cmd`): **Terminal** is Windows Terminal (`wt -d .`), else the Command Prompt (`cmd /c start "" cmd`);
**Editor** is the first of `cursor`, `code` and `zed` on `PATH`, run with `.`; **the agent's app**
is the command of Cursor, Windsurf, Zed or Visual Studio Code (`cursor`, `windsurf`, `zed`, `code`),
and Claude and Codex, which have no command that opens their app, go to the terminal. The words do
not change: "Terminal" names both, and the editors to install are the same.
Terminal, Editor and Finder need a folder. When opening fails, one quiet line in `meta` and the blocked
red shows under the row's reason line (`Couldn't open it: …`, `Nothing to open: the agent did not
say where it works.`, `No editor found. Install Cursor, Visual Studio Code or Zed.`); it goes away
with the next successful open or when the row is opened.

### Agents

Header: `Agents` / `Agent` in `title` + `3 working` / `3 个在做` in `text.tertiary` (nothing when
none). The page has no composer (decision 2 of **Input first (v31)**); `⌘K` goes to Home.
Frame 2 of v5.

1. **Working** / `在做` — agent tasks in doing or blocked, across all projects, **grouped by agent**
   (v31; the agent sub-rows of the sidebar are gone). Each group has a header in `body.strong`: the
   agent's 18 pt avatar, its name as the page spells it (`claude-code` and `Claude Code` are one
   `Claude Code` group; an agent with no name is `Agent`) and the number of its tasks at work in
   `text.tertiary`. The agents come in the order of their newest report, and a group's rows too.
   A row: the agent's 18 pt avatar, the title, one meta line `Claude Code · main · what it
   said; next: …` (agent, branch when the row is wide enough, summary), and at the right the live
   dot (last report under 10 minutes old) or, for a blocked task, a `Blocked` pill, then the age.
   Its subtasks hang under it on a 1 pt `border` rail that starts at the middle of the avatar:
   12 pt avatar, title in `body`, a status pill and the age. Pills are 13 pt `meta` labels on a
   12 % tint of the status color: `Done` / `完成` (green), `Doing` / `进行中` (blue), `Handed back`
   / `交回主代理` (grey, an agent in review: the main agent reviews it, not the user), other
   statuses by their own name. Cancelled subtasks are left out. A click on a subtask opens it; on
   the parent, the parent.
2. **Queued** / `排队中` — agent tasks in `todo` that have **no project** (a project's page lists
   its own), after Working: a section header with the count, then the same per-agent groups and rows
   (no live dot: they are not at work). Newest first.
3. **Done today** / `今天做完` — agent tasks finished since midnight, newest first, five at most:
   the title in `text.secondary`, `3 subtasks` / `3 个子任务` when it had any, and the time. Their
   subtasks are not rows.
4. **Backlog** — the last line, the fold of Today (`Backlog 2 — show`): agent tasks parked in the
   backlog with no project. Opened, a `Backlog` header and the per-agent groups appear above the line.
   Together with Queued this keeps every agent task reachable. These two also make the page not empty.

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

*Decision 2 of **Input first (v31)** is built here.*

One component, three places: **Mine** (as the big input at the top of the page, see **Mine (home)**), **every
project**, and the **inspector**. Agents, the Calendar, the history, Summary, Recent changes
and search have none: people add tasks from Home or a project. `⌘K` focuses the composer of the page; on a page without one it goes to
Home and focuses the big input there. On a project page it is **pinned at the bottom edge of the
content column** (the way ChatGPT, Claude and the Codex chat pane do it), not under the header: the page
scrolls above it, and the column keeps the page's width (the board's 980 pt, or 820 pt), centered, with 8 pt
above the box and 18 pt under it. The **answer line**, the "Added to …" line and an image notice sit
**directly above the box** and grow the dock upward; the page does not move, it is only shorter by the dock's
height (the page's bottom padding still applies, and the dock is a sibling under the scroller, not an
overlay, so the last row is never behind the box). The bulk bar rides above the dock. The Mine hero puts the
same lines **under** the box (see **Mine (home)**). It replaces the old ask field and the inline "+ Add a task"
inputs. Screens `mine-composer-*` (the hero), `project-composer-*`, `inspector-composer-*`. It is **one
line**: the page's title already says where a task goes, so the input has no label, no chip and no hint text
(the hero's hint line is under it, not in it).

- **At rest.** One line, 50 pt tall, 12 pt radius, `bg` with a 1 pt `border` hairline, no shadow;
  hovered, the border is a step stronger (`border` mixed 25 % toward `text`). **Focus changes only the
  border and the ring, nothing else** (focus used to swap the surface and the controls, which read as
  two controls; now only the border changes, as ChatGPT and Claude do): focused, holding text or
  working, the border is `accent` at 50 % with a 3 pt ring at 10 %. The background and the controls
  never move with focus. The left slot depends on state: no model, the assistant avatar (18 pt); a
  model and no second row, the `+` (attach); a second row showing, nothing (the `+` is in the row),
  except while the assistant works on a request from this box with an empty line, then the pulsing
  avatar. The placeholder is in `input` (17 / 25) `text.tertiary`; with a model connected the **model
  chip** at the right; `⌘K` in `meta` `text.tertiary` after it on a project page whenever the field is
  empty, focused or not (not in the inspector's, nor in Home's big input, whose hint line says the keys). The
  placeholder is short: Mine `Tell me anything — I'll turn it into tasks`; a project
  `Add a task, or ask me anything`; the inspector `Change anything about this task`.
- **With text.** The text area grows with its text, 1 to 6 lines, then it scrolls. Without a model
  `⌘K` gives way to a 28 pt round **send** button at the right, bottom-aligned (`accent` with text,
  absent while the field is empty); with one, the send (or a neutral **stop**, while the assistant
  works on a request from this box, see **Assistant**) is in the second row. Without a
  model the box keeps its 50 pt from rest to one line of text; with one, the second row below adds
  its height once there is text. Focusing it moves nothing either way. No `↵ to send` or `⇧↵ new line` text: the button is the
  hint.
- **Second row (M6, with a model connected).** While the box holds text or an image, while the
  assistant works on a request sent from it, and while that request's answer shows, a 28 pt row shows
  under the text: `+` (18 pt plus icon, `text.secondary`, round hover well) at the left, then the
  **model chip**, then the send (or stop) button, which moves down into this row. Otherwise the box
  is one line and the chip sits at the right of it (v9 B, so the model in use is always in sight).
  Focusing an *empty* box moves nothing (a box that grows when it is focused would shift the page
  under a press that blurs it, and the click would be lost): the one-line box already has the `+` at
  its left. Without a model there is no second row, no `+` and no chip.
- **Model chip.** 24 pt tall, 7 pt radius, `bg.subtle` on the box's `bg`, focused or not: the provider's pixel mark (18), `Provider · model`
  in `meta` `text.secondary`, a 12 pt chevron. On a one-line box with no room for its words (the
  inspector's, and a page's under 720 pt of content) it is the logo and the chevron alone, its words in
  a tooltip, so the placeholder reads whole; the second row always has the words, ellipsized.
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
- **No model connected.** The same field. `↵` does not send the text to a model; **local rules**
  (`internal/quickadd`, pure Go, no network) read it, in Chinese and English, and add what they find
  at once, in the place of the page (a project; on Today, no project):
  - *Several tasks.* Only explicit marks split a sentence: `，还有` / `, 还有`, `；` / `;`, `然后`,
    `, and also`, `, then`. A bare `and` and a line break do not ("salt and pepper", a pasted
    paragraph stay one title).
  - *Days.* `今天` `明天` `后天` `大后天`, `周一`…`周日` / `星期X` / `礼拜X` / `周天`, `这周X` `下周X`
    `下下周X`, `10月31日` `10.31` `10-31` `2026-10-31` `31号`; `today` `tomorrow` `monday`…
    `next friday` `this friday` `Oct 31` `31 Oct`. The week runs Monday to Sunday: `下周一` on a Sunday
    is tomorrow, a bare `周五` is the first one after today, `这周一` on a Wednesday is gone and
    means the coming one. A date that is over is next year's (`1月5日` in October); one that does
    not exist (`2月30日`) is not read and stays in the title.
  - *Times.* `九点` `9点` `9:00` `晚上8点半` `下午3点15分` `中午` `下午` `晚上` `今晚` `明早`; `9am`
    `3:30pm` `at 9` `noon` `tonight`. A bare hour from 1 to 6 (`3点`, `at 3`) is the afternoon; a
    period word alone means 09:00 (morning), 12:00 (noon), 14:00 (afternoon), 18:00 (dusk), 19:00
    (evening), `今晚` 20:00, as the assistant's prompt has it. A time with no day is its next
    occurrence (today if it is still ahead, else tomorrow); a day with no time is **18:00**
    (`data.DefaultDueHour`) for a deadline and **9:00** for a reminder; a day and time that are both
    given are taken as given, even if today's hour is already over.
  - *Reminder or deadline.* `提醒我…` / `remind me (to)…` sets the **reminder** instead of the due
    time; `…前` / `之前` / `截止` / `before` / `by` / `due` / `until` set the **due** time, and win over
    `提醒我` (`前台` is not `前`).
  - *Category and importance.* `#name` matches a category's name case-insensitively (Work and Life in
    the app's language or as `Work` / `Life`); an unknown `#name` stays in the title, and so does
    the `#` in `C#` or `#homepage` for a category Home. A `!`, `！` or `重要` at the end of the
    task makes it **important** (P1, the default priority otherwise).
  - *The title* is what is left once the recognized words are cut out, with spaces collapsed and no
    punctuation or dangling `的` / `to` / `by` at its ends. A sentence that is nothing but
    recognized words (`明天`) keeps its text as the title and sets no field.

  The "Added to …" line becomes a **receipt** for 6 s, `✓ Added 2: 体检 (Mon 9:00 AM reminder), 交房租
  (due Fri, Life, important)  Undo`; a plain task keeps `✓ Added to Today` for 1.5 s, which also has
  Undo. **Undo** and `⌘Z` in the empty composer delete everything that sentence added, as one
  change, for as long as the line shows. With a model connected nothing of this applies. The line
  `Set up the assistant` (a link to Settings > Advanced) shows under the field while it is focused
  and empty. The inspector's composer is not drawn without a model.
- **Scope.** The composer tells the assistant where it is. On Today (narrowed to a category by a chip too): nothing, or that category (new tasks have no
  project, so they land in Today or Later, unless the words name a project). In a project: that project, so new tasks go there unless
  the words name another. In the inspector: that task, and “this”, “it”
  mean it.
- **Answer.** What comes back shows as one line directly under the box that sent it (above the
  inspector's, which sits at the bottom). Behavior as in **Assistant**.

#### Voice input (m12)

Decision 4 of **Input first (v31)**. The microphone is a 24 pt pixel picture (12 x 12 grid at 2 pt a
pixel) in a 32 pt button at the right of the page composer's box, after the model chip: on Mine and
the project pages, not in the inspector. Platforms without a recorder (Windows until its recorder
lands) do not draw it.

- **Click** asks for the microphone if macOS has not been asked, then records (60 s at most). The
  button becomes a filled `accent` rounded square (radius 8) with the microphone in `accent.text`,
  and one line above the box shows a pixel waveform (12 bars of 2 pt cells, 2, 4, 6 or 8 cells tall,
  the loudest level of each 100 ms beat, oldest at the left) and `Listening… click to stop, Esc to
  cancel`.
- **Click again or Enter** ends it: the line shows the pixel busy indicator and `Transcribing…`. The
  text is put in the box after what is typed, **not sent**: the field takes the focus with the caret at
  the end, so the person can edit it and send it with `↵` or the send button. **Esc** cancels the
  recording or the upload; leaving the page cancels it too.
- **Failures** are one quiet `text.secondary` line above the box for 6 s: `Sign in to use voice` (a
  link that opens Settings on the Account pane; nothing is recorded), `Microphone access is off` with
  `Open System Settings` (Privacy > Microphone), `Voice limit reached for today`, `Voice isn't set
  up on this server`, otherwise `Couldn't transcribe. Try again`.
- **Hold `⌥ Space` to talk is not built**: mygo's `GlobalShortcut.Register(acc, fn)` reports key
  down only, never key up. It waits for a mygo release that reports both.
- No menu bar popover exists yet on macOS; it gets the same button when decision 6 is built.
- The button does not take the field's focus on a click, so the box does not change under the
  pointer between the two clicks.
- Screens: `mine-composer-voice`, `-listening`, `-transcribing`, `-voice-signed-out`.

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
    project or `Today` only when it is not the page the words were typed on (on Today, a task with no
    project says nothing). Up to **three** lines; more receipts end in `and 1 more` (`meta`-quiet, no
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
  It goes at once when the user types in the composer it came from. The dock shrinks back and the
  page grows with it. `Esc` closes it at once; a further `Esc` with
  nothing else to close makes the assistant forget the conversation (it forgets after 10 minutes
  idle as well).
- **⌘Z** undoes the assistant's last change that is not undone, newest first, while its line is shown
  or the conversation is live (until the second `Esc`, or the assistant idling out); with the line
  gone it comes back for a moment saying `Undone`. In a composer with text typed, `⌘Z` is the
  field's own. (See **Undo**.)
- **Memory.** The assistant keeps short notes about the user in one Markdown file, `assistant-memory.md`
  in the data folder (`## 关于我` and `## 纠正与偏好`, one `- ` bullet per note; the user may edit it by
  hand, and a missing file is no memory). Every turn reads the file afresh and puts it in the prompt;
  corrections and stated preferences are saved with `remember`, removed with `forget`, at most 40
  notes / 4000 characters, 200 per note. A write needs no confirmation: its receipt is a line like
  any other, `✓ Remembered Fitness is Life, not Work   Undo` (`✓ Forgot …`), with no task to open;
  Undo takes back that edit alone: a remembered note goes, forgotten notes come back; later notes
  and hand edits stay. Settings > Advanced ends with an **Assistant memory** row and
  an `Open file` link (the data folder when nothing is written yet).
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
view control: `All milestones ⌄`, or the milestone being filtered by (`v0.2 native ⌄`). Its menu
(drawn in the window, a mygo `PopoverMenuButton`, not the system's):

1. `All milestones`, a separator, then one item per milestone (checked when it filters the page),
   as before: `v0.2 · due Oct 28`, `v0.1 · closed`;
2. a separator and `New milestone…`;
3. a separator and one submenu per milestone, `Edit “v0.2”`, holding `Rename…`, `Due date…` and
   `Close milestone` (`Reopen milestone` for a closed one).

`New milestone…`, `Rename…` and `Due date…` open the **Milestone sheet**. Closing and reopening act
at once. A project with no milestone has no filter: `New milestone…` is in its sidebar menu.

**List view: two columns** (an empty project keeps the single list, **List view** below; the board is unchanged). A project
with at least one task is drawn as **Mine** on the left (tasks with `Source user`) and **Agents** on the
right (`Source agent`) under the header, with the composer pinned at the bottom, as in `docs/design/mockups/v7.html` and
`v7b.html` (screens `project-two-columns-*`, `project-list-*`). With no task at all the project keeps the
centred empty state.

- **Block.** Header, columns and the composer's dock share one block as wide as the content allows less the page's
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
  groups are not drawn), then its own **Done** (the newest three and `Show all`, which opens that project's history, see **Completed**) and its own
  `Backlog 2 — show` line. The two columns open and close their Done and Backlog separately. Rows sit
  26 pt in from their header, as in the single list: an Agents row's avatar under the header's name, and the
  titles of both columns a step further in (a Mine row keeps the avatar's room empty); `Show all N →`
  lines up with the titles. While the pointer is over an open row of either column, the row shows its own
  status glyph under the header's glyph, as in the single list.
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
  holds in a project's single list.
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

**List view** (one column; a project's list before M7b, and an empty project's). Groups in this order: Blocked, In review, Doing, To do. Group header: glyph +
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
The composer is pinned under the groups (see **Composer**): it is how a task is added here.
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
  status (a blocked card dropped on Doing stays blocked) and shows the Undo toast, as dragging several
  does; there are no drop targets for hidden statuses.
- **Edges.** The header, the lanes and the composer's dock share their left and right edges: one
  centered block as wide as the window allows less the page's side padding (48 pt), but no wider than
  980 pt. When even 176 pt lanes do not fit (the inspector open on a small window), the block fills
  the room and the lanes scroll sideways.

Decisions made while building it (2026-10-14):

- **One word per status**, everywhere (list groups, board lanes, the inspector, the menu strips
  and their accessibility labels, receipts, Chinese too): Backlog, To do, Doing, Blocked, In review,
  Done, Cancelled. The stale label is `3d stale` in the list and on a card, with no `!`.
- **Review without a PR** still goes to From your agents (`Claude Code finished and is waiting for
  you`, action Review). A review task of my own does not: it waits on someone else.
- **Later** shows dated tasks due within 7 days; later ones appear in their project and on the Calendar.
- **Reminders** stay in Today (an agent task's in From your agents) until marked done; the notification fires once, the row does not
  go away.
- **Tight rows** (the inspector open on a small window): Mine's rows drop the project name below 600 pt of
  content, and the branch, pull request and age below 460 pt. From your agents rows with two buttons put
  them under the summary below 560 pt, and every such row does below 420 pt (the 860 pt window
  with the inspector open leaves 282 pt), so the title keeps its width. A finished row drops who
  finished it below 460 pt and what they said below 600 pt.
- A notification click opens the task in the inspector over Today.

### No project

There is no Inbox page (v31). A task with no project is a task like any other, and it is found
where its kind is found:

- **Mine**, to do, doing or blocked: Today or Next by its date. Dated beyond a week: the Calendar.
  Undated: the `No date` fold of Home (v34); undated tasks with a project are on their project
  page only.
- **Mine, parked in the backlog**: the `No date` fold of Home (a project has its own backlog).
- **An agent's**: Agents while it is at work, From your agents when it asks for you, Done today and
  the history after.
- **Done or cancelled**: Done today, then the history (`Show all`), and search.
- Search finds all of them, grouped by project, the tasks with no project last under `No project`
  / `无项目`.

Two kinds are only in search: a task of mine in `review` (it waits on someone else, see Today) and
an agent's task in `todo` or `backlog` with no project, neither of which the UI makes.

Where the app used to say `Inbox` it now says what is true: a receipt, a toast and a notification
say `Today` (`Added to Today` / `已添加到今天`, `Moved to Today`, a notification's project line); a
place that names where a task lives (the inspector's top line, `Move to…`, the project chip) says
`No project` / `无项目`.

### Inspector (360 pt, slides in 160 ms)

*Being replaced by **Input first (v31)** items 8 and 9.*

Top line (the bar with the close button, `meta` `text.secondary`): where the task lives, `adhd-todo ·
QingYunA/adhd-todo · #7` (the project, its repository when it has one, the task's number; `No project ·
#7` for a task with no project). A long name is ellipsized before the number. Under it the pane differs by
who the task is from (v31, decisions 8 and 9; reference `docs/design/mockups/v31.html`, section 4):
`mine` is a task whose source is the user (not one an agent reported, created or took over), and
the pane for it is much shorter, because a person's task is "what, when, where".

**A task of mine**, top to bottom:

- The large editable title and, at its right, the **important star**: the 12 x 12 pixel star
  (`IconImportant` filled in `status.review` when set, `IconImportantOff`, the outline, in
  `text.tertiary` when not) drawn at 18 pt in a 32 pt button (`Important` / `重要`). A click
  toggles it. Set is P1 and unset is P2 in the database; a P0 task of mine reads as set, and
  clicking it unsets it to P2. Agents and the MCP keep P0 to P3, so the star never hides a value
  another surface wrote.
- **Status**, a segmented control (the List | Board switch's look): `To do`, `Doing`, `Done`
  (`待办`, `进行中`, `完成`). One click sets it; Done plays the completion as the row's glyph
  does. Backlog and Cancelled are reached from the right-click Task menu only; a task in one of
  those (or Blocked, In review) selects no segment and its status is named quietly after the control.
- **When**: the due day and the reminder as two chips (28 pt, 7 pt radius, `bg.subtle`, a 12 pt
  line icon, `Due` / `Remind` in `text.tertiary` while nothing is set). The due chip opens the calendar
  popover, the reminder chip the reminder presets, as the values did.
- **Where**: the category chip (its 12 pt icon in its color and its name; `Category` in
  `text.tertiary` with a list icon while it has none) and the `project · milestone` chip (`No project` in
  `text.tertiary`). They open the Category page and the `Move to…` list of the Task menu. Moving a
  task to another project clears its milestone (milestones belong to one project). A subtask has
  its parent line after Status.
- The task's **images** (see **Task images**), then the Notes.
- `Hand to an agent…` (the robot, a hairline button across the pane; not drawn on a finished task):
  the drop on the Agents tray, by button, with its toast and Undo. The task is the agent's then and
  the pane turns into the agent's.
- No Priority, From, Branch, Pull request or Activity rows.
- **Screens.** `inspector-from-you`, `inspector-mine-starred`; an agent's pane is `inspector`.

**A task of an agent's** keeps the property list: Status, **From**, Priority, **Category**,
**Project**, Milestone, Due, Remind, Branch and Pull request, with left labels, each with its line
icon, in `meta` secondary and values editable in place (a Branch longer than the pane is
ellipsized like every long value, and a click turns it into the editable text); then its **images**, Notes and
`ACTIVITY`, newest first. Save on blur or Enter. Esc closes.

- In an agent's property list the **Status**, **Priority**, **Category**, **Project** and **Remind** values open the pieces of the
  **Task menu** (below) under the value, not a system menu: the status strip, the priority strip, the
  Category page (a filter field and the categories; the row shows only the category that was set,
  after its 18 pt icon in the category's color, and `None` with no icon in `text.tertiary`, for a
  task in a repository's project too), the `Move to…` list (`No project` and the projects, the current one checked) and the reminder presets.
  Moving a task to another project clears its milestone (milestones belong to one project); the
  Milestone row follows the project.
- The body (title, properties, notes, activity, delete) is its own scroll area that takes the height
  the pinned stack leaves; the **answer and composer** are pinned at the bottom under a hairline and
  never overlap it. The answer is the one line of **Assistant**, its left edge on the properties' (24 pt),
  leaving out the title the pane already shows; the composer is the same quiet box (see **Composer**,
  without `⌘K`, placeholder `Change anything about this task`). `Ask about this task…` (⌘J) focuses it.
  Values the assistant set in the last 10 s carry a small accent mark after the value (after a chip, or
  before the star): the pixel `sparkle` (`IconSparkle`, 12 pt) on a `bg.selected` pill. An agent's priority reads as
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
- **In lists.** A row of a task with images (Today, a project's columns and list, search,
  the history) shows its first image as a 32 x 24 pt thumbnail (4 pt radius, hairline, contain on
  `bg.subtle`) just before the due label or the age; with more than one, a `+2` badge (9 pt, white on
  55 % black) at its bottom right. A row without images is unchanged.
- **On board cards.** The first image as a band across the top of the card, 96 pt tall, contain on
  `bg.subtle`, a hairline under it, the `+2` badge at its bottom right when there are more.
- **The viewer.** A click on any thumbnail opens the image over the window: the window dims (90 %
  near-black), the image fits the window whole, a bottom bar says `name · 1 / 2 · 1200 × 900` and
  offers **Open in Preview** (`在预览中打开` and the external-link icon, the file in macOS Preview; on
  Windows **Open** / `打开`, the file in its default app). Round pixel `chevron-left`
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
- **S**, **P**, **M**, **R** on the selected row (below the row; **P** toggles the star of a task of mine at once), and `⌘J`, `⌘⌫` (see Keyboard).

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
   reads `Priority · P1 High` the same way. **For a task of mine** (v31) there is no strip: one line,
   `Mark important` / `标为重要` (`Not important` / `取消重要` while it is set, its star in
   `status.review`), key hint `P`, toggles the star and closes the menu. Above it, for a task of
   mine that is not done (v34): **Do today** / `今天做` with the calendar icon, key hint `T`. It sets
   the due date to today (a time already set is kept) and closes the menu, with the toast
   `Moved to Today` / `已移到今天` and Undo. It is hidden when the task is already due today. The status strip keeps all
   seven statuses, which is how Backlog and Cancelled are reached for a task of mine.
3. One row, `Category` with the 18 pt icon of the task's category in its color (`IconList` when it
   has none), its name as the hint at the right and a `›`: it opens the **Category page**, which is
   the `Move to…` page for categories (v24 frame 2): a filter field `Find a category`, a row per
   category with its icon (the current one checked), `None` (unsorts), a hairline and `New
   category…`, which opens the category sheet and gives the new category to the task. The row takes
   no letter key;
4. a hairline;
5. `Move to…` (key hint `M`): the menu's body is replaced by a filter field and a list of the
   projects and `No project` (the current one checked). Typing filters, `↑ ↓` move, `↵` moves the task,
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
Priority (an agent's) the priority strip, Category the Category page, Project the `Move to…` list, Remind the presets.

A project's menu is in the Sidebar and is drawn the same way (a mygo `PopoverMenu`, in the theme's colors).

### Selecting several tasks

v20 (`docs/design/mockups/v20.html`, #75). Every task list can hold several tasks at once and act on
them together: Today, a project (list, two columns, board cards), Agents, the history and
search results. Calendar and Summary do not select.

- **Select button.** At the right of the page title, `Select` / `多选` (the pixel checkbox icon,
  quiet like the title row's other buttons) enters select mode: each row's status glyph becomes a
  pixel checkbox, a click ticks or unticks the row instead of opening the inspector, and the title
  row reads `2 selected` / `已选 2 件` · `Select all` / `全选` · `Done` / `完成`. `Done`, `Esc`, or
  finishing a bulk action leaves select mode. The first time the user ⌘-clicks in a list, the
  Select button flashes once (two 100 ms frames), so the button is found.
  A project drawn as two columns has no button in its title row: each column head (Mine, Agents) has
  its own `Select` / `多选` at its right end. It enters select mode for that column's rows only (a
  click on the other column's rows still opens them, `Select all` and `⌘A` take the column's rows),
  and the other column's button is hidden meanwhile; the count, `Select all` and `Done` appear in the
  head of the column that is in select mode. Pages with no column heads (Today, a board,
  Agents, the history, search) keep the button in the title row.
- **Without the mode**: `⌘`-click adds or removes a row, `⇧`-click selects from the last clicked row
  to this one, `⇧↑ ⇧↓` extend from the keyboard row, `⌘A` selects every row of the page. A plain
  click opens the inspector and makes that row the only one selected, as before. `Esc` clears the
  selection first, then does what it did.
- **Look.** Selected rows take `bg.selected`; neighbours join into one block (no rounded corners
  between them). The keyboard row also has a 1.5 pt accent ring at 55 %. Selected rows are read as
  "selected".
- **The bulk bar.** With 2 selected (1 in select mode), a pixel panel floats at the bottom centre of
  the list (the day card's shape: two-step corners, a 2 pt solid edge, a hard 4 pt shadow, no blur):
  `3 selected` / `3 件`, **Done** (`⌘↵`), **Move to…** (a project or no project), **Priority** (P0 to
  P3; when every selected task is mine, **Mark important** / `标为重要`, or `Not important` / `取消重要` when every one
  is starred, which runs at once and says `Marked 3 tasks important`; a mixed selection keeps Priority), **Hand to an agent** (`⌘⇧A`), **Delete** (in the blocked red, after a confirmation `Delete 3
  tasks?` / `删除 3 件任务？`), and `×` that clears the selection. When the window is narrow the
  buttons drop their words, keeping their icons, the count and `×`. It is a toolbar: Tab reaches it.
- **Each action is one change with one Undo** (`⌘Z`, or the toast's **Undo**). Done and cancelled
  rows can be selected (to delete or reopen them); **Hand to an agent** takes only the user's own
  unfinished tasks and says how many it skipped (`Handed 2 to an agent · 1 skipped`). Handed from
  Today, each task waits in its own project's `Waiting for an agent` (a task with no project
  waits among the agents' rows of Today and Agents).
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
  **Copy for the agent** / `复制给 Agent` and **Undo**. Copy puts `Do Withdo tasks #31, #32 and
  #33.` (`Do Withdo task #12.` for one) on the clipboard and reads `Copied` / `已复制` for 1.5 s.

Decisions made while building it (2026-10-18; screens `select-*`):

- **One selection per page**, in `Main.multi` (`app/selection`, a pure module with its own tests): cleared when the page changes or the search starts or
  ends, and held to the rows the page draws. While it is empty the keyboard row is the only selected
  row, as before; a plain click or a plain arrow leaves it empty. A `⌘`-click with nothing ticked
  counts the keyboard row as the first one. `Esc` clears the selection, and leaves select mode,
  before it does anything else.
- **Select button** (a page without column heads) is the first of the title row's controls, before the milestone menu and the
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
- **The filing toast.** When the built-in assistant creates a task and picks its category itself
  (the user named none, the task is not in a repository's project), a quiet toast says
  `Filed under Life` / `已归到 生活` with **Undo**, and goes by itself after 4 s. Undo clears the
  category (the task is Unsorted). Nothing shows when the user named the category or none fit;
  tasks typed by hand or made over MCP never get it. Screen: `mine-filed-toast-*.png`.

### Search

`⌘F`, or a click in the search field at the top of the sidebar, puts the cursor there. The field is a
28 pt well on the sidebar material (`bg.hover`, 7 pt radius) with a magnifier, the placeholder
`Search` and a `⌘F` hint at the right while empty.

Typing replaces the page with **Search results**; while they show, no sidebar row is highlighted, and
the screen under them comes back when the search ends. The page
header reads `Search` with `12 results for “login”` in `text.tertiary`. Results are tasks of every
project and of no project, whatever their status, done and cancelled included (the tasks of archived
projects are left out), matching every word of the query in the title, notes or branch, or a pull
request number (`#42` and `42`). They are grouped by project, the tasks with no project last (`No project`), groups ordered by the
newest task in them. Group header: `#` mark (a tray for no project), project name in `body.strong`, count in `text.tertiary`.
Row: status glyph (a button, like on Mine), title, branch pill and `#PR`, and at the right the age of
the last change. Done and cancelled rows keep their glyph (a button, so a finished task can be reopened from here).

- **No match**: `No tasks match “xyz”` and, in secondary, `Search looks in titles, notes, branches and
  pull request numbers, in every project.`
- **Keys**: in the field `↓` moves to the first result (and on down through the rows, as on every
  list), `↑` from the first goes back to the field, `↵` opens the highlighted row in the inspector
  (the first result when none is highlighted yet), `Esc` clears the text and leaves the field.
  With a result highlighted and the inspector open, `Esc` closes the inspector first, then clears the
  highlight, then the search. Clicking a sidebar item or `⌘1` `⌘2` `⌘3` ends the search. Search is also the way to a finished task
  without opening the history.
- At most 100 results are shown; a quiet last line says when there are more.

### Completed

The full history of what was finished (v31). It has no sidebar row: it opens from the `Show all` /
`显示全部` link at the end of a list's Done section, and a back control returns to that list.

- **Where the link is**: the end of Today's `Done today` (everything finished), of Agents' `Done today`
  (what agents wrote), of a project's Done group in its list and in each of its two columns (the project's
  history; the Agents column's is the project's agents), and of the board's Done lane. A Done section
  with rows always has it; a project's Done group still shows its newest three and a lane its newest
  five.
- **The page**: the done and cancelled tasks of the scope, newest first by completion, grouped by day.
  Above the title a back control: a 12 pt `chevron-left` and the name of the page it came from
  (`Today`, `Agents`, a project) in `meta` accent, a11y `Back to Today` / `返回今天`. Page title
  `Completed` and, beside it, a quiet line: `Done and cancelled, newest first` for everything, `Done
  and cancelled in adhd-todo, newest first` / `Done and cancelled in Agents, …` for a scope
  (`nav.projectID`, `nav.agents`; `Store.LogbookOf`). Day header in `body.strong` with the count in
  `text.tertiary`: `Today`, `Yesterday`, then a date (`Oct 12`, `Oct 12, 2025` in another year),
  written by `app/words`. Row: the status glyph (done check or cancelled slash, a button: choosing
  another status reopens the task), title, project name, and at the right who finished it (`Claude
  Code`, `You`) and the time of day.
- **Empty**: `Nothing finished yet` and `Tasks you mark done or cancel collect here, newest first.`
- 100 tasks are listed; `Show older` at the end loads 100 more.
- The same keys as any list: `↑ ↓ ↵ Space Esc`; there is no key for the page, `⌘1` to `⌘3` leave it.
- Search finds finished tasks too, without the history.

### Summary

The end-of-day ledger (#67). References: `docs/design/mockups/v18.html` (the scene, decided),
`v14b.html` (rules and export), `v15.html` (the assistant's paragraph, the end-of-day reminder);
`v16`/`v17` are the drafts that were turned down. The page is the day (or the week) as a
Stardew-Valley-style shipping: everyone who finished something throws it into a shipping bin, then
a ledger counts it up.

**Where.** A sidebar row `Summary` / `总结` under Calendar (v36; v31 had taken it out), `chart`
pixelarticon, selected while the page shows; it opens on today and plays the scene again. On Today,
`See summary ›` / `看总结 ›` at the right of the `Done today` header opens it, and so does the
end-of-day notification (`ShowSummary`). Header: the
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

**Off by default.** Settings > Advanced gets a group **Pages** (`页面`) with one switch row, **Recent
changes** (`最近变化`), `meta` `See what each project merged lately, each pull request and issue in a
sentence. Needs gh signed in.` (`按项目看最近合并的 PR 和 issue，助手用一句话说清每条做了什么。需要已登录的
gh。`), off; kept in `prefs.json`. While it is off nothing below exists: no sidebar row, no `⌘4`, no
entry on project pages, nothing fetched.

**Where.** A sidebar row `Recent changes` (`最近变化`) under Calendar, icon `git-branch`
(pixelarticons), no count, `⌘4`. Header: the title, a repository pop-up (`All repositories` /
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
化`) with the assistant's avatar. gh missing or signed out: the page says what to run (`brew install gh`, on
Windows `winget install --id GitHub.cli` / `gh auth login`) with Copy.

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

The `Calendar` sidebar item (`日历`, `IconCalendar`), under Agents; `⌘3` opens it. Its `Today` button
has the a11y label `Go to today` / `回到今天`, since Today is also a page.
Reference: `v4.html` frame 1, screens `calendar-*.png`. A month of tasks by the day that matters.

- **Header**: the month as the page title (`October 2026`, `2026 年 10 月`, written by `app/words`),
  and at the right a quiet segmented control `‹ Today ›` in the well of the list and board toggle
  (`bg.subtle`, 8 pt radius, 26 pt segments), Today raised. `‹ ›` show the previous and next month and
  move the selected day along (the same day number, the last day of a shorter month); Today returns
  to this month and selects today. Under it the grid (the page has no composer).
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
  month). The selected day is only a highlight now; it carries no scope (the Calendar's composer is gone, decision 2 of
  **Input first (v31)**).
- **Items**: a click opens the task in the inspector (it does not select the day); a right click opens
  the task menu. Drag an item onto another day: a due item moves its due date there and a reminder
  item its reminder, both keeping their time of day; the target day tints with the accent while the
  item is over it. Finished items do not drag.
- **Keyboard** (no text input focused): `← →` the previous and next day, `↑ ↓` the same day a week
  earlier or later (leaving the grid turns the page), `⌘←` `⌘→` the previous and next month,
  `⌘3` opens the page, `⌘K` goes to Home's big input. Items have no keyboard focus of their own: the task menu
  and `↵` act on the highlighted task of other pages only.
- **Data**: the snapshot holds every task including all finished ones, so no range query is needed;
  `data.BuildCalendar` places them (see `app/data/calendar.go`).

### Sheets and confirmations

Small sheets on the dimmed window, built like Settings (12 pt radius, 24 pt padding) but 400 pt wide.
`Esc` and the Cancel button close them; the primary button is the accent one.

- **New project**: title `New project`; field `Name` (focused); field `GitHub repository` (optional,
  folded by default into a `GitHub repository` row with a chevron, under the name; a click unfolds it, and each new sheet starts folded again;
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

- **Chips.** A row of mine shows a category chip only when the task has a category of its own
  (`Task.CategoryID`, set by me or the assistant), on Today as on a project's page. A task that counts
  under Work only because its project is a repository (`CategoryOf`, `projectHasRepo`) shows none;
  filtering by Work still includes it (`rowCategory`).
- **Settings > General > Categories** (v31; it was the sidebar's sub-rows and their menus). A group of
  rows in the pane's order, between Updates and Archived projects, under the label `Categories` /
  `类别`: each category is a row with its icon at 18 pt in its color, its name (Work and Life in the
  user's language) and at the right `up` and `down` arrow buttons (24 pt targets, left out at their
  end of the list, a11y `Move Reading up`), `Edit` (a link, opens the category sheet below, a11y `Edit
  Reading`) and, for the user's own, a `trash` button (a11y `Delete Reading`). The last row is `New
  category…`. The sheet is the same code as the one the task menu opens in the main window (a `Main`
  the Settings window keeps for it). Delete asks in place: the row becomes `Delete “Reading”? Its
  tasks become unsorted.` with `Cancel` and `Delete`; deleting adds a row, `Deleted “Reading” · 5
  tasks unsorted` (no count when 0) with `Undo` for six seconds, which puts the category back with its
  icon, color, place and tasks (only the ones still unsorted). Screens `settings-categories-*`.
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
  never red. After Create the sheet closes; one that came from the task menu gives the category to the task.
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
- **Delete** (see the group above) unsorts the category's tasks; Undo puts it back whole.
- **Chinese** (frame 8): `新建类别…`, `编辑类别…`, `上移`, `下移`, `删除类别`, `名称`, `图标`, `颜色`,
  `搜索图标`, `没有匹配的图标`, `查找类别`, `工作和生活不能改名。`.

### Undo

**One rule: every change that shows an Undo toast also answers `⌘Z`.** The window keeps one change on
offer, the newest, in one toast (`Undo` at the right, six seconds, a newer change replaces it); the
toast's button and `⌘Z` take back the same thing. The changes that offer it:

- Delete of one task (the task menu, `⌘⌫`, the inspector's `Delete task…`) and of several; a deleted
  category.
- Every bulk action (Done, Cancel, Move to…, Priority, Hand to an agent, Take back), and dragging
  cards to another lane, one card or several.
- Handing one task to an agent, and taking one back.
- Taking an image off a task (the tile's `x`).

The strips of the task menu that change one task's status or priority do **not** offer Undo: they are
one tap to change back. The assistant's receipts have their own Undo on their line (see **Assistant**).
The quiet `Filed under Life` toast of a task the assistant filed is not this toast either: its Undo
only clears the category, while `⌘Z` there takes back the assistant's whole change, the new task.

`⌘Z` takes the assistant's last change that is not undone first, else the toast's change. In a text
field with its own undo history (text typed, or any other input) `⌘Z` is the field's; an empty
composer has none, so there `⌘Z` goes on to the assistant and the toast.

### Keyboard

`⌥ Space` (`Alt+Space` on Windows) opens or closes the menu bar panel from any app, its composer focused (see **Surfaces**) · `⌘K` the page's composer, or Home's big input where the page has none · `⌘Z` the last change that shows an Undo toast, or the assistant's (see **Undo**) · `⌘,` Settings · `⌘F` search · `↑ ↓` move selection through rows on the current screen · `↵` open the
selected row in the inspector · `Space` toggles done for your own tasks · `Esc` closes the
inspector, then clears the selection, then the search · `⌘`-click, `⇧`-click, `⇧↑ ⇧↓` and `⌘A` select several
(see **Selecting several tasks**), `⌘↵` marks them done, `⌘⇧A` hands them to an agent · `⌘1` Today, `⌘2` Agents, `⌘3` Calendar, `⌘4` Recent changes (when it is on, see **Recent changes**). `Esc` also clears the category filter bar, after closing the inspector.

With a row selected: `S` status, `P` priority, `M` move to, `R` remind: each opens the Task menu on
that piece, where `←` `→` or `↑` `↓` move, `↵` chooses and `Esc` closes; `⌘J` asks the assistant
about the task (opens the inspector with its composer focused); `⌘⌫` deletes it, with Undo.

On Windows every `⌘` is Ctrl, and the hints the app shows say so: `keyHint` (`app/views/keyhint.go`)
writes `⌘` `⇧` `⌥` `↵` `⌫` as `Ctrl+` `Shift+` `Alt+` `Enter` `Backspace` there (`⌘⇧A` is `Ctrl+Shift+A`).

While a text input has the keyboard its keys are its own (`Element.OnTextKey`), and the window's
arrows, `↵`, `Space` and `Esc` stay out of the way:

| Input | `↵` | `Esc` | `↓` |
|---|---|---|---|
| Composer | sends (`⇧↵` breaks the line); on an empty field opens the highlighted row | cancels a running request, else closes the answer, else leaves the field | moves into the rows (Now); `⌘Z` on an empty field undoes the assistant's last change |
| Menu bar panel composer | as the Composer, filing into Mine | as the Composer; with nothing left to cancel or close it closes the panel | |
| Inspector title, branch | saves | puts the stored text back and leaves the field | |
| Inspector notes | breaks a line (saves on blur) | puts the stored text back and leaves the field | |
| Search field | opens the highlighted result (the first when none) | clears the text and leaves the field | moves into the results |
| Rename project (sidebar, page title) | saves | puts the old name back | |
| New project, milestone, category sheets | creates / saves | closes the sheet | |

### Notifications

System notifications (v12, `docs/design/mockups/v12.html`). macOS draws them; the app chooses the words,
the sound, the stack and where a click goes. Every kind opens its task in the inspector over Today, and
the notifications of one project stack together in Notification Center (`Group` = the project; the
tasks with no project are one stack).

| Kind | When | Title / subtitle / body | Sound |
|---|---|---|---|
| Handed back | an agent's report moves the task to In review | `Claude Code finished: <title>` (`Claude Code 做完了：<title>`) / project / with a PR `PR #12 is ready for your review`, else the agent's last summary | yes |
| Stuck | an agent's report moves the task to Blocked | `Claude Code is stuck: <title>` (`Claude Code 卡住了：<title>`) / project / what it waits for (`next_step`, else `summary`) | yes |
| Checks failing | the GitHub poller finds a PR's checks newly failing (the moment Mine's row appears) | `Checks are failing on PR #12: <title>` (`PR #12 检查没过：<title>`) / project / `Open the PR to see which one failed` | yes |
| Merged | the GitHub poller finds the PR merged and completes the task | `PR #14 merged: <title>` (`PR #14 已合并：<title>`) / project / `The task is done` | **silent** |
| Reminder | unchanged: `remind_at` passes, once; due while the app was closed, at the next launch | `Reminder: <title>` / — / project or Today | yes |

The agent is named as people write it (`Claude Code`, `Codex`, ...), as in the inspector's From.

- **Once.** Each report or GitHub event notifies once, when the app first sees it.
- **Not what you did.** A change the user made in the app (a status set by hand) notifies nothing.
- **Not what you are looking at.** While the app is frontmost and its window shows Today, a new
  handed-back, stuck or pull request event does not notify: its row is already on screen. In the
  background, or on another page, it does. Reminders always notify.
- **No catch-up** for agent and GitHub events: what happened while the app was closed is on Today at the
  next launch. Only reminders catch up, as before.
- **A burst becomes one.** When a third event arrives within 10 s of the first one still shown, one
  summary notification takes their place: `3 things need you` (`3 件事等你`), body `2 finished, 1 stuck
  · open Today` (`2 个做完了，1 个卡住了 · 点开去「今天」`), counting each kind that occurred; a click opens
  Today. Further events in the burst update the summary (same notification ID) until 10 s pass with none.
- **The switch.** Settings > General > Notifications (below). Off, nothing notifies, reminders included.

### Updates (M10)

The app updates itself from the releases of the public repository `QingYunA/withdo` (until 0.10.2:
`QingYunA/adhd-todo-releases`, see `docs/plans/m10-auto-update.md`) with
mygo's native updater plugin (`native.Plugin`); the window is the plugin's own, so the app draws
nothing of it. All of it shows only in a build that can update (`mygo.Updater.Enabled()`: `updates` in
`app/mygo.json`, which it has), see `docs/plans/m10-auto-update.md`. Screens: `settings-general-*`
show the switch.

- **The menu item**: `Check for Updates…` (`检查更新…`) in the app menu right after About. It opens the
  plugin's window at once: release notes of the new version, **Install Update**, **Remind Me Later**,
  **Skip This Version**, a progress bar while it downloads, then a relaunch. When there is none, the
  window says the app is up to date; a failed check says why. The label follows the app's language.
- **The window** is the plugin's native-UI window (no webview), in the language of the app. Background
  checks run every four hours, the first 10 s after launch when one is due, and never in a development build.
- **The switch**: Settings > General > Software updates (below).
- `withdo` is the app's own executable, linked at `~/.local/bin/withdo` (`adhd-core` stays as a legacy alias), and updates with it; see `docs/plans/m10-auto-update.md`.

### Settings

**Settings with a sidebar (v41, 2026-10-29)** — `docs/design/mockups/v41.html`. It replaces the toolbar
panes and the pane contents below wherever they disagree; the bullets after this block still hold
for what v41 does not change (the sign-in flows, the provider form, the category editor sheet).

1. **Window.** One fixed size, **800 × 580 pt** of content (`theme.SettingsWidth/Height`), not
   resizable. The title bar is hidden; the traffic lights sit at the top of the sidebar. The window
   title is the open pane's name (as today, through `OnTitle`).
2. **Sidebar**, 196 pt wide, the full height, `SettingsToolbar` fill with a 0.5 pt `border` at its
   right. Its first 38 pt row is empty and drags the window (the traffic lights). Then one row per
   pane, 34 pt tall, 8 pt corners, 10 pt inside: the pane's 18 pt pixel icon and its name (`body`).
   The open pane gets the `SettingsTabOn` well, its name in 600 and its icon in `accent`; others have
   the icon in `text.secondary` and the hover well. Order and icons: **Account** (`账号`, user),
   **General** (`通用`, sliders), **Categories** (`类别`, folder), **Agents** (`Agent`, robot),
   **Model** (`模型`, sparkle). At the bottom, 12 pt above the edge: **About Withdo** (`关于 Withdo`,
   diamond-gem), `meta` size in `text.secondary`, which opens the About pane. The Agents row carries a
   6 × 6 pt `accent` square at its right while an agent is found on this computer without its MCP
   (and the pane is not open). Up/Down arrows move between rows when the sidebar has focus.
3. **Pane header.** Every pane starts with its name (22 / 28, 700) and one `meta` line under it in
   `text.secondary` saying what the pane is for, then 16 pt to the first group. Panes keep the
   grouped lists (white or `#2C2C2E` blocks, 10 pt corners, 0.5 pt hairlines, rows of at least
   44 pt, group labels in `section` style). Pane padding 22 pt top, 28 pt sides. **Every pane fits
   580 pt without scrolling** in its usual state; a long list (categories, a sign-in flow) may scroll,
   with a scroll key per pane.
4. **Account**: one group holds the account (the `you` sprite at 48 pt, the name, the email in `meta`,
   **Sign out** as a link at the right), the sync state (a 10 pt square in the done green, `Synced` in
   green, `1 minute ago · 2 devices` in `meta`, **Sync now** at the right) and the Google row. Then
   **Devices** as today. The server line at the foot as in #162 (no address for the official server).
   Signed out, the sign-in card is unchanged.
5. **General**: **Language** as a three-part segmented control (`Follow system` | `English` |
   `简体中文`), not a pop-up. **Notifications and sound** (`通知与声音`): System notifications, the
   end-of-day reminder (its time as a small field beside the switch, no second row), Sound effects.
   **Updates and data** (`更新与数据`): automatic checks (`meta`: `Withdo 0.12.0 · checked 2 minutes
   ago`, or `not checked yet`: the updater cannot tell whether the version is the latest; a **Check
   now** link beside the switch), the data folder with **Show in Finder**.
6. **Categories** is a pane of its own: the categories group (rows show the icon, the name in the
   category's color and `N tasks`; the arrows and **Edit** appear only on the hovered or focused row),
   **New category…**, then **Archived projects** with Restore.
7. **Agents**: header line `Connected agents report their progress to Withdo. Only the MCP is
   installed; the agent itself is not.` Then **On this computer** with its count, and the agents
   found as a row of tiles, five per row (8 pt gaps): a tile is a group block, its agent's avatar
   (`theme.AgentAvatar`) at 48 pt on a 56 pt stage, the name (`meta` 600), one `meta` line
   (`13 reports today` / `No reports today` / `Found`) and its state: **Connected** (`已接入`, a done
   green tint), an **Connect** accent button (`接入`), or `Connecting…`. A connected agent's avatar
   **bobs**: up one sprite pixel (4 pt) and back, on the 100 ms loop beat (3 beats up, 3 down), each
   tile offset by 2 beats; with Reduce Motion it stands still. Not connected, the avatar is gray at
   45 % opacity. An install error shows under the tile's line in the blocked red. Under the tiles one
   group of rows: **N more agents** (`还有 N 个 Agent`, `meta` listing the first names, `not on this
   computer`), which opens the More list in place (the compact rows of today, with Get it);
   **Connect by hand** (`手动接入`, `meta`: the command and the setup guide), which opens in place the
   `claude mcp add` well with Copy and **Copy setup guide**; and **Open agent work in** with its
   pop-up. Then the **GitHub** group: the gh status row (account, last check, **Check now**) and
   **Recent changes** with its switch; switched on with gh missing or signed out, its `meta` says
   what to run. Tiles keep the catalog order, and a tile shows a short name (`Copilot`) where the
   full one would not fit; More and the accessible label keep the full name.
8. **Model** (the old Advanced): the provider in use as one card (36 pt logo, name, `model · ●
   Connected · 640 ms`, **Test** at the right), then **Switch to** (`换一个`) and the 12 provider tiles,
   4 × 3, 56 pt tall, always shown: logo and name, the green set-up square, a 2 pt accent ring on the
   one in use. Clicking a tile opens the key and model form under the grid (today's form); a set-up
   provider becomes the one in use at once and is tested. With nothing set up the card is replaced by
   one line: `Not set up: dates and reminders are still read on this computer.`
9. **About**: the app mark, `Withdo 0.12.0`, `Open source, AGPL-3.0`, links to GitHub, the privacy
   policy and the website (https://qingyuna.github.io/withdo/).
10. **Old pane ids.** `PaneAdvanced` is gone: callers that opened it open **Model**. `Show` accepts
    the five panes and About.

*Before v41: being replaced by **Input first (v31)** item 10.*

A window of its own (M4c; the in-window sheet is gone). `⌘,` (the app menu's `Settings…` item) and the
sidebar gear open it; there is one instance, and a second request brings it to the front, on the pane
asked for. **One fixed size, 680 × 520 pt of content**, not resizable: a pane taller than that scrolls
under the toolbar, and a tab change never resizes the window (M5; resizing to each pane's height was
the flicker). Each pane has its own scroll key. Reference: `v4.html` frames 5 to 7, and `v5.html` frames 5 and 6 (logos, Agents); screens
`settings-account-*` (the old `settings-sync-*`), `settings-general-*` (`settings-general-denied-*`: macOS refused the notifications; `settings-categories-*`: scrolled to the Categories group, `settings-categories-deleted-*`: after a delete, with Undo),
`settings-agents-*` (`settings-agents-github-*`: scrolled to the GitHub group at its end),
`settings-advanced-*` (at rest; `settings-advanced-more-*`: scrolled to the switches under the model; `settings-advanced-new-*`: nothing set up),
`settings-change-*`, `settings-change-error-*`, `settings-custom-*`. Four panes since **Input first (v31)** decision 10, in
this order: Account, General, Agents, Advanced.

- **Toolbar panes**, as in System Settings: Account (`账号`, `IconUser`, the pixelarticons person),
  General (`通用`, `IconGeneral`), Agents (`Agent`, `IconTerminal`), Advanced (`高级`, `IconStar`), each an icon above a
  label (`meta`), 78 pt wide; the open one gets a 6 % black (white in dark) 8 pt well and an `accent` icon.
  The window title is the open pane's name. The top of the window is **one band** (the toolbar
  fill, a hairline under it): the title bar is hidden, the traffic lights sit in the band's first
  row, the open pane's name is centered there in `section` style, and the pane icons sit under it.
- **Grouped lists.** Pane background `#F5F5F7` (dark `#1C1C1E`); groups are white (dark `#2C2C2E`)
  rounded 10 pt blocks with a 0.5 pt hairline; rows are at least 44 pt, the label (`body`) and an
  optional `meta` line left, the control right; hairlines between rows; an optional group label
  above the block in `section` style (13 / 18, 600, `text.secondary`).
- **General** (language, notifications, data folder, archived projects, software updates, about; the
  menu bar icon switch of v31 comes with the menu bar icon): Language (a pop-up: System / English / 简体中文); **Notifications** (v12): a group of one
  row (the end-of-day reminder is in Advanced), **System notifications** (`系统通知`) with the `meta` line `Agents done or stuck, failing or merged
  PRs, and reminders` (`Agent 交回或卡住、PR 检查没过或合并、提醒到点时通知你`)
  and a switch at the right, on by default. When the last notification was refused by macOS
  (`mygo.ErrNotificationsDenied`) the `meta` line reads, in `stale`, `macOS does not allow Withdo to
  notify` (`macOS 不允许 Withdo 发通知，开关不起作用`) and an **Open System Settings** link
  (`打开系统设置`, followed by the pixel external-link icon, as every Settings link that leaves the app) opens the Notifications settings (macOS only: Windows shows notification-area balloons and never refuses one, so the line never shows there); **Categories** (a row per category with its arrows, Edit and Delete, and `New category…`, see **Categories (M9)**); **Archived projects** (a row per
  archived project with a **Restore** link, or the empty line); **Data folder** (the path in `meta`, written
  from the home folder as `~/Library/Application Support/adhd-todo`; **Show in Finder** at the right
  reveals the real path; on Windows **Show in File Explorer** / `在文件资源管理器中显示`); About (`Withdo 0.3.0`, `Open source, MIT license`, a GitHub
  link). **Software updates** (M10; only in a build that can update): a group of one row after Notifications, `Software
  updates` (`软件更新`), `meta` `Checked every four hours; you choose when to install` (`每 4 小时检查一次，装不装由你决定`),
  and a switch at the right, on by default, which is mygo's updater plugin's "automatic checks": the
  plugin keeps it in its own `updater.json`, not in the prefs, and the row reads it back whenever the
  plugin reports a change (a check, an answer in the update window).
- **Advanced** holds **Use your own model** (`使用自己的模型`), the old Assistant pane unchanged, and under it the
  switches that left General.
- **Use your own model**, at rest (M5): the label **In use** and one card (a group): the provider's **logo**
  (38 pt, `theme.ProviderLogo`, see **Provider logos** below), its
  name (`body` 500), then `model · ● Connected · 312 ms` in `meta` (the model in `text.secondary`, the
  dot and words in the done green), and **Change** at the right. Before any test the line reads
  `Not tested yet` with a **Test** link; a test that failed shows its friendly error in `meta` under
  the line, in the blocked red, with **Test again**. Under the card, **Also set up**: a group with a
  row per other configured provider (a 28 pt logo, name, model, and for Ollama and Magpie their
  note), **Use** at the right; it makes that provider the one in use and tests it. A provider counts
  as set up when it is in use, has a stored key, or (Ollama, Magpie, Custom without a key) was saved
  once. With nothing set up the pane opens on the grid at once, without **Done**.
- **Use your own model > Change**: the label **Pick a provider** with **Done** at the right (back to rest), and
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
- **Advanced, under the model**: three groups, with no label: **Recent changes** (`最近变化`; the switch of the
  page, off by default), the **End-of-day reminder** switch with its **Reminder time** row, and **Sound effects**
  (`音效`). Their wording is as it was in General; the **Pages** label is gone.
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
- **Agents** (M6, v14; it ends with the **GitHub** group below, `docs/design/mockups/v14.html`): the label **Coding agents** (`编程 Agent`),
  one plain `meta` line under it (the M6 wording: what Install MCP does and does not do), then two
  sections, both on the first screen of the 680 x 520 window, before the pane scrolls:
  - **On this computer** with its count (`On this computer 6`, `本机已有 6`; the number is quiet, in
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
    row: `No agent found on this computer yet` / `还没在这台电脑上找到 Agent`, with **See how to
    connect** / **去看看怎么接**, which opens the README's connect section; More still lists the
    whole catalog.
  - **More** with its count: every catalog agent not on this computer, one white block of compact two
    column rows (8 in the design, four rows). A row (31 pt in the app; the mockup draws 32): the
    18 pt official mark, the name in `meta` size and primary ink, and `Get it` / `去下载` and the 12 pt pixel external-link icon at the
    right, which opens the agent's URL. There is no status line and no "Not on this computer" (the
    group says it), and More never folds.
  Under the two sections, exactly as before: the manual snippet when a config file is not plain
  JSON, or when a Cordis patch is not a patch list Withdo can append to; then the **Open agent
  work in** / `去看看时打开` row with its pop-up (`The agent's app` default,
  `Terminal`, `Editor`, `Finder`), and the `claude mcp add` command in a mono well with **Copy**.
  Under it, **Another agent** / `其他 Agent`: one `meta` line and **Copy setup guide** /
  `复制接入说明`, which copies a guide any agent can follow to add Withdo itself (`agentsetup`
  Guide, also printed by `withdo guide`): the server, how to install it (its own command first,
  else its config file after a backup), how to check it (`get_overview`), how to report, and an
  offer to add the reporting rule to the agent's global instructions after asking the user. The
  manual snippet of a failed install has the same button. (#86)
  The choice is kept in `prefs.json` next to the database. Screens `settings-agents-*`; the
  scrolled one is `settings-agents-open-with-*`. "Install MCP" is the only word for this; it is
  never called connecting. Detection (`internal/agentsetup`) reads files and looks on PATH (plus
  `/opt/homebrew/bin`, `/usr/local/bin`, `~/.local/bin` and the like, which a Dock-launched app's
  PATH lacks) every 30 s and after an install; **nothing is installed without the button**.
  - Commands, run with the absolute `~/.local/bin/withdo` and a 30 s timeout: `claude mcp add --scope
    user withdo -- <core> mcp`; `codex mcp add withdo -- <core> mcp`; `gemini mcp add -s user
    withdo <core> mcp`; `qwen mcp add --scope user withdo <core> mcp`; `kimi mcp add --transport
    stdio withdo -- <core> mcp`; `amp mcp add withdo -- <core> mcp`; `code --add-mcp
    '{"name":"withdo","command":"<core>","args":["mcp"]}'`.
  - **The entry before the rename** (`adhd-todo`, running `adhd-core`) is moved to `withdo` by the
    app on launch, in the background and without asking: `withdo` is added, then `adhd-todo` is
    removed with the agent's own `mcp remove` (Claude, Gemini and Qwen at user scope; VS Code has
    none, so its `mcp.json` is edited) or out of its config file. An agent that still has only the
    old entry shows **Install MCP**, which does the same.
  - Files (copied first to `<file>.withdo.bak`, key order, number text and indentation kept, created
    when missing): Cursor `~/.cursor/mcp.json` `mcpServers`; opencode `~/.config/opencode/opencode.json`
    `mcp` (`type: local`, `command: [core, "mcp"]`); Windsurf `~/.codeium/windsurf/mcp_config.json`
    `mcpServers`; Kiro `~/.kiro/settings/mcp.json` `mcpServers`; Cline `~/Library/Application
    Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json`
    `mcpServers`; Zed `~/.config/zed/settings.json` `context_servers`.
  - **DeepSeek Harness is the one that is not JSON.** It takes MCP servers as Cordis patch entries,
    and it is detected by `~/.dsh` alone (the design's `dsh` is not on PATH). Install appends one
    entry to `~/.dsh/cordis.patch.yml`, the home patch layer every profile reads after its own:
    `- insert:` with `@deepseek-ai/dsh-mcp-client`, `serverName: withdo`, `transport: stdio`,
    `command: <core>`, `args: [mcp]`, `failOnStartupError: false` (the plugin's documented minimal
    stdio configuration, with one change: true would make the plugin refuse to activate when
    withdo fails to start, and false degrades to no tools, so Withdo can never stop DeepSeek
    Harness from starting). A patch file that is not a patch list Withdo can append to (a
    mapping, a flow sequence or a list indented from column 0, for example) is not rewritten: the
    pane falls back to the manual snippet, as for a JSON file it cannot safely edit. Installed is
    read from the home layer and every profile's `cordis.patch.yml`.
  - **Amp, Kiro and Cline** join the catalog in v14. Amp is found by its command and installed with
    its own `amp mcp add`; its `~/.config/amp/settings.json` (`amp.mcpServers`) is read for the
    installed state. Kiro and Cline are found by their folder/file and installed by adding the
    server to their MCP JSON. Nothing was guessed: an agent whose method could not be verified is
    listed in More only, never given a button that would fail.
  - Installed is read from `~/.claude.json`, `~/.codex/config.toml` (`[mcp_servers.withdo]`),
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
  drawing: their avatar is their 18 x 18 harness mark (`theme.harnessSprite`, light and dark), the
  whale, Amp's red mark and Kiro's purple ghost, matched by the same names as `agentName`
  (`deepseek*`, `dsh`, `dsh-*`; `amp`, `amp-*`, `amp *`; `*kiro*`).
- **GitHub** (M4a; the last group of the Agents pane, under the label `GitHub`, since agent pull requests are what it watches): the pull request poller's state, from the `gh` CLI the user is already signed in
  to (the app owns no token). One line and a **Check now** link:
  - *Connected*: `Connected as QingYunA · checked 2 min ago`; while a check runs the link reads
    `Checking…` and is inert; before the first check ends the line says `Checking GitHub…`.
  - *No `gh`*: `The GitHub CLI isn't installed. Install it to follow your pull requests.` and the
    command `brew install gh` (on Windows `winget install --id GitHub.cli`) in a mono well.
  - *Signed out*: `You're not signed in to GitHub. Sign in to follow your pull requests.` and
    `gh auth login` in the mono well.
  - *Offline*: `Can't reach GitHub. Trying again soon.` Never red.
  Polling runs at launch, every 5 minutes, and when the main window comes back to the front if the
  last check is older than a minute; at most 4 `gh` calls at once. With no poller the pane says so.
- **Account** (M11, the old Sync pane, retitled; reference `docs/design/mockups/v27.html`; signing in `v32.html`): the account and
  this computer, in one group, and the server at the foot in `text.tertiary`. **The official server
  has no address on screen** (2026-10-28): the foot is only the link `Use your own server`
  (`使用自己的服务器`). A custom server shows `Server todo.example.com · Change`. Either link edits the
  address in place, for people who host their own; the field starts empty for the official server and
  holds the address for a custom one, with the placeholder `Leave empty for the official server`
  (`留空即使用官方服务器`). Saving an empty field saves the official server. "Official" is
  `data.IsOfficialSyncServer`: `https://todo.sri-ai.xyz` without spaces, a trailing slash or case.
  **The computer is named for the system** (2026-10-28): every Sync-pane line that says `this Mac`
  (`这台 Mac`) says `this PC` (`这台电脑`) on Windows; the catalog keeps a `...Windows` message beside
  each, and `app/views/system_words.go` picks it by `systemGOOS`, as for Finder and File Explorer.
  Signing in happens in the pane, never through a browser. **Signing up and signing in are one path** (M13 P2, as Linear, Notion and
  Slack do it): an email, then a 6-digit code mailed to it; an email with no account gets one at
  that step, with no name asked (the name is the part of the email before `@`). A password is
  optional: `Use a password instead` keeps it for accounts that have one. The phone's Sync screen
  has the same steps (below).
  - *Signed out*: one centered card: the 36 pt pixel person in `accent`, `Sign in or sign up`
    (`登录或注册`, 17 pt 600), one `meta` line, `Your tasks, projects and agents' reports sync to your
    phone and other computers. A new email gets an account here.` (`任务、项目和 Agent 的汇报会同步到手机和其他电脑。第一次用这个邮箱，会顺手建好账号。`),
    then one `fieldWell` 280 pt wide, `you@example.com`, focused when the pane opens, and under it the
    accent button **Continue with email** (`用邮箱继续`, 280 pt), enabled once the field holds an
    email; Enter presses it. In `text.tertiary` under it: keys and model settings stay on this Mac (this PC on Windows).
    Above the field (M13 P3, `v32.html` ①): a 280 pt outlined button **Continue with Google**
    (`用 Google 继续`, Google's four-color `G` mark at 14 pt, the one logo here that is not pixel art,
    as Google's branding rules require), then an `or` (`或`) rule, 12 pt `text.tertiary` between two
    hairlines.
  - *Google*: the button opens the system browser on Google's consent page and the card shows the
    Google mark, `Finish in your browser` (`在浏览器里完成登录`), `Choose your Google account there;
    this window picks it up by itself.` (`在那里选好 Google 账号，这里会自动继续。`), and a **Cancel**
    (`取消`) button that drops the wait and returns to *Signed out*. Once the browser is done the
    pane goes on as any sign-in (*Signing in*, then *Synced* or the both-have-tasks sheet). The
    browser tab ends on the server's own "you can close this window" page. A Google account whose
    email already has an account signs in to that account. Failures: `Google sign-in was cancelled`
    (`Google 登录已取消`), `Google sign-in didn't finish; try again` (`Google 登录没有完成，再试一次`)
    after 5 minutes or a refusal, and the network sentence.
  - *Code*: the 36 pt pixel envelope in `accent`, `Check your email` (`查收邮件`), `We sent a 6-digit
    code to <email> · Use a different email` (`6 位验证码已发到 <邮箱> · 换一个邮箱`, the address in
    `text`, the link back to *Signed out* with the email kept). Six 38 x 46 pt cells, 8 pt apart, a
    short `text.tertiary` dash after the third, digits in 22 pt mono 600; the first empty cell has
    the focus ring of the `fieldWell`. Typing a digit moves on, Backspace goes back, `⌘V` of a code
    fills all six, and the sixth digit sends it (no button). Under them `Didn't get it? Resend in
    0:24` (`没收到？0:24 后可以重发`), counting down from 30 s, then the link `Resend code`
    (`重新发送`); then the link `Use a password instead` (`改用密码登录`).
  - *Password*: the person, `Sign in with a password` (`用密码登录`), the email with `Use a different
    email`, one masked `fieldWell` `Password` (`密码`), the accent button **Sign in** (`登录`), and
    the links `Forgot password` (`忘记密码`) · `Use a code instead` (`改用验证码`). Forgot password
    mails a reset link and shows the envelope, `Reset your password from your email`
    (`去邮箱重设密码`), `We sent a link to <email>. Set a new password in the browser, then come back`
    and a **Back to sign in** button; the link opens PocketBase's own page on the server.
  - *Signing in*: the cells (or the fields) and the buttons disabled, and under them the pixel busy
    indicator with `Checking the code…` (`正在验证…`) or `Checking your account…` (`正在验证账号…`), then
    `Downloading your tasks…` (`正在下载任务…`) while the device links and pulls.
  - *Sign-in failed*: one `meta` line in `status.blocked` under the cells or the button. A wrong code:
    `That code isn't right` (`验证码不对，再看一眼邮件`), the cells emptied and the first focused, the
    link `Resend code` under it. A code older than 10 minutes: a new one is mailed at once and the line
    says `That code expired; we sent a new one` (`验证码过期了，已经发了一个新的`), the countdown starting
    again. Too many tries (codes or passwords): `Too many tries; wait a minute` (`试得太频繁了，等一分钟`).
    A wrong password: `Wrong email or password` (`邮箱或密码不对`; the password emptied and focused).
    `This account is deactivated; ask the admin` and a network failure's sentence as before. The
    email is never said to have, or not have, an account.
  - *The code email* (PocketBase's OTP template, English only, the product being for people outside
    China): from `Withdo <noreply@sri-ai.xyz>`, subject `<code> is your Withdo code`, the code
    in 34 px mono, `It works for 10 minutes. If you didn't ask for it, you can ignore this email;
    nobody can sign in without the code.`
  - *The phone* (Sync screen, `docs/design/screens/android/sync-*`): the same server rule (`isOfficialSyncServer`,
    `syncServerToSave` in `SyncRules.kt`; the hint is `sync_server_hint`), and its link-this-phone dialog
    says `on your computer`, not Mac. The same steps on its card (Google
    opens in an in-app Custom Tab, falling back to the default browser where none supports Custom
    Tabs; the toolbar takes the `bg` colour; the card waits as on the desktop),
    the cells 44 x 54 dp, the numeric keyboard with `autofillHints` one-time code, and the buttons full
    width as the card's other buttons.
  - *Synced*: row 1 the account's display name with `Signed in · 3 devices` and **Sign out** at the
    right; row 2 `This Mac` with `Synced · just now` (`已同步`, the done green) and **Sync now**. Under it
    the group **Devices**: an 18 pt pixel laptop or phone in `text.tertiary`, the device's name, a
    `This Mac` badge (the outlined `badge` of P1) on this one, and the time it last synced at the right.
  - *Both have tasks*: the first sign-in of a Mac with tasks to an account with tasks opens a sheet
    over the pane: `This Mac and your account both have tasks`, a line with the counts, two choices
    as 10 pt rounded options with a radio, **Merge into the account** (`合并到账号`, marked
    `Recommended`, picked) and **Use the account** (`以账号为准`); merging gives this Mac's tasks new
    numbers from the account's blocks (`#45` becomes `#2045`, said in the option), using the account
    first backs this Mac's data up into the data folder. **Cancel** and the accent **Continue**.
  - *Wrong account* (`settings-account-wrong-device-*`): the server answers a device that is not in
    the signed-in account with 403 and the body `{"error":"device does not belong to this user",
    "code":"device_not_in_account"}`. The runner stops (no retry loop) and row 2 `This Mac` shows, in
    `status.blocked`, `This Mac is linked to another account. Sign out, then sign in with that account.`
    (`这台 Mac 记在另一个账号下。退出后用那个账号登录。`), the `N changes wait on this Mac and are not
    lost` line under it, and a primary **Sign out** at the right (the account row loses its own link
    so there is one). No Retry, no Sync now, no Google row. Only a network failure or a 5xx is `the
    server did not answer`; any other failure reads `Sync failed: something went wrong` (`出了点问题`).
    The phone says `This phone is linked to another account. Sign out, then sign in with that
    account.` (`这台手机记在另一个账号下。退出后用那个账号登录。`) with **Sign out** in the same place
    (`sync-wrong-account-*`).
  - *Other account sheet* (`settings-account-other-account-sheet-*`): a sign-in (code, password or
    Google) that succeeds on a database already linked as a device is checked with the account's
    device list before anything syncs. If the device is not in it, a sheet as the both-have-tasks one:
    `This Mac was linked to another account` (`这台 Mac 之前登录的是另一个账号`), the line `It was synced
    with <old email>. You just signed in as <new email>.` (the old email comes from what the database
    remembered when it last synced or signed out; without it the line is only `You just signed in as
    <new email>.`), and two choices: **Sign in with <old email> instead** (`改用 <旧邮箱> 登录`, marked
    `Recommended`, picked: signs out of the new account and opens the email card with the old email
    filled; `Sign in with another account` without an old email) and **Move this Mac to <new email>**
    (`把这台 Mac 换到 <新邮箱>`: forgets the old device link, keeps the tasks and links this Mac as a new
    device of the new account through the usual link, so an empty account is seeded from it and an
    account with tasks gets the both-have-tasks sheet, whose merge or replace never loses the Mac's
    changes: merge queues every row again, replace backs the data up first). **Cancel** signs out of
    the new account. The phone has the same sheet on its card (`sync-other-account-*`).
  - *Google row* (`settings-account-google-link-*`, `-linked-*`): in the signed-in group, when the
    server lists Google, a row with the Google mark and `Google`: **Link** (`绑定`) while the account
    has no Google identity, `Linked` (`已绑定`, the done green) once it has, `Finish in your browser`
    and a **Cancel** link while the browser is open. PocketBase links the identity to the record whose
    token comes with the exchange (`auth-with-oauth2` with the `Authorization` header); if that Google
    account is already linked to another record it answers that other record's sign-in instead, so the
    client compares the record id, drops the token and says `This Google account is already used by
    another account` (`这个 Google 账号已经绑在别的账号上了`). The account's own links are read from
    `_externalAuths` (owner rule); it holds no email, so the row cannot show the Google address. The
    phone's signed-in card has the same row (`sync-google-link-*`, `sync-google-linked-*`).
  - *Uploading* (the first device seeds): row 2 reads `Uploading this Mac's data`, `1,240 / 3,512 ·
    other devices wait until it is done`, and a pixel progress bar: 40 cells 9 x 8 pt, 2 pt apart,
    `accent` for the done share and 8 % black for the rest. Under the group: the window can close and
    the upload goes on.
  - *Failed*: row 2's `meta` starts with the reason in `status.blocked` (`Sync failed: the server did
    not answer`), then when it retries, and a second line that the changes waiting (`3 changes wait on
    this Mac and are not lost`); the button is **Retry**. *Offline* is the same row in `text.secondary`
    (`Offline · changes sync when you are back online (3 waiting)`) with no button and no red.
  - **The main window stays quiet.** It shows nothing about sync while it works. When a sync has failed
    for more than 15 minutes, or the account signed out, a 12 pt pixel cloud in `status.blocked` sits at
    the right of the sidebar's Settings row; clicking the row opens Settings on Account.
- A report from an agent in an archived project's repository restores the project too.
- A test says `Connected · 640 ms` or the error in the words of **Assistant**; nothing is tested
  unless the user asks (Test, Use, Save and test).
- Keys live in the system keychain (service `adhd-todo`), never in the database, the provider file or
  the logs. The active provider and per-provider overrides are in `models.json` next to the database.

## Android (M11, reference `docs/design/mockups/v26.html`)

The phone app is the same product: the palette, the type tokens, the pixel sprites, the status
glyphs and the pixelarticons above, at the same sizes; Jetpack Compose draws them (the sprites as
bitmaps with nearest-neighbour scaling, the glyphs from the same 12 x 12 grids). Screens are 412 dp
wide; rows are at least 48 dp tall (Android's touch target). Light and dark follow the system.

- **Navigation** replaces the sidebar with a bottom bar of four: Mine (the pixel you), Agents (the
  robot), Calendar and Projects, an 18 dp icon over a 12 sp label; the open one has an `bg.selected`
  pill 56 x 28 dp behind its icon and its label in `text` 600. Mine carries the sidebar's red unseen
  badge. The Calendar tab opens the desktop's month view; the first version may ship without it.
- **Header**: the page's title in `title` and its quiet second line (the date, `3 working`), and at the
  right the sync state as an 18 dp pixel cloud in `text.tertiary`; it only turns `status.blocked` for
  the reasons the desktop's sidebar does.
- **The composer** is pinned above the bottom bar, as on the desktop.

**v35 (2026-10-26), reference `docs/design/mockups/v35.html`.** Home on the phone follows v34. Where
this block disagrees with the bullets around it, it wins.

- **Colors and lines.**
  - The phone takes the Radix tokens of **Color**, light and dark.
  - Lines are 0.5 dp.
  - Selection is neutral (`bg.selected`, slate 4): the bottom bar's pill and a highlighted sheet row.
- **Bottom bar.**
  - The first tab is `Today` / `今天` (was Mine / 我的).
  - The bar is `bg.sidebar` with a 0.5 dp top line.
- **Header.** The date is the title: 24 sp / 600, `10月20日 星期二`. The cloud stays at the right.
- **Page order, top to bottom:**
  - **The Agents tray**, the desktop's v34 tray and cards: one tinted button per card and the
    second action as `slate 11` text (`回复`, `打开 PR`, `标为完成`). `打开页面` is no longer a
    button.
  - **Today**, **Next**, **No date** (a fold) and **Done today** (`查看全部` at the right),
    with the desktop's rules; `查看全部` is left out until the phone has a summary.
  - Rows are 48 dp. Categories are their name in color.
- **The composer.**
  - **Place.** It stays docked above the bottom bar, where a thumb reaches; it is the page's only
    floating surface.
  - **Shape.** Radius 20, a 0.5 dp `slate 7` edge, and a soft shadow up and down.
  - **Text row.** 14 dp in from the edge. The 24 dp assistant avatar, then 17 sp text. The
    placeholder is `明天 9 点打给房东，让 Claude 修 bug` / `Call the landlord at 9 tomorrow, have
    Claude fix the bug` (one line on a 412 dp screen).
  - **Control row.** Only once the phone can add images or dictate (not yet): image and microphone
    (40 dp, radius 12). Until then the send button sits at the end of the text row. The send button
    is 40 dp, radius 12; `slate 3` while empty, `iris 9` with a white arrow once there is text.
  - **No key hint.**
  - **Answers.** The receipt and the other answers sit **above** the composer (the bar is below
    it), in 13 sp `slate 11`, with `撤销` in `iris 11`.
  - **A new row** gets the `iris 3` fill for 2 s. An undated add opens the No date fold.
- **The task menu sheet.**
  - **Do today** / `今天做` (calendar icon) comes first under the status strip, for a task of
    mine that is not done and not due today.
  - The strip's buttons are 44 dp, radius 12; rows are 52 dp.

- **Mine** and **Agents** keep the desktop's sections and rows. In **From your agents** the buttons go
  under the summary (the rule for content under 420 pt), left-aligned, with the age at the right of
  that line. **Answer** on a phone writes a reply that is filed in the task's activity, which the
  agent reads with its next report: a phone cannot open the agent's app.
- **A project** stacks the desktop's two columns as a segmented control, `Mine 5 | Agents 3`, under the
  title; groups by status with the glyph in the header, rows 26 dp in with the author's avatar; Done
  folds into one `Completed 12` row.
- **A task** opens full screen (the inspector's content): a back bar with the task's number at the right
  in `mono` `text.tertiary`, the title at 22 sp 700, the property rows with their 18 dp icons and a
  104 dp label column, the priority strip, Subtasks, then Activity on its rail. The composer at the
  bottom answers the agent.
- **The task menu** (reference `docs/design/mockups/v30.html`) is the desktop's **Task menu** as a
  modal bottom sheet (`bg`, 28 dp top corners, a 32 x 4 dp handle, the scrim at 32 % black). It opens
  by a **long press** on any task row (Mine, Agents, a project, search): the row takes `bg.selected`
  at once and the sheet opens when the finger lifts without moving (a long press that moves is kept
  for dragging rows); and by the `⋯` button in a task's back bar, left of its number. A task page's
  Status, Category, Project and Remind rows (each with a `›` at the right) open the sheet at their
  page, and its priority strip stays as it is.
  - Top: the title (17 sp 600, one line) and `#58 · adhd-todo` in `meta` `text.tertiary` (for an
    agent's task `#12 · Claude Code · adhd-todo`).
  - `Status · Doing` in `section` style and the seven status glyphs at 24 dp, each in a 52 dp tall
    cell sharing the width, the current one tinted `bg.selected`; then `Priority · P1 High` and
    `P0 P1 P2 P3` in 44 dp cells (P0 in `status.blocked`, the current one tinted). A tap sets the
    value, gives a light haptic and closes the sheet.
  - A hairline, then 52 dp rows with 20 dp pixelarticons in `text.secondary`: `Category`, `Move to…`
    and `Remind me…`, each with its current value and a `›` in `text.tertiary` at the right
    (`None` when empty); `Open PR #73` (`external-link`, `GitHub` as the hint) only when the task has
    a pull request; a hairline and `Delete` in `status.blocked`.
  - `Category`, `Move to…` and `Remind me…` replace the sheet's body with a page: a back chevron and
    the page's name, then for Move to a 48 dp filter field (focused, the keyboard up) and rows of
    `Inbox` and the projects with their colored folders, the current one on `bg.hover` with a check in
    `accent`; Category is the same list of categories with `None` last and no `New category…` (a
    phone does not make categories). Remind me lists the desktop's presets with their moments at the
    right (`Later today`, `Tomorrow`, `Next Monday`, `This weekend`, by the same rules), a hairline,
    `No reminder`, and `Custom…`, which opens Material's date picker and then its time picker. A
    choice applies and closes the sheet. The system back goes from a page to the main page, then
    closes the sheet; a swipe down or a tap on the scrim closes it.
  - `Delete` does not ask: the sheet closes and a snackbar above the composer says `Deleted
    “title”` with `Undo` for six seconds; Undo puts the task back with its number.
  - The desktop's `Open in…` and `Ask about this task…` have no phone version.
- **Notifications**: reminders that come due use Android's notifications; agents being stuck do not
  push in the first version.
- **Updates** (screens `update-banner-*`, `update-banner-downloading-*`, `sync-update-available-*`).
  mygo does not cover Android, so the phone has its own updater, quiet like the desktop's. At most once
  a day, when the app starts or comes back to the front, it asks the latest release of
  `QingYunA/withdo` (no sign-in; a failed answer, such as GitHub's rate limit, says nothing
  and waits for tomorrow). A newer `v<x.y.z>` with a `withdo-<x.y.z>.apk` shows a **banner** under
  the Mine header (and the other tabs' header): a `GroupCard` of one 56 dp row, `Version 0.9.0 is
  available` (`有新版本 0.9.0`) in `body`, a `Later` text link (`稍后`) and a `QuietButton` **Update**
  (`更新`). `Later` hides the banner until a newer version. **Update** swaps the row for `Downloading
  the update… 42%` over the pixel progress bar, checks the file's SHA-256 against the release's
  `SHA256SUMS.txt`, and opens the system installer. If installs from this app are not allowed yet, the
  row says `Allow installs from Withdo in Settings, then come back` while Android's settings screen
  is open, and the installer opens by itself on return. A failed download or checksum shows `Couldn’t
  download the update` in `status.blocked` with Later and Update again. The **Sync screen** has the
  same offer as its own group above the Server footer, always shown: `App updates`, the version or
  the state below it, and `Check now` (`立即检查`) or `Update`.

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
- The app runs on macOS and Windows: no "this Mac" / 「这台 Mac」 in shared copy. Say "this computer" /
  「这台电脑」 (「本机」 where short), and "this device" in sync, where devices are what is linked.
- A system's own apps and commands are named for the system the app runs on (`systemGOOS`,
  `app/views/system_words.go`): Finder / File Explorer (`访达` / `文件资源管理器`), Open in Preview /
  Open (`在预览中打开` / `打开`, the default app), `brew install gh` / `winget install --id GitHub.cli`.
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
