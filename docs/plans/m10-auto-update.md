# M10: automatic updates

User request: the app updates itself from GitHub releases, and the `adhd-core` CLI / MCP binary
updates with it. v0.5.0 is meant to be the last manual install.

## Decisions

- **mygo's updater, not our own.** `mygo.Use(native.Plugin)` (all of this app's windows are native UI,
  so no webview for the update window) and `updater.MenuItem()` after About. mygo checks the signature
  of every update against the public key in `app/mygo.json` (`updates.publicKey`) and replaces the
  app. Releases are `v0.x.y` tags on `QingYunA/adhd-todo`, so the default tag prefix and the latest
  release are right: one release holds the app and `adhd-core`.
- **The key.** The secret half of the pair is `~/Library/Application Support/mygo/update-keys/mygo-update.key`.
  It is never in the repository (`mygo-update.key` is in `.gitignore` as a guard); losing it strands
  every installed app. `scripts/build-mac.sh` reads it, `MYGO_UPDATE_KEY` names another file.
- **Release notes** come from the `## <version>` section of `CHANGELOG.md`; mygo fails the build when
  the file exists without the section of the version being built.
- **Settings > General > Software updates** is the plugin's `SetAutomaticChecks`. The plugin persists
  it (`updater.json` in the data folder), so the app's prefs do not.
- **`adhd-core` rides inside the app.** `scripts/build-mac.sh` builds it for arm64 and amd64
  (`CGO_ENABLED=0`, `-trimpath -ldflags "-s -w"`), `lipo`s them into `app/resources/darwin/adhd-core`
  (gitignored), and mygo copies it to `Contents/Resources/adhd-core` and signs it with the app. The
  same binary is also published alone as `adhd-core-<version>-macos-universal.tar.gz`, for machines
  without the app. `adhd-core --version` prints `adhd-core <version>`.
- **A stable path for the agents.** Agents' MCP configs name `~/.local/bin/adhd-core`
  (`agentsetup.Env.CorePath`), so on launch of a build that runs from an app bundle
  (`internal/corelink`) the app makes that path a symlink to its bundled copy when the path is
  - missing (its folder is made),
  - already a symlink into some `*.app` (an older install, or one that moved or was deleted), or
  - a regular file whose `--version` is below the bundled one, or has none (v0.4.0 and older); it is kept as `adhd-core.old` next to the link.

  Anything else is left alone: a symlink elsewhere, a regular file as new as the bundled one (a
  developer's own build), a directory. The swap is a symlink renamed over the path, so the path is
  never missing. The app logs what it did; there is no UI.

## Release steps (for the release PR)

1. Bump the version in `app/mygo.json`, `app/main.go` and `internal/mcpserver/server.go`; finish the
   `CHANGELOG.md` section of that version.
2. `scripts/build-mac.sh`: writes into `app/build/darwin-universal/` the DMG (also under the release
   name `ADHD-Todo-<v>-macos-universal.dmg`), the signed update archive
   `adhd-todo-<v>-darwin-universal.tar.gz`, `update-darwin-universal.json`,
   `adhd-core-<v>-macos-universal.tar.gz` and `SHA256SUMS.txt`.
3. Upload them to a draft release `v<v>`, with the notes below, then publish it: that makes it the
   latest, which is what installed apps ask.

The release notes say (the changelog's 0.5.0 entry does too): **a coding agent keeps running the
old `adhd-core` process until it is restarted**, so after an update restart Claude Code and the other
agents to pick up the new tools.

## Not done

- **Developer ID signing and notarization.** The app is signed ad hoc and not notarized, so the first
  install from the DMG needs right-click > Open. Updates do not: the app replaces itself, and mygo
  has checked their signature. Set `macos.signingIdentity` and `macos.notarize` in `app/mygo.json`
  when there is a Developer ID.
- **Delta updates** begin with the second published version: mygo reads the published manifest to
  make them, and there is none yet.
- **Restarting agents.** The app cannot restart a coding agent's MCP process after an update.
- **Windows and Linux** updates: the plugin supports them, but the build script and the symlink are
  macOS only.
