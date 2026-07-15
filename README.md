# Swiflow

A self-hosted AI agent runtime. Single Go binary + Vue UI, SQLite or Postgres,
OpenAI-compatible providers, tool use (filesystem / web / shell / skills),
SSE streaming. See `docs/SPEC.md` for the full development specification.

## Quick start

```bash
cp config.example.json config.json   # then edit auth_token / encryption_key
make dev      # API :8000 + UI http://localhost:5173
```

`serve` applies the database schema by default (`--migrate` is on). Use
`--migrate=false` to skip.

Postgres (optional): set `db_driver` to `postgres` and `db_dsn` to a pgx URL
(or `SWIFLOW_DB_DRIVER` / `SWIFLOW_DB_DSN`). Schema is applied from
`embed/schema.pg.sql` (greenfield; no SQLite upgrade scripts).

```json
{
  "db_driver": "postgres",
  "db_dsn": "postgres://swiflow:swiflow@localhost:5432/swiflow?sslmode=disable"
}
```

Build & run:

```bash
make build    # webui + Go binary → ./swiflow
./swiflow serve -v
```

Docker:

```bash
cp config.example.json config.json   # edit secrets
make image
docker compose up
```

Built-in skills are embedded in the binary (`embed/init-skills/`). User overrides go in
`./data/user-skills/` (see `config.example.json`). For local skill development without
rebuilding, set `init_skills_dir` or `SWIFLOW_INIT_SKILLS` to a filesystem directory.

## Phase 2 features

- **MCP** — configure MCP servers at `/mcp` (stdio / sse / streamable). Tools appear as `mcp_<server>_<tool>` in Settings.
- **Cron** — schedule agent runs at `/cron` using cron expressions (`0 9 * * *`, `@hourly`, etc.).

## Status

Phase 1 (single-tenant minimal). See `docs/SPEC.md` §15 for the roadmap.
