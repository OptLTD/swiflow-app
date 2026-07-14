# Swiflow docs

| Document | Role |
|----------|------|
| [SPEC.md](SPEC.md) | Product / API / schema **contracts** and phased roadmap. Historical clean-room development basis; keep behavioral claims aligned with code. |
| [AGENT_ARCHITECTURE.md](AGENT_ARCHITECTURE.md) | **As-implemented** agent runtime: wiring, run loop, concurrency, SSE, tools, gaps, evolution. Prefer this when reading Go sources. |
| [IMPLEMENTATION_STATUS.md](IMPLEMENTATION_STATUS.md) | Short checklist of what landed vs still deferred. |
| [schema.sql](schema.sql) | Documented schema mirror; runtime applies [`embed/schema.sql`](../embed/schema.sql) + upgrades. |

**Related (outside docs/):**

- [`config.example.json`](../config.example.json) — config keys including search / exec / browser.
- [`embed/init-skills/`](../embed/init-skills/) — built-in skill packs (`window-context`, `example`).

**Reading order for the agent runtime:** `AGENT_ARCHITECTURE.md` → `SPEC.md` §6.7 / §7 / §8 / §10 → `internal/agent/agent.go`.
