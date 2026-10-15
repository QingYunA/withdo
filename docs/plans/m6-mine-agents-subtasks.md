# M6: Mine and Agents, subtasks, the hop, provider logos, MCP for more agents, images

User feedback (2026-10-15) on M5 in use: subagents' tasks were titled with their worktree branch
("worktree-agent-adc17554…") and sat in "Needs you" as review forever, because nobody marked them
done. Asked for: tasks and subtasks; a separate tab for the agents and one for the person; a pixel
monster that hops from the agents' tab to the person's when something needs them; real provider
logos; MCP for more agents; a model picker and images in the composer. Approved direction:
`docs/design/mockups/v5.html` (render `v5.png`) with the suggested answers, and two corrections:
the person's tab is "我的" ("Mine"), not "你的"; and "connect" is said plainly as installing the
adhd-todo MCP into that agent ("安装 MCP"). No accent bar marks a newly handed-over item: the hop is
the signal.

## Already on main (the shared base)

- `data.Task.ParentID *int`, `data.CreateTaskInput.ParentID`, `Task.IsSubtask()`,
  `data.Subtasks(tasks, id)`; the Fake stores ParentID on create.

## P1: subtasks in the core and the MCP server

- Migration `004_subtasks.sql`: `tasks.parent_id INTEGER REFERENCES tasks(id) ON DELETE SET NULL`,
  an index on it. Domain `Task.ParentID`; the real data adapter maps it both ways.
- `report_progress` takes `parent_task_id`. A new task gets it as its parent. Without it, a report
  whose cwd is a git worktree under `<repo>/.claude/worktrees/` (or any linked worktree whose branch
  starts with `worktree-agent-`) becomes a subtask of the most recently reported open task of the
  same agent on the main worktree's branch of the same project, reported within the last 12 hours;
  none found: a top-level task as today.
- Default title: the first sentence of the summary (up to 60 runes), not the branch. The branch
  stays on the task.
- Closing: when a task becomes done or cancelled, its open subtasks become done (one report each,
  "Closed with its parent task"). A subtask whose branch no longer exists in the repository, or is
  merged into the main worktree's HEAD, becomes done ("Its branch was merged or deleted"); a core
  function the app calls on launch and on focus (with the GitHub poll), cheap and bounded.
- `get_task` lists subtasks; `list_tasks` takes `parent_task_id`.
- MCP instructions add: pass a title on your first report; when you start a subagent, give it your
  task_id and tell it to pass `parent_task_id`; a subagent reports "done" when it hands its work
  back (its parent reviews it, not the user).
- The Fake mirrors the closing rule (parent done closes the subtasks) so the views behave alike.

## P2: Mine and Agents, subtasks in the views, the hop

- The sidebar's top two rows replace Now: "Mine"/"我的" with the `theme.You` avatar and
  "Agents"/"Agent" with an accent-tinted robot avatar (18 pt). Counts as today; Mine shows a red
  badge with the number of handed-over items the user has not seen.
- Mine: title "Mine"/"我的" with the date, the composer, then three sections: "From your agents" /
  "Agent 交过来的" (blocked by an agent, done and waiting for review, a PR to review, failing checks,
  changes requested, a closed PR, a due reminder of an agent task — the old Needs-you kinds, minus
  subtasks), each row with the agent's avatar, a one-line reason and an action button, in the
  bordered box; "Today"/"今天" (the user's due or reminded today, and doing); "Later"/"之后" (the next
  7 days, then undated by priority, at most 7). Done today stays as it is now at the end.
