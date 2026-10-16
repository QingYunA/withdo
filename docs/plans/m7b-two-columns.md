# M7b: a project's list in two columns, and handing a task to an agent

Approved 2026-10-16. Drafts: `docs/design/mockups/v7.html` (the two columns) and `v7b.html` (the
boundary, option ② chosen, and the hand-off storyboard). Match them.

> **Revised 2026-10-16.** After trying it, the user does not want an agent started for now, and
> wants the sprite during the drag instead of an animation after it. So: **no launching**: a drop only
> moves the task to `Waiting for an agent` (no Terminal, no prompt, no clipboard; the launcher,
> `internal/handoff` and the prompt strings are gone). **Drag-time sprite**: the `you` sprite carries
> the lifted row and walks while the pointer moves; the header robot hops while the row is over the
> tray and squashes once on the drop; the row lands straight in Waiting with its 8 % wash. The 1.3 s
> sequence below (parcel, meeting in the middle) was removed; the parcel stays as the group's glyph.
>
> **Revised again (round 3): a hand-off goes to any agent.** With nothing launched, picking one is
> meaningless (the real data showed `Codex` because it reported last). `domain.HandOff(ctx, taskID)`
> takes no agent and writes `handed_to = "agent"` (`domain.AnyAgent`, the word reports use for an
> unnamed client); `Store.HandOff(id)` returns the `Task` (no `HandOffResult`); `pickAgent`,
> `LastReportAgent` and `agentsetup.Match` are gone. Nothing names an agent: the tray says `Drop to
> hand to an agent` / `松开，交给 Agent`, the toast `Handed to an agent` / `已交给 Agent`, and the
> waiting row shows the accent robot of the Agents header and, under the title, `Tell any agent “do
> #44” to pick it up` / `在任意 Agent 里说「做 #44」就会接手`. Rows of an earlier build that hold an
> agent's id still read as waiting. The sections below say what was first approved and are
> superseded where they mention launching, prompts, the clipboard, picking an agent or the
> post-drop animation.

## What the user asked for

> Two columns: people on the left, agents on the right. Agents are mostly for people to look at;
> people do not need to act on them. The boundary may be clearer, but not overdone. I want to drag
> a task of mine to the agents' side, with a little sprite handing it over.

## Decisions

**Layout (project list view only).** The Inbox, the board, Mine, Agents and Completed are unchanged.

- Header and composer as today; under them two equal columns, `Mine` / `我的` on the left (tasks with
  `Source user`), `Agents` / `Agent` on the right (`Source agent`). The block (header, composer,
  columns) is as wide as the board's: up to `theme.BoardMax` (980 pt), 48 pt side padding.
- A column header: 20 pt avatar (`you`; the grey robot tinted accent for Agents), name in
  `body.strong`, count of its active tasks in `text.tertiary`, a 1 pt `border` line under it.
- **The boundary is option ②**: the Agents column sits on a faint tray, about half as dark as
  `bg.subtle` (`#f8f8fa` light; pick the dark-mode equivalent in `theme`), 14 pt radius, 16 pt inner
  side padding, 28 pt between the columns. The Mine column has no background. No vertical rule.
- Each column groups its tasks by status as the list does today (Blocked, In review, Doing, To do),
  then its own Done (newest three and `Show all n →`) and its own Backlog fold line.
- **Agents column, new first group `Waiting for an agent` / `等 Agent 接手`**: tasks with
  `HandedTo != ""`. Its glyph is the pixel parcel (draw it in `theme/pixel.go`, 10 × 10, the brown box
  of v7b). Row meta: `Handed to Claude Code · opened in Terminal` / `交给 Claude Code · 已在终端里打开`
  (or `· prompt copied` / `· 已复制提示词` when it was not launched; keep which in the view's memory,
  fall back to the plain `Handed to Claude Code` after a restart), and the age at the right.
- Mine rows: no avatar (the column says whose), title, P0/P1 badge and due label at the right.
- Agents rows: the agent's 18 pt avatar, title with the live dot and age at the right, then what it
  last said **up to two lines** in `text.secondary`, prefixed `#12 · ` when it has a pull request.
  **No branch** in the row (the inspector keeps it).
- **Empty column**: the header and one quiet `text.tertiary` line. Mine: `Nothing for you here —
  write one above` / `这里还没有你要做的事 —— 在上面写一句就行`. Agents: `No agent has reported in this
  project yet` / `还没有 Agent 在这个项目里汇报` with the `Install MCP →` link (opens Settings > Agents).
  A project with no task at all keeps today's centred empty state.
- **Narrow** (content narrower than 640 pt, e.g. the inspector open): the columns stack, Mine
  first, the Agents tray under it, 14 pt apart.
- Keyboard order (`listIDs`): every Mine row, then every Agents row.

**Hand-off (drag a Mine row onto the Agents column).**

