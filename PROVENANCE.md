# Provenance

Swiflow is a clean-room implementation. It is built solely from `docs/SPEC.md`,
with the following permissively-licensed projects used as architectural
references (studied, not copied unless a notice is recorded in `NOTICE.md`):

- **hermes-agent** — https://github.com/NousResearch/hermes-agent — MIT (Python).
  Primary reference: agent loop, skills, gateway shape.
- **ZeroClaw** — https://github.com/zeroclaw-labs/zeroclaw — MIT OR Apache-2.0
  (Rust). Secondary reference: security hardening, sandboxing, observability.

**Authoring rule.** Implementation consults only this SPEC, the public standards
named therein, and the permitted references above. No module is produced by
transforming, renaming, or porting files from any other private or third-party
agent runtime.

## Per-module statement

Each module was authored from `docs/SPEC.md` and the public standards (HTTP,
SSE, SQL, the OpenAI chat-completions HTTP API).

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
| `cmd/swiflow` | SPEC §14 |
| `webui/` | SPEC §13 |

## Audit

For maximum legal safety, a reviewer who has not consulted unlisted third-party
agent runtimes should audit the result against `docs/SPEC.md`. Swiflow uses its
own module naming, a normalized `messages` table, and SSE (not WebSocket) to
keep the design independently readable.