- Agents: title "Agents"/"Agent" with "3 working"/"3 个在做", the composer, then "Working"/"在做"
  (agent tasks in doing or blocked, newest report first, live dot), each parent with its subtasks
  indented under a hairline rail: avatar 12, title, a status pill ("Done", "Doing", "Handed back" /
  "交回主代理" for review), the age; then "Done today"/"今天做完" with "3 subtasks" on a parent.
  Subtasks never appear as top-level rows anywhere (lists, board, Mine, Agents, search shows them
  with their parent's title as meta).
- The inspector of a parent lists its subtasks; of a subtask, its parent as a link.
- The hop: when an item enters "From your agents" and the user has not seen it, and the window
  becomes active or the sidebar is visible, the agent's 24 pt sprite rises from the Agents row,
  arcs (peak 30 pt above the Mine row), lands on the Mine row by its count, squashes and settles;
  the badge counts up as it lands. 0.9 s; several items hop 120 ms apart, at most 3. Drawn in a
  `ui.Overlay` with `AnimatePose` or a 16 ms `c.After` only while it plays; nothing redraws when it
  is still. Reduce Motion: no hop, the badge fades in. "Seen" is stored per task (in the app's
  settings, by task id and the time it was handed over) and set when Mine is shown.

## P3: provider logos and installing the MCP into more agents

- Vendor lobe-icons SVGs (MIT, `@lobehub/icons-static-svg@1.95.1`) under `app/theme/logos/` with its
  LICENSE; `go:embed`; drawn with `ui.Image` (color) or `ui.Icon` (mono, follows the text color):
  claude-color, openai, deepseek-color, kimi, zhipu-color, qwen-color, gemini-color,
  openrouter-color, siliconcloud-color, ollama. Magpie keeps a tinted robot; Custom a plus. Replace
  the monogram tiles everywhere (grid, rest card, other providers, composer chip).
- Settings > Agents: "Agents on this Mac" / "这台 Mac 上的 Agent" with one plain line: installing the
  MCP adds adhd-todo's MCP server to that agent, so it reports its progress here; it does not
  install or start the agent. A two-column list: Claude Code, Codex, Gemini CLI, Cursor, opencode,
  Copilot (VS Code), Qwen Code, Kimi CLI, Windsurf, Zed (and Amp, Kiro if cheap). Per row: pixel
  avatar, name, a subtitle ("12 reports today", "Found", "Not on this Mac"), and at the right
  "✓ MCP installed" / "Install MCP" / "Get it ↗".
- Detection in a new package (`internal/agentsetup`): a command on PATH (claude, codex, gemini,
  qwen, kimi, amp, opencode, code) or a config folder (~/.cursor, ~/.codeium/windsurf, Zed's
  settings) — and whether adhd-todo is already in its config (read the config; for CLIs, the
  config file they write). Install: run the agent's own `mcp add` command where there is one
  (exact commands in the plan's research, absolute path to adhd-core); otherwise back up the config
  file (`<file>.adhd-todo.bak`) and add the entry, keeping the rest of the JSON/TOML intact. Errors
  show inline. Every install is behind the button; nothing runs on its own. Tests use temp HOME dirs
  and a fake command runner.
- Agent names and avatars: add sprites for opencode and Copilot; `theme.AgentAvatar` and
  `agentName` learn opencode, Copilot ("Visual Studio Code"), Qwen Code, Kimi, Zed, Windsurf, Cline.

## P4: the model picker and images in the composer

- A model chip at the bottom right of the focused composer (logo 16 + "DeepSeek · deepseek-chat ⌄"),
  hidden when no model is configured. Click: a drawn popover listing the configured providers with
  their model; choosing one makes it the provider in use (the same as Settings).
- Images: ⌘V of an image (OnPaste with empty text, then read PNG/TIFF from the pasteboard), files
  dropped on the composer (`DroppedFiles`, png/jpg/gif/webp/heic), and a "+" that opens a file
  panel. Thumbnails 64 × 48 above the text, × to remove, at most 4, each at most 5 MB after
  downscaling to 1600 px. Sent with the text: `data.Assistant.Ask` gains attachments,
  `agent.Ask` builds `schema.Message.UserInputMultiContent` (text + image parts, base64);
  `internal/providers/anthropic/convert.go` turns image parts into image blocks (and keeps a
  message with only images). A model that rejects images: the friendly error "<model> cannot read
  images" with the chip open. Images are not stored in the database.

## Done when

- Surfaces match v5.png in light and dark, English and Chinese; ui-spec updated; tests, vet, gofmt,
  `go tool mygo i18n check app` clean; screens regenerated and reviewed after the merge; adhd-core
  rebuilt; the MCP change tried with a real subagent.
