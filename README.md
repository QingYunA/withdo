# adhd-todo

A todo app for people juggling many things at once. Coding agents (Claude Code first) report
which repo, branch and PR they are working on through a local MCP server, so you can see what
every agent is doing across parallel PRs. Data lives in a local SQLite file.

Status: M1 is the TypeScript core only (SQLite + stdio MCP server + `overview` CLI). No UI yet.

## Build

Requires [Bun](https://bun.sh) 1.3+.

```sh
bun install
bun test && bunx tsc --noEmit
bun run build        # -> packages/core/dist/adhd-core
```

## Use

Register the MCP server with Claude Code (user scope):

```sh
claude mcp add --scope user adhd-todo -- /absolute/path/to/packages/core/dist/adhd-core mcp
```

Then see what your agents are doing:

```sh
packages/core/dist/adhd-core overview
```

Data is stored in `~/Library/Application Support/adhd-todo/adhd-todo.db` on macOS
(`$XDG_DATA_HOME/adhd-todo/` on Linux, `%APPDATA%\adhd-todo\` on Windows). Set `ADHD_TODO_DB`
to use another file.

Smoke test the built binary:

```sh
ADHD_TODO_DB=/tmp/adhd-smoke.db bun run --cwd packages/core smoke
```
