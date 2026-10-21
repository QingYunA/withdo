# M10: automatic updates

User request: the app updates itself from GitHub releases, and the `adhd-core` CLI / MCP binary
updates with it.

**Status: updates are off.** `QingYunA/adhd-todo` is private, so its release files answer 404 to an
app that is not signed in to GitHub, and mygo's updater needs them public. v0.5.0 was published with
updates on, found unable to update, withdrawn, and re-released without them: `app/mygo.json` has no
`updates`, so `mygo.Updater.Enabled()` is false and the app loads no update plugin, shows no `Check
for Updates…` and no Settings switch. `adhd-core` is still the app's own program and is linked on launch.

To turn updates on, once the release files are public (this repository made public, or a public
repository holding only the releases, named in `github`): put back in `app/mygo.json`

```json
"updates": {
  "publicKey": "JKSbO8LWKNM6FUnzD1oemJTAfCx/wkg2hfS4osPS6qw=",
  "github": "QingYunA/adhd-todo",
  "changelog": "../CHANGELOG.md"
}
```

and build with `scripts/build-release.sh`, which then asks for the private key (see below). The first
version built so is the last one to install by hand.

## Decisions

- **mygo's updater, not our own.** `mygo.Use(native.Plugin)` (all of this app's windows are native UI,
  so no webview for the update window) and `updater.MenuItem()` after About. mygo checks the signature
  of every update against the public key in `app/mygo.json` (`updates.publicKey`) and replaces the
  app. Releases are `v0.x.y` tags on `QingYunA/adhd-todo`, so the default tag prefix and the latest
  release are right: one release holds the app and `adhd-core`.
- **The key.** The secret half of the pair is `~/Library/Application Support/mygo/update-keys/mygo-update.key`.
  It is never in the repository (`mygo-update.key` is in `.gitignore` as a guard); losing it strands
  every installed app. `scripts/build-release.sh` reads it, `MYGO_UPDATE_KEY` names another file.
- **Release notes** come from the `## <version>` section of `CHANGELOG.md`; mygo fails the build when
  the file exists without the section of the version being built.
- **Settings > General > Software updates** is the plugin's `SetAutomaticChecks`. The plugin persists
  it (`updater.json` in the data folder), so the app's prefs do not.
- **The app is `adhd-core`.** `app/main.go` hands over to `internal/cli` before anything of mygo
  starts when it is run under the name `adhd-core`, or with one of the command line's own commands
  or flags (`app/cli.go`); anything else opens the window. The bundle carries that one program (about
  8 MB of DMG less than with a second copy). `cmd/adhd-core` builds the same command line alone:
  `scripts/build-release.sh` builds it for arm64 and amd64 (`CGO_ENABLED=0`, `-trimpath -ldflags "-s -w"`)
  and publishes it as `adhd-core-<version>-macos-universal.tar.gz`, for machines without the app.
  `adhd-core --version` prints `adhd-core <version>`.
- **A stable path for the agents.** Agents' MCP configs name `~/.local/bin/adhd-core`
  (`agentsetup.Env.CorePath`), so on launch of a build that runs from an app bundle
  (`internal/corelink`) the app makes that path a symlink to its own executable (`Contents/MacOS/ADHD Todo`) when the path is
  - missing (its folder is made),
  - already a symlink into some `*.app` (an older install, or one that moved or was deleted; v0.5.0 linked `Contents/Resources/adhd-core`, which the app no longer has), or
  - a regular file whose `--version` is below the bundled one, or has none (v0.4.0 and older); it is kept as `adhd-core.old` next to the link.

  Anything else is left alone: a symlink elsewhere, a regular file as new as the bundled one (a
  developer's own build), a directory. The swap is a symlink renamed over the path, so the path is
  never missing. The app logs what it did; there is no UI.

  A run from `/Volumes` (a disk image) or from an `AppTranslocation` path makes no link: it would
  point at a path that is gone once the image is ejected or the app moved. Installing into
  /Applications and starting it again makes the link.

  On Windows the agents run the console `adhd-core.exe` the installer puts next to `ADHD Todo.exe`
  (`agentsetup.defaultCore`; the build script puts it in `app/resources/windows-amd64/`, which mygo
  installs beside the executable): no link and no PATH change, so agents are given its absolute path.

## Release steps (for the release PR)

1. Bump the version in `app/mygo.json`, `app/main.go`, `internal/mcpserver/server.go` and `versionName`
   in `android/app/build.gradle.kts`, and raise its `versionCode` by one (`TestVersionsAgree` in
   `app/` fails when they differ); finish the `CHANGELOG.md` section of that version.
2. `scripts/build-release.sh`: builds macOS and Windows x64, which every release ships, and gathers
   into `app/build/release/` `ADHD-Todo-<v>-macos-universal.dmg`,
   `adhd-core-<v>-macos-universal.tar.gz`, `ADHD-Todo-Setup-<v>-windows-x64.exe` (which carries
   `adhd-core.exe`), the signed update archives and manifests when updates are on, and
   `SHA256SUMS.txt`.
3. `scripts/android-release.sh`: builds `dist/adhd-todo-<v>.apk`, signed with the release keystore
   (made once by `--init`; its password is in the keychain; needs JDK 17 and the Android SDK, found
   at `ANDROID_HOME` or else `~/Library/Android/sdk`). Copy the APK into `app/build/release/` after
   step 2 (which empties that folder) and append its line:
   `(cd app/build/release && shasum -a 256 adhd-todo-<v>.apk >> SHA256SUMS.txt)`.
4. Upload them all to a draft release `v<v>`, with the notes below, then publish it: that makes it
   the latest, which is what installed apps ask.

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
