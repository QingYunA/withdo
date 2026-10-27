# M14 Rename to Withdo

The product is called **Withdo** ("with" your agents + "do" your todos) from 2026-10-27; it was
ADHD Todo / adhd-todo. The new pixel-check icon is PR #151. This plan records what was renamed, what
keeps the old name and why, how installed copies move over, and the steps outside this repository.

## What changes

- Everything people read: the app's name (`app/mygo.json` `name`, so the macOS bundle
  `Withdo.app`, the executable, the DMG and the Windows installer), the window title and tray
  tooltip, the locales, Android's label, the README and `docs/design/ui-spec.md`, the PocketBase
  app name, mail sender and sign-in code mail (a new migration, `pbauth/migrate_withdo.go`, renames
  a database that already ran the earlier ones), the release file names and titles.
- What agents see: the MCP server is `withdo` (tools `mcp__withdo__*`), the command line is `withdo`
  (`cmd/withdo`, `~/.local/bin/withdo`, Windows `withdo-cli.exe` next to the app), its guide and the
  server's instructions say Withdo. Reports the core files itself are by `withdo`.

## What keeps the old name

| Kept | Why |
|---|---|
| `identifier` `dev.adhdtodo.desktop` | notification permission, login item, single-instance lock and the Windows uninstall entry are keyed on it |
| keychain service `adhd-todo`, data folder `adhd-todo/adhd-todo.db`, `ADHD_TODO_DB` | stored model keys, sign-in token and the database |
| Android `xyz.sriai.adhdtodo` | another id is another app: no update path, data and signing lost |
| APK asset `adhd-todo-<v>.apk` | Android 0.9–0.10 match that exact name (`UpdateRules.kt`) |
| `QingYunA/adhd-todo-releases` | every installed app reads its updates from it (`updates.github`, `LATEST_RELEASE_URL`) |
| Go module path `github.com/QingYunA/adhd-todo` | an internal string; renaming the GitHub repo does not need it |
| `adhd-core` link and `adhd-core.exe` | agent configs written before the rename run them; they are the same program as `withdo` |
| `adhd-sync`, deploy paths, sample repo names in tests and seed data, history docs | not the product name |

## How installed copies move over

The old version installs the update, so its code decides where the files go and what starts next
(mygo v0.6.0 `updater.go`):

- **macOS.** The new bundle is put at the old path, `ADHD Todo.app`, and the old executable path is
  started again; it no longer exists, so the app quits and stays closed. **The user opens it once.**
  Then the app renames its bundle to `Withdo.app` and starts again from there (`app/legacy.go`,
  before the single-instance lock; not on a disk image, translocated or in a folder it cannot write).
  `~/.local/bin/withdo` and `adhd-core` are linked to it on that launch.
- **Windows.** The files are swapped into `Programs\ADHD Todo`. The archive carries
  `ADHD Todo.exe`, a launcher (`cmd/legacy-launcher`) that starts `Withdo.exe`, so the restart and the
  old shortcuts reach the new version instead of looping on the old one. On first launch the app
  renames the Start menu and Desktop shortcuts to `Withdo.lnk` and sets the uninstall entry's name
  and icon. The folder keeps its old name. The shortcuts still point at the launcher, so they show
  the launcher's plain icon.
- **Agents.** On every launch the app moves an agent that has only the `adhd-todo` entry to `withdo`
  (added, then the old one removed), in the background (`agentsetup.Migrate`). Until then the old
  entry keeps working through `adhd-core`. Permission allow-lists that name `mcp__adhd-todo__*`
  need the new name by hand.
- **Android** updates in place; only the label changes.

## Release notes (for the next version's CHANGELOG section)

- **ADHD Todo is now Withdo.** Same app, same data, new name and icon.
- On a Mac, open Withdo once after this update: the update cannot restart the app under its new name.
- Your coding agents are moved to the new MCP server name `withdo` by themselves. If you allowed
  `mcp__adhd-todo__*` tools in an agent's permissions, allow `mcp__withdo__*` instead.
- Restart your coding agents after the update, as with every release.

## Verify before the release

- A real update from 0.10.2 on a Mac: the app quits, opening it renames the bundle, Settings > Agents
  shows the agents on `withdo`, `~/.local/bin/withdo` and `adhd-core` point into `Withdo.app`.
- A real update from 0.10.2 on Windows: the restart opens Withdo, the shortcuts are `Withdo.lnk`,
  Settings > Apps lists Withdo, `.ADHD Todo.exe.old-*` is gone after the next launch.
- `gemini`, `qwen` and `kimi` `mcp remove` were read from their sources and docs, not run.

## Outside this repository (the owner's steps)

1. GitHub: rename `QingYunA/adhd-todo` to `QingYunA/withdo` if wanted (GitHub redirects the old
   URLs; the module path can stay). Do **not** rename `adhd-todo-releases`.
2. Site repo `adhd-todo-site`: name, copy, favicon and social image from the new icon.
3. Google OAuth consent screen (project adhd-todo-512407): app name Withdo, logo; a changed name or
   logo can send the brand through verification again.
4. PocketBase admin UI after the deploy: check Settings > Application and Mail show Withdo (the
   migration renames a sender name of exactly `ADHD Todo`).
5. Domains: `withdo.app` / `withdo.dev` were free on 2026-10-26; `todo.sri-ai.xyz` stays for now.
6. After the release: upstream mygo could restart an updated app through its new bundle or
   executable name, which would close the macOS gap for future renames.
