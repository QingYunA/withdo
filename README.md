# adhd-todo

A todo app for people juggling many things at once. Coding agents (Claude Code first) report which
repo, branch and PR they are working on through a local MCP server, so you can see what every
agent is doing across parallel PRs. A native macOS app shows it live: one calm Now screen, then a
list per project. Everything is Go,
and the data is one local SQLite file.

<p>
  <img src="docs/design/screens/now-light.png" alt="Now" width="420">
  <img src="docs/design/screens/project-list-light.png" alt="A project as a list" width="420">
</p>

More in [`docs/design/screens/`](docs/design/screens/); the design is specified in
[`docs/design/ui-spec.md`](docs/design/ui-spec.md).

## Install and update

Install the app once from the DMG of the [latest release](https://github.com/QingYunA/adhd-todo/releases/latest)
(drag it to Applications; the app is not notarized, so open it the first time with right-click >
Open). Install a later version the same way. Automatic updates are built in but off for now
(`docs/plans/m10-auto-update.md`).

`adhd-core` is the app itself: on launch the app links `~/.local/bin/adhd-core`, where the MCP
registration below points, to its own executable, which runs as `adhd-core` under that name, so installing the app brings both. It leaves a build of your own
(one that is as new as the app's) alone. Coding agents keep running the old `adhd-core` until you
restart them. Without the app, `adhd-core-<version>-macos-universal.tar.gz` on the release is the
binary by itself.

## Build

Requires Go 1.27.1 or newer (macOS for the app).

```sh
# the app: app/build/darwin-arm64/ADHD Todo.app and a .dmg
cd app && go tool mygo build

# the CLI and MCP server
go build -o ~/.local/bin/adhd-core ./cmd/adhd-core
```

`scripts/build-release.sh` makes the release files of both platforms each release ships into
`app/build/release/`: for macOS the DMG of the universal app (which is `adhd-core` too) and the
standalone `adhd-core` tarball, plus the signed update archive when updates are on; for Windows x64
the installer and the `adhd-core.exe` zip. See `docs/plans/m10-auto-update.md`.

## Connect Claude Code

Register the MCP server once (user scope):

```sh
claude mcp add --scope user adhd-todo -- ~/.local/bin/adhd-core mcp
```

Other agents can add the server themselves: paste them the output of `adhd-core guide` (or use
Settings > Agents > Copy setup guide in the app).

Agents then report progress to the app. To see what they are doing from a terminal:

```sh
adhd-core overview
```

## Ask field (built-in assistant)

The ask field on the Now screen can understand what you type ("明天九点提醒我开会", "PR 12 merged,
close that task") and change tasks for you, with an Undo. It needs a model: open Settings > Models,
pick a provider (Anthropic, Magpie, OpenAI, DeepSeek, Kimi, OpenRouter, Ollama or any compatible
endpoint), paste its API key (stored in the system keychain) and press Test. Until then the field
just adds Inbox tasks. Provider choices live in `models.json` next to the database. To run the
optional live check: `ADHD_TODO_LIVE_MODEL=1 go test ./internal/agent -run TestLiveModel -v`.

## Data

The database is `~/Library/Application Support/adhd-todo/adhd-todo.db` on macOS
(`$XDG_DATA_HOME/adhd-todo/` on Linux, `%APPDATA%\adhd-todo\` on Windows). Set `ADHD_TODO_DB` to use
another file. The app and `adhd-core` share it; the app notices the other process's writes within
half a second.

## Develop

```sh
scripts/check.sh                                 # while working: gofmt, vet, i18n, short tests
scripts/check.sh full                            # before a PR: + race on the core (CI runs it as two jobs)
scripts/check.sh race                            # only the race tests
scripts/screens.sh                               # re-render the design screenshots; review them in the PR
git config core.hooksPath .githooks              # once: the static checks run on every commit
cd app && go tool mygo dev                       # run the app against the real database (.dev identifier)
go run ./app -fake                               # demo data instead of the database (-empty: none)
```

Layout: `cmd/adhd-core` (CLI), `internal/{paths,store,gitctx,domain,mcpserver,watch}` (core), `internal/{agent,providers}` (the built-in assistant),
`app/` (mygo UI: `data`, `theme`, `views`).

