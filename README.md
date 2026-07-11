# Mira

A self-hosted AI agent runtime. Single Go binary + Vue UI, SQLite-backed,
OpenAI-compatible providers, tool use (filesystem / web / shell / skills),
SSE streaming. See `docs/SPEC.md` for the full development specification.

## Quick start

```bash
cp config.example.json config.json   # then edit auth_token / encryption_key
make build
./mira serve --migrate -v
```

UI (dev):
```bash
make web-install
make web-dev   # http://localhost:5173, proxies /api to :18800
```

## Status

Phase 1 (single-tenant minimal). See `docs/SPEC.md` §15 for the roadmap.
