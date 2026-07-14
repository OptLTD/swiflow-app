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

## Deferred / not product features yet

| Item | Notes |
|------|--------|
| Subagents | Phase 2 remaining |
| Mid-run message queue / interrupt-and-continue | Reject while busy (409); UI blocks send |
| Task-level auto verification | Prompt/tools only, no gate |
| Self-evolution of skills/agents | No closed loop; `skill_manage` is opt-in tool |
| Chat events fan-out to `sesshub` | Watch mainly sees scheduled turns |
| Global concurrent-run cap / workspace write mutex | Shared process resources |
| Postgres / multi-tenancy | Phase 3 |
| OTel / rate limiting / LLM titles | Phase 4 |
