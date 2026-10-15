# M5: calendar, pixel avatars, icons, Settings

User request (2026-10-15): a calendar view; agents that report more often and in the user's
language; more icons; a pixel avatar for what a person wrote and one per agent ("人与 Agent 共生的
Todo List"); fix the Settings tab flicker and make adding a provider as simple as cc-switch /
magpie. Approved direction: `docs/design/mockups/v4.html` (render `v4.png`), with the suggested
answers: the calendar shows due, reminder and completion days; pixel art only for avatars and
empty states (line icons stay); agents report after each step.

## Already on main (the shared base)

- `app/theme/pixel.go`: `Sprite`, `Who`, `You`, `Assistant`, `AgentAvatar(clientName)`,
  `AvatarIcon(c, who, size)`, `SpriteIcon`, `DrawSprite`, `DuoIcon(c, size)`; sizes `AvatarSmall`
  12, `AvatarRow` 18, `AvatarLarge` 60. Unknown agents get a robot colored by their name.
- `app/theme/icons.go`: `IconDone`, `IconStatus`, `IconAgent`, `IconBranch`, `IconGeneral`,
  `IconTerminal`, `IconStar`, `IconOctocat`, `IconFlag`; `ProjectColor(name)`.
- `internal/mcpserver`: the instructions ask for a report after each step, written in the
  language the user writes in.

## A: avatars and icons in the views

- Who wrote a task: `Source == user` is `theme.You` (the assistant writing it for you still counts
  as you); `Source == agent` is `theme.AgentAvatar(name)` with the name of its reports (the
  latest report's agent; "agent" when it has none).
- List rows: an 18 pt avatar in the column of the group glyph, titles stay aligned with the group
  title. Board cards: a 12 pt avatar first in the meta line. Inspector: a "From" / "来自" property
  row (avatar + name); every property label gets a 14 pt line icon. Activity timeline: each report
  starts with its agent's 18 pt avatar.
- The composer's ✦ becomes the assistant's 18 pt avatar; a one-line answer that changed nothing
  starts with its 12 pt avatar (receipts keep the green ✓).
- Sidebar: Now `IconNow`, Inbox `IconInbox`, Completed `IconDone`; projects get `IconFolder`
  tinted with `theme.ProjectColor(name)`.
- The drawn menus (task menu and its pages) get a 15 pt line icon before each item; Delete is red.
- Empty states (Now, Inbox, a project, search with no hits, Completed): `DuoIcon` at 60 pt above
  a title and one line of `text.secondary`.

## B: the calendar

- A new screen `screenCalendar`, a sidebar row "Calendar" / "日历" with `IconCalendar` between
  Inbox and Completed.
- Header: the month as the title ("October 2026" / "2026 年 10 月"), at the right a quiet
  segmented control ‹ Today ›. Then the composer, then a month grid of 5 or 6 weeks (Monday first
  in zh-Hans, Sunday first in en), hairline cells, weekends on a faint fill, other months' days in
  `text.tertiary`, today's number in an accent pill, the selected day ringed in accent.
- A cell lists its items, at most 3, then "2 more" / "还有 2 项": tasks due that day (title),
  reminders (time in `text.tertiary`, then title), tasks completed that day (title in
  `text.tertiary`, struck through or with the done glyph), milestones due that day (accent tinted
  chip with `IconFlag`). Each task item starts with its author's 12 pt avatar. An overdue open
  task's title is red. A task due and reminded the same day shows once, as the reminder.
- Click a day: select it; the composer then adds to that day ("Add to Fri, Oct 16, or ask me
  anything" / "添加到 10 月 16 日周五，或者问我任何事"): the agent's scope carries the day, and a
  task with no time words is due that day; with no model a plain task gets that day as due date.
- Click an item: open the inspector. Drag an item to another day: its due date (or its reminder,
  keeping the time) moves there. Done items do not drag.

## C: Settings

- The window keeps one size (680 × 520 content); a pane taller than that scrolls. No resize on a
  tab change, so no flicker. Each pane has its own key.
- Toolbar icons: General `IconGeneral`, Assistant `IconStar`, Agents `IconTerminal`, GitHub
  `IconOctocat`.
- Assistant pane at rest: "In use" / "当前在用", one card: the provider's logo tile, name, model,
  and the last test ("● Connected · 312 ms" in green, or the error), with "Change" / "更换". Under
  it the other configured providers, each with "Use" / "用这个". Nothing configured: the grid shows
  directly.
- Change: a grid of 12 tiles, 6 × 2 (Anthropic, OpenAI, DeepSeek, Kimi, 智谱 GLM, 通义千问, Gemini,
  OpenRouter, 硅基流动, Ollama, Magpie, Custom): a colored logo tile with a monogram, the name, a
  green dot when configured, an accent ring when picked. Under it only API Key (with "Get a key ↗")
  and Model (the provider's `/models` list fetched after the key is saved, falling back to the
  preset's suggestions, plus "Other…"), and "Save and test". A test that works makes it the
  provider in use and returns to the rest view. Custom adds Base URL and the OpenAI / Anthropic
  choice. Ollama and Magpie have no key field.
- Agents pane: the agents that ever reported (avatar, name, reports today, the last report's age),
  then the existing connect instructions.

## Done when

- The surfaces match v4.png in light and dark, English and Chinese; ui-spec updated; tests, vet,
  gofmt, `go tool mygo i18n check` clean; screens regenerated and reviewed after the merge.
