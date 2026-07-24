# Swiflow Implementation Status

Tracked against [`SPEC.md`](SPEC.md) and [`AGENT_ARCHITECTURE.md`](AGENT_ARCHITECTURE.md).

## Landed

| Item | Notes |
|------|--------|
| Phase 1 core (SQLite, SSE chat, agents/providers, fs/web/skill tools, Vue UI) | Done |
| Goal-first stop policy (`maxRounds=32`, stall wrap-up, soft/hard nudges) | Done — see `AGENT_ARCHITECTURE.md` |
| MCP client (stdio / sse / streamable) | Done |
| MCP tools in `tool.Registry` (`mcp_<server>_<tool>`) | Done |
| MCP server CRUD API + reload + UI | Done |
| Per-tool enable via `tool_policy` | Done |
| Cron scheduler + CRUD API + UI | Done |
| `web_search` (`duckduckgo` / `brave` / `searxng`) | Done |
| `browser` tool (config-gated) | Done |
| `window_*` + SSE `ui_request` / `/api/window/reply` | Done |
| `schedule_run` / `schedule_create` tools | Done |
| Multi Chat tabs (one tab per session; sidebar hides while chat tab active) | Done (frontend) |
| Multi-session parallel runs (`busy` per sessionKey) | Done |
| Per-round observe (`internal/observe`) + `tool_timeout_sec` + `max_concurrent_runs` | Done |
| Mid-run message queue (busy → 202 + FIFO; Abort keeps queue; drain → sesshub) | Done |
| Chat event Publish to sesshub; UI watch for auto-continue | Done |
| `subagent_spawn` / `subagent_status` / `subagent_wait` async subagent (summary via status/wait; child session UI) | Done |
| `todo_write` / `todo_read` (session checklist; soft acceptance via prompt) | Done |
| `skill_draft` + drafts API/UI (accept → user-skills; no auto `system_extra`) | Done |
| Clarify mid-run (`clarify` tool + Chat UI; ≤15m wait via window reply) | Done |
| Chat upload pending bar + `@/` workspace refs in message/history + system prompt | Done |

## Deferred / not product features yet

| Item | Notes |
|------|--------|
| Subagent deepen: tool to read child session transcript | v1 summary + workspace paths only |
| Hard acceptance gate before `done` | Prompt/skill policy only |
| Unattended skill/`system_extra` evolution | Drafts require human Accept |
| Queue persistence across process restart | In-memory FIFO; restart drops pending messages |
| Full OTel / Prometheus export | Phase 4; v1 is slog + optional counters |
| Postgres / multi-tenancy | Phase 3 |
| Workspace write mutex / browser multi-page pool | Single-page browser pool; concurrent gate helps |
