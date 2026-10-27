# Connect yourself to Withdo

## 1. Goal

Add Withdo's MCP server to yourself, at user scope (all projects), so you can report your work to the user's Withdo app.

## 2. The server

- Name: `withdo`
- Transport: stdio
- Command: `{{CORE}}`
- Args: `["mcp"]`
- Env: none needed

As a generic JSON entry:

```json
{{ENTRY}}
```

## 3. How to install

1. Prefer your own command for adding MCP servers (for example an `mcp add` subcommand), at user scope.
2. If you have none, edit your MCP config file: copy it to `<file>.withdo.bak` first, add only the `withdo` entry, change nothing else, and keep the file valid.
3. If you cannot tell where your config is, ask the user instead of guessing.
4. If you already have an MCP server named `adhd-todo` (Withdo was called adhd-todo before, and its command was `adhd-core`), it is the old entry of the same app: add `withdo` as above, then remove the `adhd-todo` entry (for example with your `mcp remove adhd-todo` command, or by deleting that one entry from your config file).

Do not install anything else.

## 4. Verify

Reload your MCP servers (or ask the user to restart you), then call the `get_overview` tool. If it answers, the server works.

If it fails, check that the file at the command path exists, and that running it with the single argument `--help` prints its usage.

## 5. How to use it

Call `report_progress`:

- when you start a task (summary = the goal; status "doing"),
- after each step,
- when you are blocked (status "blocked"; say in next_step what you need),
- when you open a PR (pr_url, status "review"),
- when you finish (status "done").

Always pass `cwd`: your current working directory. Pass `task_id` when the user names a Withdo task ("do #44"); read it with `get_task` first (it lists attached pictures; `get_task_image` shows one). When one piece of work covers several tasks, report on each with its own `task_id`. Write summaries in the user's language. Keep each report to one or two short sentences. Do not report trivial questions or read-only exploration.

## 6. Optional: a standing rule

Some agents do not read an MCP server's own instructions. Offer to add a short rule with the points of step 5 to your global instructions file (for example `~/.claude/CLAUDE.md`, `AGENTS.md`, or your equivalent). Ask the user first; do it only if they agree.

## 7. Tell the user

Finish by telling the user which file or command you used, and the result of the check in step 4.
