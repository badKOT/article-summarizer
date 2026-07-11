# AGENTS.md

Telegram bot that summarizes articles via an LLM agent (OpenRouter) and
Jina reader. Single Go module, no framework — `agenticgokit v1beta` for the
agent loop, `go-telegram/bot` for Telegram webhook.

## Layout

- `main.go` — entrypoint. Loads `.env` (mandatory), opens Postgres, registers
  agent tools, starts Telegram webhook on `:8080`. `db.Close()` deferred.
- `service/` — command routing (`/credits`, `/model`, `/models`, otherwise
  → `callAgent`). Tool calling is handled natively by the SDK's ReAct
  continuation loop (`WithReasoningConfig` in `callAgent`), not by a custom
  handler.
- `db/` — Postgres access. Single table `chat_info(chat_id PK, model, last_summary)`,
  auto-created in `InitDB`. Interface in `db.Database`; impl is `*SqlDB`.
- `connectors/` — HTTP clients for Jina (`r.jina.ai`) and OpenRouter
  (`openrouter.ai/api/v1`). API keys are package-level vars set from env in `main`.
- `tools/` — agent tools: `jina_parser` (URL → markdown, stores result in
  `chat_info.last_summary`) and `get_last_article` (reads `last_summary` by
  `chatId` from context). Both require `chatId` in `ctx.Value`; tools will
  error if it's missing.
- `constants/` — `SystemPrompt` and tool descriptions used by the agent.
- `models/` — DTOs for OpenRouter `/models` response.

## Commands

- Run: `go run .` (requires `.env` and reachable Postgres)
- Build: `go build` — produces binary `./article-summarizer` at repo root
  (gitignored)
- Test: `go test ./...` (no test files currently; tool-calling now lives in
  the SDK's agent loop, not custom parsing code, so there's little to unit
  test without mocking the LLM provider)
- DB: `docker compose up -d postgres` (compose.yaml has `app` + `traefik`
  services commented out; only `postgres` is active)

## Required env (.env)

`TELEGRAM_API_TOKEN`, `OPENROUTER_API_KEY`, `JINA_READER_API_KEY`,
`DEFAULT_LLM` (e.g. `openai/gpt-5.4`), `WEBHOOK_URL` (public URL Telegram
will POST updates to), plus `DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD`/
`DB_NAME` (defaults match the compose service). `main.go` `log.Fatal`s if
`.env` is missing or DB ping fails.

## Gotchas

- **Telegram transport is webhook, not polling.** The bot calls
  `b.SetWebhook` with `WEBHOOK_URL` and serves `:8080` via `http.ListenAndServe`.
  No long polling fallback.
- **Tool calls only work through the text/ReAct fallback, not native
  function-calling.** The `internal/llm` OpenRouter adapter this SDK version
  uses (`v1beta` → `internal/llm.OpenRouterAdapter`) never sends
  `prompt.Tools` to the API and never reads back structured tool calls — it
  only builds a plain chat completion request. Tool use only works because
  the SDK also appends a ReAct-style text description
  (`Action: <name>\nAction Input: <json>`) to the system prompt and parses
  it back out of the model's plain-text reply (`v1beta.ParseToolCalls`).
- **`WithTools(v1beta.WithReasoningConfig(...))` is required, not optional.**
  Without `Reasoning.Enabled`, the agent runs the tool-execution loop exactly
  once and returns *before* re-prompting the LLM with the tool result — the
  final response ends up being a raw `"<tool> result: <content>"` dump
  instead of a natural-language answer. With reasoning enabled, the SDK
  feeds the tool result back to the LLM and returns its synthesized answer.
- **`Capabilities.Tools` is always `nil` for custom handlers in this SDK
  version (v0.5.9).** In `agent_impl.go`, the `Tools` field on the
  `Capabilities` struct passed to a `HandlerFunc` is commented out
  (`// Tools: a.tools`), so `caps.Tools` can never be populated — this is a
  real limitation of the installed SDK version, not a registration bug in
  this repo. There is currently no `WithHandler` in use; tool execution is
  driven entirely by the SDK's own agent loop (`a.tools`, populated from
  `RegisterInternalTool` via `DiscoverInternalTools`) before any custom
  handler would run.
- **Tool descriptions lie.** `LastArticleTool` description tells the model
  to pass `{"count": 1}`, but the tool ignores args entirely and reads by
  `chatId` from context. Don't fix the description without also implementing
  the arg.
- **`AGK_TRACE=true` is set on every message** inside `callAgent`
  (`service/service.go:120`). Traces land in `.agk/runs/` (gitignored).
- **`chatId` lives in `context.Value(ctx, "chatId", ...)`** as `int64`.
  `JinaTool.Execute` does `fmt.Sprintf("%d", chatId)` — passing a string
  here will silently mis-key the row.
- **Build artifact lands in repo root** as `./article-summarizer`
  (gitignored). Use `go build -o /tmp/app .` if you don't want it there.
- **Go 1.25+ required** (`go.mod` declares `go 1.25.0`).