- Rows of the Mine column drag like board cards (lift, 2° tilt, an empty dashed slot left behind;
  reuse `ui.Drag` / `ui.DragOver` / `ui.Drop` as `board.go` and `card.go` do). Only non-terminal
  tasks of the user drag. While one is over the Agents tray, the tray gets the 7 % accent wash and a
  1.5 pt accent ring at 35 %, and its first line reads `Drop to hand to Claude Code` / `松开，交给
  Claude Code` (the agent `HandOff` would pick; expose a cheap `data` helper for the name if needed).
- On drop the view builds the prompt (localized, see below) and calls `Store.HandOff(id, prompt)`.
  When `Launched` is false it copies the prompt to the clipboard. Then the one-line receipt, as other
  actions do, with ⌘Z: `Handed to Claude Code — opened in Terminal` / `已交给 Claude Code，已在终端里打开`,
  or `Prompt copied — paste it to any agent` / `已复制提示词，粘贴给任意 Agent`. Undo = `TakeBack` and
  restore the old status (it cannot close the Terminal window; that is fine).
- **The animation** (about 1.2 s, frames of v7b): the row shrinks into the parcel; the `you` sprite
  hops from the Mine header to the gap between the columns carrying it; the receiving agent's sprite
  hops from the Agents header to meet it; the parcel passes over (the receiver squashes as it
  catches); the agent hops back and the parcel opens into the new row of `Waiting for an agent`,
  which keeps an accent wash (8 %) that fades over 2 s. Build it on `theme.HopAt` and the hop overlay
  of `views/hop.go` (frames only while it plays; the fake clock drives tests). **Reduce Motion**: no
  hop, the row fades out on the left and in on the right (160 ms).
- **Dragging an Agents row onto the Mine column** = take it back: `TakeBack`, no animation, receipt
  `Yours now` / `现在归你了`; ⌘Z puts Source and HandedTo back as they were, without launching
  anything (add the smallest `data` call that can do it).

**The prompt** (views, `app/locales`), in the app's language, e.g. zh:

```
请完成 adhd-todo 里的任务 #44：<title>
<notes, when there are>
开始时调用 report_progress（task_id: 44，status: "doing"），之后按 adhd-todo 的说明继续汇报。
```

## Data and core (part A)

- Migration `005_handoff.sql`: `tasks.handed_to TEXT`, `tasks.handed_at TEXT`; map them in
  `internal/domain` and `app/data/real_map.go`. Bump `user_version` as the other migrations do;
  `~/.local/bin/adhd-core` must be rebuilt afterwards (an old core cannot open the new schema).
- `domain`: `HandOff(taskID, agentID)` (Source agent, status todo, completed_at null, handed_to,
  handed_at) and `TakeBack(taskID)`; **any report on a task clears `handed_to`/`handed_at`** (in the
  same write as the report, both `AddReport` and `ReportProgress`).
- Which agent: the agent of the newest report on any task of the project, matched to an
  `agentsetup.Agent` by name (`claude-code` → `claude`, as `views.agentName` matches); else the
  first agent in `agentsetup.Agents` whose command is found; else `claude`.
- Launching (`internal/handoff` or in `app/data`, behind an interface so tests never open
  Terminal): only when the project has a `repo_path` that exists and the agent's command is found.
  Write an executable `.command` file in the data folder's temp area (`cd` to the repo, run the
  agent with the prompt, both shell-quoted) and `open` it, which runs it in Terminal. Commands:
  `claude <prompt>`, `codex <prompt>`, `gemini -i <prompt>`, `qwen -i <prompt>`; check each one's
  `--help` on this Mac before relying on the flag, and treat any other agent as not launchable
  (clipboard).
- `data.Real.HandOff` / `TakeBack` replace the stubs in `app/data/handoff_real.go`. The Fake already
  implements both (`app/data/handoff.go`), and its reports clear `HandedTo`.
- Make sure a waiting task does not show as handed over to the user on Mine (From your agents), is
  not stale, and is not on the Agents page's Working section (it is To do, so it should not be:
  verify with tests).

## Views (part B)

`app/views/list.go` and new files as needed (keep files under 400 lines), strings in `app/locales`
(English and zh-Hans), `docs/design/ui-spec.md` (**Project** section: rewrite the list view
paragraph; add the hand-off), tests next to the existing ones (`list_test.go`, `hop_test.go`), and
the screens: `project-*` list screens regenerated, new `project-two-columns-*`, `-empty-agents-*`,
`-narrow-*`, `handoff-*` (a few frames of the hop) in light/dark, en/zh as the others are.

## Done when

- `go test -race ./...` passes, `go vet ./...` is clean.
- The screens match v7 / v7b; review every PNG, with the real-data copy as well as the seed.
- Dragging a task in the real app opens Claude Code in Terminal in the repo with the prompt, the
  task waits in `Waiting for an agent`, and moves to Doing when Claude Code reports.
