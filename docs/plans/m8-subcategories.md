# M8: Mine and Agents get sub-rows

User request (2026-10-16, adhd-todo task #58): the sidebar's Agents splits into the agents at work
(Claude Code, Codex, ...), and Mine splits into Work and Life, which agents fill in on their own.
Decided with the user: two fixed categories, one level only, a task in a repo's project counts as
work, and a task nobody classified is Unsorted (no keyword guessing).

## P1: a task has a category

- Migration `007_category.sql`: `tasks.category TEXT` (NULL, `work` or `life`). `domain.Category`
  with `Valid()`, like `Priority`. NULL is "not classified".
- `CreateTask` takes an optional category; `UpdateTask`'s patch can set or clear it (the same
  present/null shape the patch uses for notes and project). Restore keeps it.
- **Effective category** (`data.Task.Kind()` or a helper of that sort): the task's own category if
  set; else `work` when its project has a repository; else none (Unsorted). An explicit category
  always wins.
- MCP `create_task` and `update_task` take `category` (`work` | `life`; null clears on update). Its
  description says: what the task is for the user, work or life; leave it out for a task in a
  repository's project, which counts as work.
- The assistant (`internal/agent`): `create_task` / `update_task` tools take `category`; the prompt
  tells it to classify every new task it writes as work or life from the words, to leave it out when
  unsure or when the task goes into a repository's project, and to change it when the user asks.
- A new task typed on a Mine sub-page goes into that category (a `Place` field, like a project page
  puts tasks in its project), with or without a model.

## P2: the sidebar's sub-rows

- Under Mine: `Work` / `工作` and `Life` / `生活`, always; `Unsorted` / `未分类` only while it has
  rows. Under Agents: one row per agent with a task at work or finished today, by
  `agentName(taskAgent(t))` (so `claude-code` and `Claude Code` are one row), in the order of the
  Agents screen; an agent with no name is `Agent`.
- A sub-row is a sidebar row one step in from its parent: a 12 pt slot flush with the parent's label,
  empty for a category and holding the agent's avatar for an agent, then the label.
  The count is the parent's count for that slice: rows asking for attention for Mine's, tasks at
  work for an agent's. The red unseen badge stays on Mine only.
- `nav` gets a sub key (category or agent name). A sub-row opens its parent's screen showing only
  that slice: the same sections, built from the tasks of that slice so the counts and "Show all"
  links stay right. The page title is the slice's name (`Work`, `Claude Code`). `⌘1` / `⌘2` still go
  to the whole Mine and Agents.
- A subtask goes where its parent goes.

## P3: setting it by hand

- The inspector gets a `Category` / `分类` row under Priority: Work, Life, None, through the same
  drawn menu page as Priority. The task menu's main page gets the same strip.

## Not in M8

- User-defined categories (that is M9, `m9-custom-categories.md`), nested levels, categories for agents' own tasks on the Agents screen.
