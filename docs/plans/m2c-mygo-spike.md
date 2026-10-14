# M2c spike: replace the Tauri shell with mygo

Branch `spike/mygo-shell`, cut from `feat/m2b-desktop`. Goal: decide with measurements
whether mygo (https://github.com/egoist/mygo, web-frontend mode) replaces Tauri as the
desktop shell. The React frontend and the `adhd-core` sidecar stay. `feat/m2b-desktop`
stays untouched as the baseline.

## User feedback this spike must address

1. No global shortcut by default. Shortcuts conflict too easily.
2. Show the Dock icon. Without a shortcut, the Dock and the tray are the only entry points.
3. Esc must close the quick panel. It does not in the Tauri build.

## Shell design

| Concern | Tauri build (baseline) | mygo spike |
|---|---|---|
| Activation policy | Accessory (no Dock icon) | Regular: Dock icon visible. Clicking the Dock icon shows the main window. |
| Global shortcut | `Alt+Space` | None. Remove the shortcut code and settings UI. |
| Quick panel | NSPanel via `tauri-nspanel`, opened by shortcut or tray | `mygo.NewFlyout` anchored to the tray icon, `URL: "/#/panel"`, `Focusable: true`. Left click on the tray toggles it; it closes on outside click and on Esc. |
| Tray menu | Left click toggles panel | Right click opens a menu: Open main window, Quit (`RoleQuit`). |
| Main window | Hidden at start | Shown at start. Close hides it; Quit still quits (`quitting` flag set in `OnBeforeQuit`, `OnClose` only prevents default when not quitting). |
| Sidecar | `externalBin`, tauri-plugin-shell | Bundled under `resources/darwin-arm64/bin/adhd-core`; located with `mygo.App.Path(mygo.PathResources)`; started with `os/exec` in `WhenReady`, stdin kept open, first stdout line parsed; restart once; killed in `OnQuit`. Stdin EOF remains the fallback. |
| Endpoint to frontend | `invoke('core_endpoint')` | Bound Go service `Core.Conn(ctx) (CoreConn, error)`, blocks until the sidecar is ready. |
| Reminder notifications | Main webview listens to SSE and calls plugin-notification | Go subscribes to `/events` itself and calls `mygo.NewNotification`. Clicking a notification shows the main window at `#/main/task/<id>`. This no longer depends on a hidden webview staying awake. |
| Open PR links | plugin-opener | `target="_blank"` / `window.open`, which mygo sends to the default browser; or a bound `Shell.OpenExternal`. |
| Vite dev URL | `http://localhost:1420` | Keep port 1420 (`devUrl` in `mygo.config.ts`), so CORS needs only one new origin. |

## Changes

- `packages/core/src/http/cors.ts`: add `mygo://localhost`. Keep the Tauri origins for now.
- `apps/desktop`: delete `src-tauri/` and the Tauri npm packages; add `main.go`, `go.mod`,
  `mygo.config.ts`, small Go files for sidecar, tray, windows, notifications; replace
  `src/core/endpoint.ts`, `src/core/notify.ts`, `src/core/provider.tsx`,
  `src/platform/shell.ts` usages with the generated mygo client; remove the shortcut
  settings view. Browser mode (`VITE_CORE_URL`) keeps working.
- In React, any Escape handler that must not close the flyout calls `e.preventDefault()`.

## Steps

1. `mygo` scaffold inside `apps/desktop` with the existing Vite app on port 1420; empty
   main window loads. Go 1.27.1 toolchain is cached locally.
2. Sidecar manager in Go with unit tests (`go test`) using a fake sidecar script:
   ready-line parsing, timeout, restart once, kill on quit.
3. `Core.Conn` binding and frontend switch to the generated client; Vitest stays green.
4. Tray + flyout panel, Dock icon, main window hide-on-close, Quit.
5. Go-side reminder notifications from `/events`.
6. `mygo build`; measure and compare with the Tauri release build.

## Measurements (report both builds, release mode)

| Metric | How |
|---|---|
| `.app` size | `du -sh` |
| Shell binary size, excluding the sidecar | `ls -l` |
| Idle memory | sum of RSS of the app and its WebKit processes after 30 s idle, panel closed |
| Cold start to main window | rough, seconds |

## Done when

- `go vet`, `go test ./...`, `bunx vitest run`, `bunx tsc --noEmit`, and core `bun test` pass.
- `mygo build` produces an `.app` with the sidecar inside, and the app starts it.
- Browser mode still works.
- The report has the measurement table and a manual checklist for the user: Dock icon,
  tray toggles the flyout, Esc closes it, typing works (including Cmd+V), SSE refresh
  within 1 s of an MCP report, reminder notification while the main window is closed,
  Quit leaves no `adhd-core serve` process.

## Out of scope

Signing, notarization, auto-update, settings beyond what exists, Windows and Linux.
