# Mira Implementation Status

Tracked against `docs/SPEC.md`. Phase 2 MCP and cron implemented.

## Phase 2 — Extensibility

| Item | Status |
|------|--------|
| MCP client (stdio / sse / streamable) | Done |
| MCP tools in `tool.Registry` (`mcp_<server>_<tool>`) | Done |
| MCP server CRUD API + reload | Done |
| MCP UI (`/mcp`) | Done |
| Per-tool enable via existing `tool_policy` | Done |
| Cron scheduler (`agentine/cadence`, fork of `robfig/cron` with active maintenance) | Done |
| Cron job CRUD API + reload | Done |
| Cron UI (`/cron`) | Done |

## Phase 1 (complete)

Build chain, tests, CI, frontend features — see prior sections in git history.

## Deferred

| Item | Phase |
|------|-------|
| Subagents | Phase 2 (not yet) |
| `web_search` provider integration | Phase 1 stub |
| Postgres / multi-tenancy | Phase 3 |
| OTel / rate limiting | Phase 4 |
