# Swiflow docs

| Document | Role |
|----------|------|
| [GETTING_STARTED.md](GETTING_STARTED.md) | Quick start, build, Docker, Skills config. |
| [cloudflare-worker-dl.js](cloudflare-worker-dl.js) | CDN Worker for `dl.swiflow.cc` (update.json + release-assets). |
| [SPEC.md](SPEC.md) | Product / API / schema **contracts** and phased roadmap. Historical clean-room development basis; keep behavioral claims aligned with code. |
| [AGENT_ARCHITECTURE.md](AGENT_ARCHITECTURE.md) | **As-implemented** agent runtime: wiring, run loop, concurrency, SSE, tools, gaps, evolution. Prefer this when reading Go sources. |
| [AGENT_WORKFLOW_PATTERNS.md](AGENT_WORKFLOW_PATTERNS.md) | **Learning guide**: Subagent / queue / clarify / observe / skill drafts — **landed** except deepen-read and queue durability. |
| [IMPLEMENTATION_STATUS.md](IMPLEMENTATION_STATUS.md) | Short checklist of what landed vs still deferred. |
| [SOFT_ASYNC_AND_DELEGATION.md](SOFT_ASYNC_AND_DELEGATION.md) | Soft-async tools + cost-probe handoff to `delegate_task`（场景 / 用途 / 逻辑）. |
| [schema.sql](schema.sql) | Documented schema mirror; runtime applies [`embed/schema.sql`](../embed/schema.sql) + upgrades. |

**Related (outside docs/):**

- [`config.example.json`](../config.example.json) — config keys including search / exec / browser.
- [`embed/init-skills/`](../embed/init-skills/) — built-in skill packs (`window-context`, `reflection-loop`).

**Reading order for the agent runtime:** `AGENT_ARCHITECTURE.md` → `AGENT_WORKFLOW_PATTERNS.md` (concepts) → `SOFT_ASYNC_AND_DELEGATION.md`（慢工具并行与委派）→ `SPEC.md` §6.7 / §7 / §8 / §10 → `internal/agent/agent.go`.
