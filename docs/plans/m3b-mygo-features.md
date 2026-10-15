# M3b: mygo 0.5 features and a lighter Anthropic client

Requested 2026-10-15, before M4 (gh polling, which waits).

## 1. Keys in mygo Secrets instead of go-keyring

- `providers.Keyring` stays the interface. The mygo-backed implementation lives in `app/`
  (so `internal/providers` and `cmd/adhd-core` do not import mygo):
  `mygo.Secrets.Service("adhd-todo")`, account `model/<id>`. The same service and account as
  today, so existing keys are found.
- go-keyring stored values as `go-keyring-base64:<base64>` (older: `go-keyring-encoded:<hex>`).
  On read, decode either prefix and write the plain value back, so the item is then the app's.
- Remove `github.com/zalando/go-keyring`.
- Tradeoff, accepted: the app is ad-hoc signed, so macOS asks once after each new build before
  the app can read the key ("Always Allow"). With a Developer ID signature it asks once.

## 2. OnTextKey for the text inputs

Keys typed in a focused input are handled by that input with `Element.OnTextKey`, instead of
window shortcuts plus focus checks: the ask field (Enter sends, Esc cancels / closes the sheet,
Down moves into the rows), the inline add input (Enter adds, Esc closes), inspector fields
(Enter saves, Esc reverts). Window shortcuts keep Cmd+K, Cmd+1, Cmd+2 and Esc when no input has
focus. Behavior stays as in `ui-spec.md` > Keyboard unless listed in the commit.

## 3. Simplified Chinese UI

- mygo catalogs: `app/locales/en.json` (source) and `zh-Hans.json`, `i18n` in `app/mygo.json`,
  generated `catalog_gen.go` committed.
- Every user-visible string goes through the catalog: views, notifications, menus, window title,
  empty states, errors, receipts. Relative ages and due labels ("2m", "Today", "Fri", "Oct 20")
  use the catalog and `i18n` date formatting with `App.Region()`.
- The app follows the system language. Settings gets a Language row: System, English, 简体中文
  (`App.SetLocale`).
- Chinese copy reads like macOS's own Chinese UI: short, no translationese.
- The assistant still answers in the language the user writes in.
- Screens: Chinese renders of Now (real data copy), a project list, the conversation sheet and
  Settings, light and dark.

## 4. No AWS or Google SDKs in the app

`eino-ext/components/model/claude` imports `anthropic-sdk-go/bedrock` and `/vertex`, which pull
the AWS and Google Cloud SDKs into the app (about 25 MB). Replace it with a small Eino
`model.ToolCallingChatModel` in `internal/providers/anthropic` on the core
`github.com/anthropics/anthropic-sdk-go` package: system prompt, text, tool definitions, tool
calls and results, usage, errors. Stream wraps Generate (the agent does not stream).

relaykit (QuantumNous/new-api) was considered: it converts request and response data between
protocols but leaves HTTP, SSE and the Eino side to the host, so it would add a conversion layer
without removing code.

Done when `go list -deps ./app | grep -E 'aws|cloud.google'` is empty and the app is measured.
