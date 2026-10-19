# ADHD Todo for Android

React Native / Expo development-build client for the existing Go / MyGo project. Tasks live in on-device SQLite: creating, editing and completing them works without the computer or a network connection. Optional synchronization uses the separate Go service described in [android-sync.md](../docs/android-sync.md).

The first Android client includes Mine, Agents, projects, task details and subtasks, calendar, reminders, search, English / Chinese, light / dark appearance, and a visible offline queue / conflict workflow. A new installation starts empty. Existing desktop data can be bootstrapped with the explicit read-only snapshot import; continuous desktop replication and production OIDC remain later M7 work.

## Develop

Use Node.js 24 and pnpm 12.3.4. Native dependencies are pinned to Expo SDK 58; use `pnpm exec expo install <package>` to add compatible modules.

```sh
cd mobile
pnpm install --frozen-lockfile
pnpm start
```

Use an installed development build, not Expo Go. For a local Android build, install JDK 21 and the Android SDK, configure `ANDROID_HOME`, then run:

```sh
pnpm android
```

The Android native project is generated from `app.json`; do not edit generated `android/` files. The app ID is `dev.adhdtodo.mobile`.

## Test and build

```sh
pnpm typecheck
pnpm lint
pnpm test
pnpm export:android
pnpm prebuild:android
cd android
./gradlew assembleRelease -PreactNativeArchitectures=arm64-v8a,x86_64
```

The test APK includes its JS bundle and runs without Metro. Local generated release builds use the template's testing signing configuration; configure a private release key before a Play release. The Android workflow builds the APK and runs offline / process-restart checks on an emulator. Test files use actual SQLite; set `MOBILE_SYNC_TEST_URL` and `MOBILE_SYNC_TEST_TOKEN` to include the live Go-service interoperability case.

EAS is optional. `eas.json` has development, preview APK and production AAB profiles; cloud builds require the user's own Expo account and appropriate signing setup.

## Connect synchronization

Run the Go service on a computer-independent host for cross-device use. In Settings, enter its URL and access key, then Connect. Credentials use platform secure storage; queued task data does not contain the key.

For Android Emulator local testing, the host service URL is `http://10.0.2.2:8787`. Real devices need a reachable LAN or HTTPS address. The test configuration permits cleartext LAN development traffic; use HTTPS for a deployed service. Switching an already configured server is blocked while unresolved queued edits exist, to avoid sending or dropping data unexpectedly.

This service currently uses one private owner's bearer access key. It is not a public multiuser login service. A phone can manage its local tasks before configuring synchronization; while disconnected, other-device changes arrive only after it reconnects.

## Reminders and validation limits

Enable reminders in Settings and allow Android notification permissions. Each reminder is scheduled with the OS and reconciled after task edits and synchronization. OS scheduling and background execution are best effort; notification denial never discards task edits.

Automated SQLite, HTTP, build and emulator evidence is distinct from a real-phone acceptance run. Verify notification delivery, battery restrictions, keyboard behavior, TalkBack and upgrades on actual devices before treating the Android client as a production release. iOS source compatibility is retained, but iOS builds and device validation are not part of this Android delivery.
