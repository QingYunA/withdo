# M15: the official model

The input's model comes from the sync server for anyone signed in (ui-spec, **Input first** item 5,
decided 2026-10-21; built in v44). The app never holds the upstream key. Own keys stay available in
Settings > Model on the desktop.

## Server: `/v1/llm`

The same shape as `POST /v1/transcribe`: a session-authenticated proxy with a per-account daily
quota, off when its key is empty.

- `POST /v1/llm/chat/completions`: an OpenAI-compatible chat completions request, forwarded to the
  upstream with the server's key. The server sets `model` to its own and passes the reply (streamed
  or not) back unchanged. Body at most 1 MiB; upstream timeout 90 s.
- **Quota by turns.** The client sends `X-Withdo-Turn: <uuid>`, one per thing the user sent (one
  `agent.Ask`, however many model calls it takes). The server counts distinct turns per account per
  UTC day against `LLM_DAILY_TURNS` (default 100); a request without the header is a turn of its own.
  A hard cap of 12 requests per turn and `LLM_DAILY_TURNS × 12` requests a day bounds a client that
  lies. Over the quota: `429 {"code":"llm_quota"}`. Every answer carries `X-Withdo-Turns-Left`.
- `GET /v1/llm/quota`: `{"limit":100,"left":86,"resets_at":"2026-11-04T00:00:00Z"}`.
- Off (`LLM_API_KEY` empty): `503 {"code":"llm_disabled"}`; `GET /v1/config` says `"llm": false`
  so clients do not offer it.
- Environment (sync.env): `LLM_API_KEY`, `LLM_BASE_URL` (default `https://api.deepseek.com`),
  `LLM_MODEL` (default `deepseek-chat`), `LLM_DAILY_TURNS`. Requests and replies are never logged;
  the log line has the account, the turn count and the status.
- No paid tier yet: one Free quota for everyone. A tier is a later change to how the limit is read.

## Clients

- `internal/providers`: an `Official` source next to the presets. Its model is the OpenAI protocol
  at `<server>/v1/llm` with the session token as the bearer, and a turn id put on every request of
  one `Ask` (the agent's `Config.Model` is called once per turn).
- **Which model answers.** The user's own provider when one is chosen in Settings > Model;
  otherwise the official one when signed in to a server that offers it; otherwise none (local rules:
  `internal/quickadd`).
- Desktop: Settings > Model shows the official card (v44); the composer's model chip reads
  `Withdo 官方` / `Withdo`.
- Phone: `mobile/core` gets `Ask(text, placeJSON)`, `Undo(receiptID)`, `Reset()` and
  `ModelState()` (official, left, limit), running `internal/agent` against the phone's database.
  The phone has no own-key setting in v44.
