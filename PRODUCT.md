# Product

<!-- impeccable:product-schema 1 -->

## Platform

android

## Stack

Confirmed by 云青: React Native + Expo development build, TypeScript and pnpm. Existing desktop clients remain Go / MyGo. Android is the current delivery target; iOS follows later.

## Users and purpose

The user manages personal tasks and watches coding agents across repositories, branches and pull requests. The phone must remain useful when the computer is off and when the network is unavailable. Offline edits persist across process restarts and synchronize across devices when connected.

## Product commitments

Confirmed in this conversation: independent phone use, offline editing, multi-device synchronization, Android first. The accepted scope and acceptance tests are in `docs/plans/m7-mobile.md`.

Existing product identity is documented in `docs/design/ui-spec.md` and `app/theme/`. Preserve its calm task lists, pixel status glyphs and icons, user/agent distinction, language and light/dark behavior. Translate navigation, touch targets, keyboard and system Back to Android conventions.

## Operating context

Native phone use, including one-handed interaction, Chinese input and interrupted sessions. Local edits must not wait for a server response. Agent reports originate on the machines running those agents; stopped agents have no new progress to display.

## Evidence and validation

The existing app and screenshots in `docs/design/screens/` are design evidence. Do not invent user projects, task counts, connectivity or agent activity. Use isolated fixtures for automated tests. Validate source, APIs, databases and Android build output; record real-device validation separately. Deployment provider and production identity provider remain open implementation dependencies.
