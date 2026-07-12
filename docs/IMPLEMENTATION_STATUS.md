# Mira Implementation Status

Tracked against `docs/SPEC.md` Phase 1. Last updated with the comprehensive improvement pass.

## Build and DX

| Item | Status |
|------|--------|
| `webui/` embed + Makefile web targets | Done |
| Default port 8000 | Done |
| `serve --migrate` default true | Done |
| GitHub Actions CI | Done |
| Docker Compose | Done |
| Example `skills/example` | Done |

## Backend

| Item | Status |
|------|--------|
| `initial/schema.sql` + `upgrades/` | Done |
| Disabled skill/provider enforcement | Done |
| Provider URL validation | Done |
| `GetProviderByID` | Done |
| `WebDistDir` dev override | Done |
| SSE terminal error events | Done |
| Tool loop key sorting | Done |
| `Registry.Get` | Done |
| SQLite WAL + connection pool | Done |
| History truncation (`max_history_messages`) | Done |
| UUIDv7 IDs | Done |
| Request ID + access logging | Done |
| Default agent seed (when providers exist) | Done |
| Unit tests (secure, migrate, store, llm, agent) | Done |

## Frontend (webui)

| Item | Status |
|------|--------|
| Settings tool toggles | Done |
| Agents / Providers edit | Done |
| Chat history tool-call restore | Done |
| Agent selector (new sessions) | Done |
| API types (`types.ts`) | Done |
| highlight.js code blocks | Done |
| Pinia stores split | Done |

## Known Phase 1 gaps (deferred)

| Item | Notes |
|------|-------|
| `web_search` provider integration | Still stub |
| Postgres / multi-tenancy | Phase 3 |
| MCP / subagents / cron | Phase 2 |
| OTel / rate limiting | Phase 4 |
