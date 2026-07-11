# Provenance

Mira is a clean-room implementation. It is built solely from `docs/SPEC.md`,
with the following permissively-licensed projects used as architectural
references (studied, not copied unless a notice is recorded in `NOTICE.md`):

- **hermes-agent** — https://github.com/NousResearch/hermes-agent — MIT (Python).
  Primary reference: agent loop, skills, gateway shape.
- **ZeroClaw** — https://github.com/zeroclaw-labs/zeroclaw — MIT OR Apache-2.0
  (Rust). Secondary reference: security hardening, sandboxing, observability.

**Forbidden references.** No code in this repository derives from, and no
implementation work consulted, any of the following codebases:
`goclaw`/`nextlevelbuilder/goclaw`, `duo-claw`, `openclaw`, or `swiflow`/
`swiflow-new`. These are off-limits for the clean-room reimplementation. Where a
design decision in Mira resembles one of them, it is an independent choice made
from `docs/SPEC.md` and noted there.

## Per-module statement

Each module was authored from `docs/SPEC.md` and the public standards (HTTP,
SSE, SQL, the OpenAI chat-completions HTTP API). No module was produced by
transforming or renaming files from any forbidden codebase.

| Module | Authoring basis |
|---|---|
| `internal/config` | SPEC §6.1, §11 |
| `internal/secure` | SPEC §6.3, §12 |
| `internal/store` + `internal/store/sqlite` | SPEC §6.2, §5 |
| `internal/migrate` | SPEC §6.9, §5 |
| `internal/llm` | SPEC §6.4, §7 |
| `internal/skill` | SPEC §6.6, §9 |
| `internal/tool` | SPEC §6.5, §8 |
| `internal/agent` | SPEC §6.7, §7 |
| `internal/server` | SPEC §6.8, §10 |
| `cmd/mira` | SPEC §14 |
| `web/` | SPEC §13 |

## Audit

For maximum legal safety, a reviewer who has never seen the forbidden codebases
should audit the result against `docs/SPEC.md`. The Mira design intentionally
diverges from the forbidden codebases in module naming, the normalized
`messages` table, and the choice of SSE over WebSocket, to make that audit
straightforward.
