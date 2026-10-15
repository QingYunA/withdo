# M3: built-in agent behind the ask field

Goal: the user types anything into the Now screen's ask field — "明天九点提醒我开会",
"PR 12 merged, close that task", "what is blocked?" — and a small built-in agent turns it into
task changes, asks a short question when it truly cannot decide, and reports what it did with
an Undo. The user never has to install Claude Code or Codex.

## Decisions

| Concern | Choice | Why |
|---|---|---|
| Agent framework | Eino (`github.com/cloudwego/eino` + `eino-ext`), `adk.ChatModelAgent` with tools | Mature Go framework, model-agnostic, tool calling and multi-turn built in; no hand-written loop. |
| Model protocols | Anthropic Messages (`eino-ext/components/model/claude`, `BaseURL` supported) and OpenAI-compatible (`eino-ext/components/model/openai`, `BaseURL`) | These two cover nearly every provider and gateway. |
| Presets | Anthropic · Magpie (`http://127.0.0.1:3425`, key `magpie`, Anthropic protocol) · OpenAI · DeepSeek · Kimi · OpenRouter · Ollama (local) · Custom | Magpie gives access to the user's Claude/ChatGPT subscriptions without us touching OAuth. |
| Subscription login | Not in M3. Through Magpie. | Anthropic forbids third-party apps using Claude.ai login. |
| API keys | macOS Keychain via `github.com/zalando/go-keyring` (no cgo); never in the DB or logs | |
| Act or confirm | Act immediately, show a receipt with **Undo** | Lowest friction; ADHD-friendly. |
| Data sent to the model | Only what the tools return for this request, plus project names | |

## Agent

System prompt (kept in one Go file, tested): who the user is (a developer running coding
agents), the current local date-time with weekday and time zone, the user's language (answer in
the language they wrote in), the project list, and the rules:

- Prefer acting. Resolve relative times ("明天九点", "after lunch" = 13:30, "tonight" = 20:00)
  against the given clock; store absolute times.
- A reminder request creates a task with `remind_at`; "提醒我开会" -> title "开会".
- Ask only when an action would be wrong without the answer (two matching tasks, no time for a
  reminder). Ask with `ask_user`, offering 2–4 options.
- Never invent task ids; look tasks up first.
- Keep the final reply to one short sentence.

Tools (typed with `utils.InferTool`, all through `internal/domain`):

| Tool | Does |
|---|---|
| `find_tasks` | Search by text, status, project, source; returns ids, titles, status, due/remind. |
| `create_task` | Title, project, notes, priority, due_at, remind_at. Source `user`. |
| `update_task` | Any field, by id. |
| `complete_task` | Mark done, by id. |
| `list_projects` | Names and ids. |
| `ask_user` | Question + 2–4 options. Ends the turn (`ReturnDirectly`); the user's next input continues the same conversation. |

Every mutating tool returns a receipt the UI shows, and records what Undo must revert
(create -> delete, update -> previous values, complete -> previous status). `domain` gains
`DeleteTask` for this.

Conversation state lives in memory for the current conversation only; it resets after 10
minutes idle or when the user presses Esc twice. No chat history is stored.

## UI (add to `docs/design/ui-spec.md`)

- **Ask field** keeps its look. Enter sends. While the agent works: the sparkle pulses and the
  placeholder becomes "Thinking…"; Esc cancels the request.
- **Conversation sheet** opens directly under the ask field, inside the content column, pushing
  the sections down. It holds this conversation only:
  - the user's line in secondary text, left-aligned like everything else,
  - the agent's short reply in body text,
  - **receipts**: one compact row per change, same row style as the lists (glyph, title,
    "Tomorrow 9:00 · reminder"), with a quiet **Undo** at the right,
  - **questions**: the question in body text and the options as small tinted buttons (same
    style as Needs-you actions); clicking one sends it. Typing a free answer also works.
  The sheet closes 4 s after a finished turn that asked nothing, or on Esc; receipts' Undo stays
  available until then. Errors show inline in the sheet in neutral words with "Retry".
- **No model configured**: focusing the ask field shows one inline line under it, "Connect a
  model to let me understand you — Set up", which opens Settings > Models. Until then Enter still
  adds a plain Inbox task (M2 behavior).
- **Settings > Models**: a list of provider rows (preset logo-less name, base URL, model); one
  is active (radio). Selecting a row opens its fields: Base URL (prefilled), Model (text with a
  few suggestions per preset), API key (secure field, stored in Keychain), and **Test** (sends a
  1-token request, shows "Works · 640 ms" or the error). Magpie row shows "Uses your Magpie
  gateway at 127.0.0.1:3425".

## Tests

- Agent loop with a scripted fake `model.ToolCallingChatModel` (no network): reminder creation
  with correct absolute time across time zones and DST, ambiguous task -> `ask_user` -> answer
  -> update, Undo for each mutating tool, cancel mid-request, provider error surfaces.
- Prompt test: system prompt contains clock, time zone, projects, language rule.
- Provider factory: each preset builds the right Eino model with the right base URL; keys read
  from a fake keyring.
- View tests: sheet states (thinking, reply, receipt + Undo, question + options, error + Retry,
  no-model hint), Esc behavior.
- Optional live test behind `ADHD_TODO_LIVE_MODEL=1` using the configured provider.
- Screenshots (light/dark): `now-thinking`, `now-reply-receipt`, `now-question`,
  `now-no-model`, `settings-models`, `settings-models-edit`.

## Out of scope

Claude CLI provider, streaming token-by-token text (a single final reply is enough for one
sentence), chat history, voice, gh polling (M4).
