# Android

<!-- impeccable:surface-schema 1 -->

Mode: Operate. Platform: Android. This is an adaptation of the existing task UI, not a new visual identity.

## Authority

`PRODUCT.md`, `docs/design/ui-spec.md`, `app/theme/tokens.go`, `app/theme/glyph.go`, the vendored `app/theme/pixel-icons/` and screenshots in `docs/design/screens/` remain the source. The Android app preserves flat, calm lists, violet selection, distinct pixel status silhouettes, real report timestamps, user/agent separation, and Chinese/English support. It starts empty and never invents agent activity.

## Android translation

- Expo Router native stack owns detail routes and Android Back. Unsaved task edits prompt before leaving, including the system Back gesture.
- Five primary destinations use compact navigation below 720 dp; wider windows use a rail. Calendar and task list separate into columns at 1000 dp. Forms and lists cap their reading width rather than stretching across tablets. Orientation remains unlocked.
- System Roboto, scalable text roles, at least 48 dp touch targets, native switches, date/time dialogs and confirmation alerts. Calendar can scroll horizontally below the width required for seven 48 dp day targets.
- Insets protect system bars and cutouts. Editors scroll above the IME, with save accessible in the footer. Lists use FlatList/SectionList for bounded rendering.
- Surface, subtle surface, selected container, primary/on-primary, outline, text and error are explicit light/dark roles. Violet retains the desktop identity. Secondary text and placeholders use darker values than the desktop's tertiary text for phone contrast.

## Product states

Mine shows overdue/today/later personal tasks; Agents shows the most recent actual report and report age. An old report never implies a running agent. Active/completed filters and search make saved work reachable. Project archive preserves tasks; deleting a parent preserves detached children, matching the shared store contract.

Every local operation calls the durable store and displays failures. Home links its local-save/offline/sync summary to settings. Sync status, pending changes and version conflicts appear in settings; conflicts show both saved versions and require an explicit choice. Credentials stay in platform secure storage through the store adapter.

## Validation boundary

Typecheck and lint are required. Android phone/tablet screenshots, dark appearance, enlarged text, IME and Back behavior must be captured from an emulator or device. A JavaScript export is build evidence, not device acceptance. Physical device reminder delivery and multi-device tests remain separately recorded by the parent delivery workflow.
