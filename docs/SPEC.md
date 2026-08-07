# Swiflow — Development Specification

> **Status:** Canonical **product contract** document (API shapes, data model,
> security rules, phased roadmap). For the **as-implemented** agent runtime
> (wiring, stop policy details, concurrency, gaps), see
> [`AGENT_ARCHITECTURE.md`](AGENT_ARCHITECTURE.md). Prefer code + that doc when
> they diverge from older phrasing here.
>
> **Version:** 1.1 (Phase 1 detailed; MCP / cron / window tools landed; subagents
> still deferred).

---

## 1. Overview & goals

Swiflow is a **self-hosted AI agent runtime**. A single server process hosts
configurable LLM-backed agents, exposes them to clients over an HTTP API with
Server-Sent Events (SSE) streaming, and lets agents use tools — filesystem
operations, web access, shell execution, and skills — scoped to a workspace.

**Target users:** developers and small teams who want to run conversational,
tool-using agents on their own infrastructure, behind their own auth, with
durable conversation history.

**Deployment model (Phase 1):** a single binary serving HTTP + SSE, backed by a
local SQLite file, serving one tenant (single-user, shared-token auth). One
process; restart-safe. Later phases add multi-tenancy and a second database
backend.

**Goals**
- Run agents that stream assistant output and reasoning to clients in real time.
- Support tool use within a run, with configurable per-tool policy.
- Persist conversation history in a queryable store.
- Plug in any OpenAI-compatible chat-completions endpoint as a provider, with
  secrets encrypted at rest.
- Ship a Vue UI for chatting with agents and managing configuration.

**Non-goals (original Phase 1 scope; status as of v1.1)**
- Multi-tenancy, RBAC, users/membership (**still Phase 3**).
- Nested subagent deepen tools / reading child-session transcripts (**deferred**;
  `delegate_task` summary-only subagents **have landed**).
- Voice/audio, knowledge graphs, RAG, memory vaults.
- A first-party LLM. Swiflow is provider-agnostic.
- Training or fine-tuning.
- Clarify mid-run protocol; automatic hard acceptance gates; unattended
  `system_extra` mutation (see `AGENT_ARCHITECTURE.md` / `AGENT_WORKFLOW_PATTERNS.md`).

Mid-run **message queue**, per-round **obs** + run gates, `todo_*` /
`skill_draft` + draft confirm UI, and browser/window tools have **landed**.

---

## 2. Clean-room rules & references

Swiflow is a **clean-room** implementation. The rules below are binding for anyone
implementing from this document.

**References (permissively licensed, may be studied):**
- **hermes-agent** — `NousResearch/hermes-agent`, MIT, Python. Primary
  architecture reference (agent loop, skills, gateway shape, scheduled
  automations, subagents).
- **ZeroClaw** — `zeroclaw-labs/zeroclaw`, MIT OR Apache-2.0, Rust. Secondary
  reference (security hardening, sandboxing, observe approach).

**Authoring boundary:** implementers work only from this document, the public
standards it cites, and the permitted references above. Other private or
third-party agent runtimes are out of scope and must not be read or copied.
Where a design choice here resembles one elsewhere, it is an independent
decision recorded in this SPEC.

**The wall:**
- The spec author may be informed by the references above. The implementer works
  only from this document and the public standards.
- This document describes **what** the system does and the contracts it must
  satisfy; it does not prescribe literal code. Identifiers in this document are
  domain labels for readability — the implementer chooses actual code
  identifiers, and is expected to use Swiflow’s own naming.
- Implementation is written from scratch against this document, not produced by
  transforming, renaming, slimming, or refactoring any existing third-party file.

**Provenance:** a `PROVENANCE.md` at the repo root records, per module, that it
was authored from this spec and lists any permitted reference consulted. A
`NOTICE.md` retains the license/notice of any code actually copied from a
permissive reference (expected: none — fresh Go).

**Licensing:** the resulting codebase is the user's to license (commercial
product). Third-party agent-runtime obligations do **not** attach to Swiflow
when the clean-room rules are followed; an independent reviewer should audit
against this SPEC.
---

## 3. Tech stack & conventions

**Backend:**
- Go 1.22+ (toolchain 1.26 acceptable). Prefer the standard library.
- HTTP: `net/http` with the Go 1.22+ `ServeMux` method-pattern routing.
- DB: `database/sql` + `jmoiron/sqlx` for scan ergonomics.
- SQLite driver: `modernc.org/sqlite` (pure-Go, **no CGO**) — keeps the build
  hermetic and cross-compiles trivially. Driver registers as `sqlite`.
- Postgres driver (Phase 3): `github.com/jackc/pgx/v5` + stdlib adapter.
- Crypto: Go stdlib `crypto/aes` + `crypto/cipher` (AES-256-GCM).
- IDs: UUIDv7 as text (lexicographically sortable). Generate with
  `github.com/google/uuid`.
- Streaming: Server-Sent Events (SSE) over HTTP/1.1 chunked transfer.

**Frontend:**
- Vue 3 (Composition API) + Vite + Pinia + Vue Router + Tailwind CSS.
- Markdown: `markdown-it` + `highlight.js`. SSE via `fetch` streaming (not
  `EventSource`, because chat is a POST).

**Conventions:**
- Module path: `swiflow` (single word). Internal packages under `internal/`.
- Time stored as ISO-8601 text (SQLite) / `TIMESTAMPTZ` (Postgres).
- All API JSON is `snake_case` for keys.
- Errors: handlers return `{"error": "message"}` with an appropriate HTTP status.
- Logging: `log/slog` (structured, text handler by default).
- No external service dependencies in Phase 1 (no Redis, no broker).

---

## 4. Architecture

Layered. Dependencies point downward.

```
cmd/swiflow              CLI entrypoint (serve, migrate)
internal/server       HTTP REST + SSE, auth, CORS, static embed  ──┐
internal/agent        run loop, session guard, tool execution     │
internal/llm          provider interface + OpenAI-compatible      │
internal/tool         tool interface + registry + built-ins       │
internal/skill        skill discovery + summary                   │
internal/secure       SSRF guard, path sandbox, AES-GCM           │
internal/store        Store interface                                             │
  internal/store/sqlite  SQLite implementation                    │
internal/migrate      applies embed/schema.sql + upgrades                          │
internal/config       config loading (JSON + env)                                 │
embed/                embedded schema.sql + upgrades/*.sql + init-skills/         │
webui/                Vue app (built to webui/dist, embedded into binary)
```

**Module responsibilities & allowed dependencies:**
- `config` — loads config; depends on nothing internal.
- `migrate` — applies embedded `embed/schema.sql`; depends on `database/sql`.
- `store` — defines the `Store` interface and types; `store/sqlite` implements
  it. Depends on `database/sql`, `sqlx`. No dependency on `agent`/`server`.
- `secure` — pure helpers (SSRF, sandbox, crypto); depends on nothing internal.
- `llm` — `Provider` interface + OpenAI-compatible client; depends on `secure`
  (for validating provider base URLs) only if desired; otherwise standalone.
- `tool` — `Tool` interface, `Registry`, and built-in tools; depends on
  `secure`, `skill` (for skill tools). Tools do not depend on `agent`.
- `skill` — discovery + summary; depends on `config`-like paths only.
- `agent` — the `Runner`; depends on `llm`, `tool`, `store`, `skill`, `secure`.
- `server` — HTTP layer; depends on `agent`, `store`, `config`, `secure`.
- `cmd/swiflow` — wires everything; depends on all.

**Key invariant:** `store` knows nothing about agent semantics; `agent` knows
nothing about HTTP; `server` knows nothing about LLM wire formats. This keeps
each layer substitutable and testable.

---

## 5. Data model

Phase 1 schema lives in `embed/schema.sql` (also documented in
`docs/schema.sql`). Summary and rationale:

- `schema_migrations(version, applied_at)` — tracks applied files in
  `embed/upgrades/`.
- `providers(id, name, display_name, api_base, api_key_enc, enabled, created_at,
  updated_at)` — `name` unique; `api_key_enc` is `nonce || ciphertext` from
  AES-256-GCM; `enabled` is 0/1.
- `agents(id, key, display_name, provider, model, system_extra, created_at,
  updated_at)` — `key` unique; `provider` references `providers.name` (not a
  hard FK, so providers can be swapped); `system_extra` is extra system-prompt
  text appended after the auto-generated system prompt.
- `sessions(id, key, agent_key, title, created_at, updated_at)` — `key` unique,
  client-supplied; `agent_key` references `agents.key`; `title` nullable.
- `messages(id, session_id, seq, role, content, thinking, tool_calls_json,
  tool_call_id, tool_name, created_at)` — normalized: one row per turn.
  `seq` is monotonic per session from 1; `UNIQUE(session_id, seq)`. `role` ∈
  system/user/assistant/tool. `tool_calls_json` holds a JSON array of the
  assistant's tool-call requests (empty for non-tool-call assistant turns).
  `tool_call_id`/`tool_name` are set only for `role='tool'` rows.
- `tool_policy(tool_name, enabled)` — absent row means enabled.
- `skill_disabled(slug)` — absent row means enabled.

**Why normalized `messages` (not a JSON blob):** enables Phase 2 full-text
search, cheaper incremental appends, and per-message inspection in the UI
without rewriting a whole history blob.

**Schema initialization:** `embed/schema.sql` is embedded into the binary.
All statements use `CREATE IF NOT EXISTS`, so the base schema is idempotent.
Incremental changes go in `embed/upgrades/` as `0001_*.sql`, `0002_*.sql`, …;
`migrate.Apply` applies the base schema, then any unapplied upgrade files in
order (recorded in `schema_migrations`). `swiflow migrate` applies both and exits;
`swiflow serve --migrate` applies then serves.

**Postgres compatibility (Phase 3):** keep SQL dialect-portable: use
`TEXT`/`INTEGER`/`BLOB`; avoid SQLite-only functions in schema DDL except
`datetime('now')` (wrapped per-backend). UUIDs stored as `TEXT`. Booleans as
`INTEGER` 0/1. The Phase 3 `store/pg` implementation will use the same
`Store` interface.

---

## 6. Module specifications

### 6.1 `config`
Loads a JSON file (path from `-c` flag or `SWIFLOW_CONFIG` env, default
`config.json`), then applies env overlays (env wins). Falls back to defaults for
missing fields. See §11 for the full key list.

```go
type Config struct {
    HostAddress, DatabaseDSN string
    WorkspaceDir, InitSkillsDir, UserSkillsDir string
    AllowedOrigins []string
    Tools          ToolsConfig
}
func Load(path string) (Config, error)
```

### 6.2 `store`
Interface abstracting persistence so Postgres can drop in later.

```go
type Store interface {
    // providers
    CreateProvider(ctx, *Provider) error
    ListProviders(ctx) ([]Provider, error)
    GetProviderByName(ctx, name string) (*Provider, error)
    UpdateProvider(ctx, id string, fields map[string]any) error
    DeleteProvider(ctx, id string) error
    // agents
    CreateAgent(ctx, *Agent) error
    ListAgents(ctx) ([]Agent, error)
    GetAgentByKey(ctx, key string) (*Agent, error)
    UpdateAgent(ctx, id string, fields map[string]any) error
    // sessions + messages
    CreateSession(ctx, *Session) error
    GetSessionByKey(ctx, key string) (*Session, error)
    ListSessions(ctx) ([]Session, error)
    UpdateSessionTitle(ctx, key, title string) error
    AppendMessage(ctx, sessionKey string, msg Message) error  // assigns seq
    ListMessages(ctx, sessionKey string) ([]Message, error)
    // policy
    ToolEnabled(ctx, name string) bool
    SetToolEnabled(ctx, name string, enabled bool) error
    ListToolPolicy(ctx) ([]ToolPolicy, error)
    DisabledSkills(ctx) ([]string, error)
    SetSkillEnabled(ctx, slug string, enabled bool) error
}
```

`Provider.APIKey` is the **plaintext** key in memory; the sqlite impl
encrypts/decrypts at the boundary using `secure`. `Message` mirrors the table
row. `AppendMessage` reads `MAX(seq)` for the session and inserts at `seq+1`
atomically (transaction).

### 6.3 `secure`
Pure helpers:
- `Encrypt(key []byte, plaintext []byte) ([]byte, error)` — returns
  `nonce || ciphertext`; `Decrypt` inverses. Key is 32 bytes (AES-256); derived
  from config `EncryptionKey` via SHA-256 if not already 32 bytes.
- `CheckURL(rawURL string) error` — SSRF guard (§12). Resolves host; blocks
  private/loopback/link-local/CGNAT/metadata endpoints.
- `SandboxPath(workspace, requested string) (string, error)` — resolves
  `requested` against `workspace`, rejects traversal (`..` or absolute escapes),
  returns the cleaned absolute path or an error.

### 6.4 `llm`
```go
type ToolCall struct {
    ID        string
    Name      string
    Arguments map[string]any
}
type Message struct {
    Role       string // system|user|assistant|tool
    Content    string
    Thinking   string
    ToolCalls  []ToolCall
    ToolCallID string
    ToolName   string
}
type ToolDef struct {
    Name, Description string
    Parameters        map[string]any // JSON Schema
}
type ChatRequest struct {
    Model    string
    Messages []Message
    Tools    []ToolDef
}
type StreamChunk struct {
    Content  string
    Thinking string
    Done     bool
}
type ChatResponse struct {
    Content, Thinking string
    ToolCalls         []ToolCall
    FinishReason      string // stop|tool_calls|length|error
}
type Provider interface {
    Chat(ctx, ChatRequest) (*ChatResponse, error)
    ChatStream(ctx, ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error)
}
```
`NewOpenAICompatibleProvider(name, apiBase, apiKey, defaultModel) *OpenAIProvider`.
Behavior in §7 and the run loop. Base URL validated via `secure.CheckURL` is
**not** required (providers may legitimately point to private inference
servers), but the URL must be absolute http(s).

### 6.5 `tool`
```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]any        // JSON Schema
    Execute(ctx, args map[string]any) (string, error)
}
type Registry struct{ ... }
func NewRegistry() *Registry
func (r *Registry) Register(Tool)
func (r *Registry) Get(name string) (Tool, bool)
func (r *Registry) Definitions() []llm.ToolDef          // all enabled tools
func (r *Registry) SetEnabled(name string, enabled bool)
func (r *Registry) Execute(ctx, name string, args map[string]any) (string, error)
```
`Execute` checks enable state, looks up the tool, runs it with **panic recovery**
(returning a tool-error string on panic), and returns the result. The registry
holds the in-memory tool instances; enable/disable policy is persisted via
`store` and mirrored into the registry at startup and on change.

### 6.6 `skill`
Discovers built-in skills from an embedded FS (`embed/init-skills/`, compiled into
the binary) and user skills from `UserSkillsDir`. Optionally `InitSkillsDir`
overrides embedded builtins from disk (local development). A skill is a directory
containing a `SKILL.md` (or `skill.md`) with YAML-ish front matter:
```
---
slug: summarize
name: Summarize
description: Summarize a file or URL
---
<body: instructions>
```
`Discover(ctx) ([]Skill, error)` returns skills with
`{Slug, Name, Description, Source(init|user), Path}`. `Summary(skills) string`
produces a markdown list injected into the system prompt. Disabled slugs (from
`store.DisabledSkills`) are excluded.

### 6.7 `agent`
```go
type Event struct {
    Type      string         // delta|thinking|tool_call|tool_result|done|error|ui_request
    Content   string
    Thinking  string
    ID, Name  string
    Arguments map[string]any
    Result    string
    IsError   bool
    Error     string
    Title     string
}
type Runner struct{ ... }
func NewRunner(deps RunnerDeps) *Runner
func (r *Runner) Run(ctx, sessionKey, agentKey, userMessage string, onEvent func(Event)) error
func (r *Runner) Abort(sessionKey string) bool
func (r *Runner) IsBusy(sessionKey string) bool
```
Semantics in §7.

### 6.8 `server`
HTTP layer. Wires `store`, `agent.Runner`, `tool.Registry`, `skill.Catalog`,
`config`. See §10. Auth middleware validates the bearer token; CORS middleware
applies `AllowedOrigins`; static handler serves embedded `webui/dist` for non-API
routes.

### 6.9 `migrate`
`func Apply(ctx, db, schemaSQL string, upgradesFS fs.FS) error` — executes the
embedded base schema, then applies unapplied `NNNN_*.sql` files from
`embed/upgrades/` in order, recording each in `schema_migrations`.

---

## 7. The agent run loop

This is the system's core. The `Runner.Run` behavior, precisely:

**Inputs:** context `ctx`, session key, agent key, user message text, event
callback `onEvent`.

**Setup:**
1. Resolve agent by key from `store`. If missing → emit `error`, return.
2. Resolve provider by `agent.Provider`. Build the `llm.Provider` client (cached
   per provider name; invalidated when the provider row changes — see §7.3).
3. Acquire the session guard for `sessionKey` (§7.2). If busy → caller should
   have enqueued (HTTP layer); `Run` returns `ErrBusy`. Also enforce optional
   `max_concurrent_runs`. Register the run's `ctx` cancel for `Abort`.
   Each tool `Execute` is wrapped with `tool_timeout_sec` (default 120s).
   Structured observe: round/tool start-end via `internal/observe`.
4. Load session (create if absent, binding `agent_key`). Load history
   (`store.ListMessages`).
5. **Persist the user message immediately** (`AppendMessage`, role=user). This
   happens before any LLM call so a first-round failure does not lose the user's
   input.
6. Build the LLM message list:
   - `system`: the generated system prompt (§7.1).
   - then each history message (excluding the just-added user message) mapped to
     `llm.Message` (tool-call arrays and tool-result fields preserved).
   - then the user message.
7. Build tool definitions from `tool.Registry.Definitions()` (only enabled
   tools).

**Round loop** (`for round := 0; round < MaxRounds; round++`, `MaxRounds = 32`):
1. Call `provider.ChatStream(ctx, req, onChunk)`. In the chunk callback:
   - `Thinking != ""` → emit `thinking` event.
   - `Content != ""` → emit `delta` event.
2. On error → emit `error`, release guard, return.
3. If `resp.ToolCalls` is non-empty **and tools were offered this round**:
   - Loop-detection (§7.4). If triggered → set a forced no-tool wrap-up for the
     **next** round (do not hard-`error`).
   - Build an assistant `Message` (content + thinking + tool calls), persist it.
   - For each tool call (in order):
     - Emit `tool_call` event (id, name, arguments).
     - If `store.ToolEnabled(name)` is false → result is an error string
       "tool <name> is disabled". Else `registry.Execute(ctx, name, args)`.
     - On error → result = `"error: " + err`; `IsError=true`.
     - Emit `tool_result` event (id, name, truncated result, isError).
     - Build a tool `Message` (role=tool, content=result, tool_call_id,
       tool_name), persist it, append to the LLM message list.
   - If three consecutive tool errors → forced wrap-up next round.
   - `continue` to the next round.
4. Else (no tool calls — final answer, or wrap-up with tools withheld):
   - Persist the assistant message (content + thinking).
   - Set session title if unset: title = first ~60 chars of the first user
     message (truncated, no LLM call). Emit `done` with `Title` if newly set.
   - Emit `done`. Release guard. Return.

**Stop policy (goal first, budget last):**
1. **Goal reached** — model returns a final answer with no tool calls → `done`.
2. **Diverged / stalled** — identical tool calls repeated 3×, or 3 consecutive
   tool errors → force a no-tool wrap-up that explains the stuck state (not a
   hard `error`).
3. **Soft budget** — after ~75% of rounds, nudge the model to prefer finishing.
4. **Hard safety fuse** — last round withholds tools and asks for a progress
   summary. `MaxRounds` (default 32) is only a runaway guard, not the normal
   completion signal.

**Event ordering guarantees:** `delta`/`thinking` stream during a round;
`tool_call` precedes its `tool_result`; exactly one terminal event (`done` or
`error`) per run, emitted last.

### 7.1 System prompt
```
You are Swiflow agent <agent.key>.
<agent.system_extra, if nonempty>

## Workspace
Workspace root: <abs path>. File tools are restricted to it.
User messages may cite files as @/relative/path (@/ = workspace root). UI uploads append a block:
[UPLOAD FILES START]
@/path
[UPLOAD FILES END]
Resolve with fs_*. The chat UI strips this block from the visible bubble and shows path chips instead.

## Skills
<skill.Summary(discovered - disabled); omitted entirely if empty>

## When to stop
<goal-first stop guidance>

## Scheduling
<schedule_run / schedule_create guidance>

## Skill authoring
<skill_manage guidance>
```
See `AGENT_ARCHITECTURE.md` §3.4 for the exact implemented wording.
### 7.2 Session guard (single-run-per-session)
At most one run is active per session key. Implementation: an in-memory
`map[string]struct{}` (the "busy set") guarded by a mutex, plus a
`map[string]context.CancelFunc` for abort. `Run` does a `TryClaim` (insert into
busy set; if already present → busy). On any exit path, the run removes its key
from the busy set and the cancel map. `Abort(key)` looks up the cancel func,
calls it, and removes both entries. `IsBusy` checks the busy set.

This is stricter and simpler than nesting interrupts into the round loop: a
second chat on a busy session is **queued** (HTTP **202**) rather than rejected.
`Abort` cancels the in-flight run but **retains** the FIFO; when the run exits,
`drainQueue` starts the next message as a new `Run`. Chat events are also
published to `sesshub` so `/watch` clients see auto-continued turns. A global
`max_concurrent_runs` gate (0 = unlimited) rejects new runs with HTTP **409**
when full (busy sessions still enqueue).

### 7.3 Provider/agent cache invalidation
The `Runner` may cache the resolved `llm.Provider` per provider name. The cache
must be invalidated when:
- a provider row is created/updated/deleted, or
- an agent's `provider` or `model` is updated,
so a subsequent run uses fresh config. The `server` calls
`runner.InvalidateProvider(name)` and/or `runner.InvalidateAll()` after the
corresponding REST mutations. There is **no** `InvalidateAgent` API; changing an
agent's provider/model either invalidates all providers or relies on the next
resolve reading fresh agent rows (provider client remains cached by name).

### 7.4 Tool-loop detection
Compute a key = JSON of the current round's tool-call set (sorted by tool call
id). If it equals the previous round's key, increment a repeat counter; when the
counter reaches 3, **force a no-tool wrap-up** on the next round (stall nudge),
instead of terminating with `error`. Reset the counter when the key changes.
Three consecutive tool execution errors also enter this wrap-up path.

### 7.5 Tool-result truncation
Tool results fed back to the LLM are truncated to 4000 characters
(`result[:4000] + "\n...[truncated]"`) in the `tool_result` event **and** the
persisted tool message. (The full result is not retained in Phase 1; the
truncated form is what the model sees and what is stored.)

---

## 8. Built-in tools

Each tool advertises a JSON Schema for its parameters. Names use underscores
(e.g. `fs_read`) to namespace categories while matching provider function-name
patterns (`^[a-zA-Z0-9_-]+$`); dots are not permitted by OpenAI-compatible APIs.

### `fs_read`
- Description: "Read a UTF-8 text file from the workspace."
- Parameters: `{ "type":"object", "properties":{ "path": {"type":"string"} },
  "required":["path"] }`
- Behavior: resolve `path` via `secure.SandboxPath(workspace, path)`; read file
  (cap 256 KB; truncate with a marker if larger); return content.
- Security: traversal/absolute-escape → error.

### `fs_write`
- Description: "Write text to a file in the workspace (creates/overwrites)."
- Parameters: `path` (string, required), `content` (string, required).
- Behavior: sandbox-resolve; create parent dirs; write. Returns "wrote <path>".

### `fs_list`
- Description: "List entries in a workspace directory."
- Parameters: `path` (string, default ".").
- Behavior: sandbox-resolve; return one entry per line, `/`-suffixed for dirs.

### `fs_edit`
- Description: "Replace a unique old string with a new string in a file."
- Parameters: `path` (string), `old` (string), `new` (string), all required.
- Behavior: sandbox-resolve; read; require `old` to occur exactly once; replace;
  write. Error if not found or not unique.

### `web_fetch`
- Description: "Fetch a URL and return its text content."
- Parameters: `url` (string, required), `max_chars` (int, default 20000).
- Behavior: `secure.CheckURL(url)`; HTTP GET (10s timeout, max 5 MB); strip HTML
  to text (naive tag strip is acceptable for Phase 1); truncate to `max_chars`.
- Security: SSRF guard enforced; non-http(s) rejected.

### `web_search`
- Description: "Search the web and return titles, URLs, and snippets."
- Parameters: `query` (string, required), `limit` (int, default 5, max 10).
- Behavior: disabled when `tools.search_provider` is empty (returns
  "web search is not configured"). Supported providers:
  - `duckduckgo` — HTML results (no API key); falls back to Instant Answer API
  - `brave` — Brave Search API; requires `tools.search_api_key`
  - `searxng` — self-hosted SearXNG; requires `tools.search_base_url`
  Returns a numbered list of title / URL / snippet. Env overlays:
  `SWIFLOW_SEARCH_PROVIDER`, `SWIFLOW_SEARCH_API_KEY`,
  `SWIFLOW_SEARCH_BASE_URL`.

### `exec`
- Description: "Run a shell command in the workspace."
- Parameters: `command` (string, required), `timeout` (int seconds, default 30).
- Behavior: gated by `config.Tools.ExecEnabled` (default false). If disabled at
  config level, `exec` is not registered. When enabled, run with `sh -c` in the
  workspace dir, capture stdout+stderr, cap output 256 KB, enforce timeout via
  `context`. Returns combined output.

### `browser`
- Description: "Browser automation against pages (when enabled)."
- Gated by `tools.browser_enabled` / `browser_headless`. See implementation in
  `internal/tool/browser.go`.

### `schedule_run`
- Description: "Re-invoke the agent in the current session after a delay."
- Parameters include `delay_seconds` and `message` (as a new user turn). Publishes
  through `sesshub` for `/watch` subscribers.

### `schedule_create`
- Description: "Create a recurring cron job."
- Parameters include name/schedule expression and message payload.

### `window_opened`
- Description: "List file tabs currently open in the user's Swiflow window."
- Parameters: none (`{}`).
- Behavior: SSE RPC `ui_request` to the connected UI; UI returns JSON
  `{ files:[{path,title}], count }` (Welcome/Explore/Settings excluded).
  Errors if no UI is bound for the session or reply times out (~8s).

### `window_active`
- Description: "Get the file tab currently focused in the user's Swiflow window."
- Parameters: none (`{}`).
- Behavior: same RPC; UI returns `{ path, title }` or `{ path:null, reason }`.

### `window_open`
- Description: "Open (or focus) a workspace file in the user's Swiflow window."
- Parameters: `path` (string, required, workspace-relative).
- Behavior: sandbox-resolve; require an existing non-directory file; then SSE
  RPC so the UI calls `openFile(path)`. Returns `{ opened:true, path }`.

### `skill_use`
- Description: "Load and apply a skill's instructions by slug."
- Parameters: `slug` (string, required), `input` (string, optional).
- Behavior: find skill by slug among discovered (minus disabled); return its
  body (and note the input). The model is expected to follow the returned
  instructions.

### `skill_search`
- Description: "Search available skills by keyword."
- Parameters: `query` (string, required).
- Behavior: naive substring match over discovered skills' name+description;
  return matching slugs + descriptions.

### `skill_manage`
- Description: "Create or patch user skills (SKILL.md)."
- Parameters: `action` (`create` \| `patch`) and content/edit fields. User skills
  override built-ins by slug.

### `skill_draft`
- Description: "Save a skill draft for human review (does not write user-skills)."
- Parameters: `slug`, `content` (SKILL.md body), optional `note`.
- Behavior: writes under `user_skills/.drafts/`; UI Accept promotes to user skill.

### `todo_write` / `todo_read`
- Session-scoped checklist (in-memory). Long tasks should maintain items via
  system guidance. Acceptance before `done` is **prompt policy**, not a hard gate.

### `clarify`
- Description: "Ask the user a clarifying question and wait for their answer."
- Parameters: `question` (required), `options` (optional string[]), `allow_free_text` (bool, default true).
- Behavior: emits SSE `ui_request` `{name:"clarify",...}` via the window bridge; waits up to
  15 minutes for `POST /api/window/reply` with `{"answer":"..."}`. Session stays busy while
  waiting; mid-run user messages still queue. Abort cancels the wait. Subagents cannot use
  `clarify`.

### Subagent tools (`subagent_spawn` / `subagent_status` / `subagent_wait`)
- **`subagent_spawn`**: Start **one** async sub-agent for a batch; returns immediately with `{child_session, status: running}`.
  Parameters: `goal` (required), `context` (optional), `max_rounds` (default 10, max 16).
- **`subagent_status`**: Non-blocking progress read (todos, last_action, metrics; summary/artifacts when terminal).
  Parameter: `child_session` (required).
- **`subagent_wait`**: Block until child terminal or timeout (default 900s). Allowed only when exactly one subagent is still running for the parent session.
  Parameters: `child_session` (required), `timeout_seconds` (optional, max 900).
- Behavior: child `sessionKey` `sub-{parent}-{id}`; child cannot use subagent tools or `clarify`; parent abort cancels children; child runs in background goroutine; progress via `subagent_progress` / `subagent_done` SSE on parent session.

**Panic recovery:** `Registry.Execute` wraps every tool call in a recover; a
panic becomes a tool-error result (`"error: panic: <msg>"`), not a run crash.

---

## 9. Skill system

- **Built-in:** embedded in the binary (`embed/init-skills/`); read-only at runtime.
- **User:** `UserSkillsDir` (default `./data/user-skills`), user-writable; overrides
  built-in skills by slug.
- **Dev override:** optional `InitSkillsDir` / `SWIFLOW_INIT_SKILLS` replaces embedded
  builtins from a filesystem directory (rebuild not required).
- **Format:** each skill is a directory with a `SKILL.md` containing YAML
  front matter (`slug`, `name`, `description`) and a markdown body.
- **Discovery:** walk both dirs, parse front matter, dedupe by slug (init first,
  user overrides). `Source` records `init` or `user`.
- **Summary injection:** `skill.Summary` produces a markdown list
  (`- slug: name — description`) of enabled skills, injected into the system
  prompt (§7.1). Disabled slugs (from `store.DisabledSkills`) are excluded.
- **Disable/enable:** `PUT /api/skills/{slug}` with `{"enabled": bool}` writes
  to / removes from `skill_disabled`.
- **Reload:** `POST /api/skills/reload` re-runs discovery (picks up new files)
  and clears the in-memory disabled cache. Called automatically after
  enable/disable.

---

## 10. HTTP + SSE protocol

All API routes are prefixed `/api`. Auth: `Authorization: Bearer <token>` where
token = `config.AuthToken`. Missing/invalid → `401`. CORS: `Access-Control-
Allow-Origin` reflects `AllowedOrigins` (or `*` if empty list). Static: non-`/api`
routes serve embedded `webui/dist` (SPA fallback to `index.html`).

### 10.1 Health
- `GET /api/health` → `200 {"status":"ok"}`

### 10.2 Providers
- `GET /api/providers` → `{"providers":[{id,name,display_name,api_base,enabled}]}` (no api_key).
- `POST /api/providers` body `{name, display_name?, api_base, api_key, enabled?}` → `201` provider (no api_key). Validates name unique, api_base absolute http(s).
- `GET /api/providers/{id}` → provider (api_key omitted).
- `PUT /api/providers/{id}` body with any of `{display_name, api_base, api_key, enabled}` → updated provider. If `api_key` omitted, key unchanged.
- `DELETE /api/providers/{id}` → `200 {"status":"deleted"}`. (Reject if an agent references it? Phase 1: allow; agent runs will error at resolve time. Simpler.)
- After any mutation: `runner.InvalidateAll()`.

### 10.3 Agents
- `GET /api/agents` → `{"agents":[agent]}`.
- `POST /api/agents` body `{key, display_name?, provider, model, system_extra?}` → `201`. Defaults `provider="openai"`, `model="gpt-4o-mini"`. Validates provider exists.
- `GET /api/agents/{key}` → agent.
- `PUT /api/agents/{key}` body with any of `{display_name, provider, model, system_extra}` → updated agent. Validates provider if changed.
- After any mutation: `runner.InvalidateAll()`.

### 10.4 Sessions & chat
- `GET /api/sessions` → `{"sessions":[{id,key,agent_key,title,created_at,updated_at}]}`.
- `GET /api/sessions/{key}` → `{"session":{...}, "messages":[message]}`.
- `POST /api/sessions/{key}/chat` body `{"message":"...", "agent_key":"..."?}`:
  - If session **idle** and under global concurrent cap → **SSE stream** (HTTP 200).
  - If session **busy** → HTTP **202** `{"queued":true,"position":N}` (1-based FIFO).
  - If global concurrent gate full (and not busy-enqueue) → HTTP **409**
    `{"error":"too many concurrent runs"}`.
  - `agent_key` optional (defaults to session's agent, or `"default"`). SSE headers:
  `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`. Each event is `data: <json>\n\n`. Event JSON shapes:
  - `{"type":"delta","content":"..."}` `{"type":"thinking","content":"..."}`
  - `{"type":"tool_call","id":"...","name":"...","arguments":{...}}`
  - `{"type":"tool_result","id":"...","name":"...","result":"...","isError":false}`
  - `{"type":"ui_request","id":"...","name":"window_opened|window_active|window_open","arguments":{...}}` — mid-run RPC to the UI (not persisted as a chat message); UI must `POST /api/window/reply`.
  - `{"type":"user","content":"..."}` — emitted on auto-dequeue (watch/hub).
  - `{"type":"queued","position":N}` — client-side convenience from 202 (not SSE).
  - `{"type":"done","title":"..."?}` `{"type":"error","error":"..."}`
  - A terminal event (`done`/`error`) ends the stream. Auto-continued queue runs
    publish via `sesshub`; clients use `/watch` after the original SSE closes.
- `POST /api/sessions/{key}/abort` → `200 {"aborted": bool}`. Cancels the in-flight
  run; **queue is retained** and drained after the run exits.
- `GET /api/sessions/{key}/watch` → SSE of `sesshub` events for live / auto-continue.

### 10.4.1 Window UI reply
- `POST /api/window/reply` body `{"id":"...","result":"..."?,"error":"..."?}` → `200 {"ok":true}`.
  Completes a pending `ui_request` from `window_*` tools. `result` is a JSON
  string forwarded to the tool; `error` fails the tool call.

### 10.5 Tools
- `GET /api/tools` → `{"tools":[{name, description, enabled, parameters}]}`.
- `PUT /api/tools/{name}` body `{"enabled": bool}` → `200`. Persists to `tool_policy`, mirrors into registry.

### 10.6 Skills
- `GET /api/skills` → `{"skills":[{slug,name,description,source,enabled}]}`.
- `PUT /api/skills/{slug}` body `{"enabled": bool}` → `200`.
- `POST /api/skills/reload` → `200 {"status":"reloaded"}`.
- `GET /api/skills/drafts` → `{"drafts":[{id,slug,note,content,created_at}]}`.
- `POST /api/skills/drafts/{id}/accept` → promotes draft into `user_skills` + reload.
- `DELETE /api/skills/drafts/{id}` → discard draft. Never auto-mutate `system_extra`.

---

## 11. Configuration

JSON file with env overlay (env wins). Defaults shown.

| Key (JSON) | Env | Default | Notes |
|---|---|---|---|
| `host_address` | `SWIFLOW_HOST_ADDRESS` | `127.0.0.1:8000` | listen `host:port` |
| `database_dsn` | `SWIFLOW_DATABASE_DSN` | `sqlite://./data/swiflow.db` | `sqlite://…` or `postgres://…` |
| `workspace_dir` | `SWIFLOW_WORKSPACE` | `./data/workspace` | file-tool sandbox root |
| `init_skills_dir` | `SWIFLOW_INIT_SKILLS` | (empty) | dev override for built-in skills |
| `user_skills_dir` | `SWIFLOW_USER_SKILLS` | `./data/user-skills` | user skills |
| `allowed_origins` | — | `[]` | CORS; empty = allow all |
| `web_dist_dir` | — | (embedded) | override for dev |
| `max_history_msgs` | — | `100` | under `context`; truncate chat history per run |
| `max_context_chars` | `SWIFLOW_MAX_CONTEXT_CHARS` | `120000` | under `context`; in-memory LLM prompt budget (0 = proactive fit off) |
| `max_concurrent_runs` | `SWIFLOW_MAX_CONCURRENT_RUNS` | `0` | under `context`; global in-flight Run cap; `0` = unlimited |
| `tool_timeout_sec` | `SWIFLOW_TOOL_TIMEOUT_SEC` | `120` | under `context`; default per-tool deadline |
| `disable_thinking` | `SWIFLOW_DISABLE_THINKING` | `true` | under `context`; GLM thinking off |
| `tools.exec_enabled` | `SWIFLOW_EXEC` | `false` | register `exec` if true |
| `tools.browser_enabled` | — | `false` | register `browser` if true |
| `tools.browser_headless` | — | `true` | browser headless mode |
| `tools.search_provider` | `SWIFLOW_SEARCH_PROVIDER` | `""` | `duckduckgo` \| `brave` \| `searxng`; empty disables |
| `tools.search_api_key` | `SWIFLOW_SEARCH_API_KEY` | `""` | Brave Search API key |
| `tools.search_base_url` | `SWIFLOW_SEARCH_BASE_URL` | `""` | SearXNG base URL |

`config.example.json` ships documenting these. `Load` errors if `auth_token` or
`encryption_key` are empty.

---

## 12. Security

**SSRF guard** (`secure.CheckURL`), used by `web_fetch` (NOT by provider base
URLs, which may be private):
- Parse URL; require scheme `http` or `https`; require a host.
- Block hostnames: `localhost`, `*.localhost`, `*.local`, `*.internal`,
  `metadata.google.internal`, `169.254.169.254`.
- If host is an IP literal: block if private/loopback/link-local/CGNAT
  (`0.0.0.0/8`, `10.0.0.0/8`, `127.0.0.0/8`, `169.254.0.0/16`, `172.16.0.0/12`,
  `192.168.0.0/16`, `100.64.0.0/10`, and IPv6 `::1/128`, `fc00::/7`,
  `fe80::/10`).
- Else resolve host via `net.LookupHost`; block if any resolved address is
  private. (Note: this is a check-then-fetch; DNS rebinding is a known residual
  risk, acceptable for Phase 1. A future hardening can pin the resolved IP for
  the fetch.)

**Path sandbox** (`secure.SandboxPath`): join `workspace` + requested, clean,
then verify the result is within `workspace` (via `filepath.Rel` or prefix
check after `EvalSymlinks`). Reject `..` and absolute paths that escape. Used by
all `fs.*` tools.

**Encryption:** provider API keys encrypted with AES-256-GCM. The 32-byte key
is `SHA-256(config.EncryptionKey)`. Ciphertext stored as `nonce(12) || ct`.
Encrypt on write to `providers.api_key_enc`; decrypt on read into memory only.

**Auth (Phase 1):** single shared bearer token (`config.AuthToken`). Every
`/api/*` route requires it. No users, no RBAC. (Phase 3 adds users/roles/tenants.)

**Logging:** never log API keys or full secrets. Log only the last 4 chars if
needed for identification.

---

## 13. Frontend (Vue 3)

**Stack:** Vite + Vue 3 (Composition API) + Pinia + Vue Router + Tailwind. Build
to `webui/dist`, embedded into the Go binary via `//go:embed`.

**Pages:**
- **Chat** (`/`): left = session list (with titles); right = message stream for
  the selected session + composer. Messages render markdown (`markdown-it` +
  `highlight.js`). Tool calls/results render as collapsible blocks. Streaming
  deltas append to the in-progress assistant bubble.
- **Agents** (`/agents`): list + create/edit form (key, display_name, provider,
  model, system_extra).
- **Providers** (`/providers`): list + create/edit (name, display_name, api_base,
  api_key, enabled). API key field masked on edit.
- **Skills** (`/skills`): list with enable/disable toggles + reload; pending
  skill drafts (preview / accept / reject).
- **Settings** (`/settings`): read-only view of tools + their enable state with
  toggles.

**State (Pinia):**
- `sessionStore`: sessions list, current session, messages, streaming state.
- `agentStore`, `providerStore`, `skillStore`, `toolStore`: CRUD collections.

**API client:** a `fetch` wrapper that injects `Authorization: Bearer <token>`
(token entered in Settings, stored in localStorage) and prefixes `/api`. Chat
uses `fetch('/api/sessions/<key>/chat', {method:'POST', body, headers})` and
reads the response body as a `ReadableStream`, parsing SSE `data:` lines
incrementally (since `EventSource` cannot POST).

**Dev/prod:** `pnpm dev` runs Vite at `:5173` proxying `/api` to `:18800`; prod
build is embedded and served by Go.

---

## 14. Build & run

**Makefile targets:**
- `dev`: local API `:8000` + Vite `:5173` (parallel)
- `build`: `webui/dist` + `go build -o swiflow`
- `image`: `docker build -t swiflow:latest .`
- `migrate`: `go run ./cmd/swiflow migrate`
- `test`: web build + `go vet` + `go test` + `go build`

**Embedding:** `internal/server` (or `web`) has `//go:embed dist/*` over the
built `webui/dist`; the static handler serves it. For dev, set `web_dist_dir` to
the Vite dev path or run the UI separately.

**CLI:**
- `swiflow serve [--migrate] [-c config.json] [-v]` — start server; `--migrate`
  applies schema and upgrades first.
- `migrate` — apply schema and upgrades, then exit.
- `--migrate` runs `migrate.Apply` before serving.

---

## 15. Phased roadmap

**Phase 1 — single-tenant minimal:** SQLite, SSE chat, fs/web/exec/skill tools,
skills, Vue UI, shared-token auth.

**Phase 2 — extensibility (partially done):**
- **Done:** MCP client (stdio / SSE / streamable-http) → `tool.Registry` as
  `mcp_*`; cron scheduler + APIs/UI; `window_*` UI RPC; `web_search`; `browser`
  (gated); goal-first stop policy with stall wrap-up.
- **Not yet:** **subagents** (spawn an isolated run of another agent, collect its
  final answer). See evolution notes in `AGENT_ARCHITECTURE.md`.

**Phase 3 — multi-tenancy & Postgres:** `tenants`, `users`, `tenant_membership`,
roles (user/admin/owner); every entity gains `tenant_id`; per-tenant workspaces,
tool policy, skill config; `store/pg` Postgres backend behind the same `Store`
interface; RBAC-enforcing auth middleware replacing the shared token. SSE/REST
shapes extend with tenant/user context.

**Phase 4 — production hardening:** structured LLM call tracing + optional
OpenTelemetry OTLP export; prompt caching (Anthropic-style and OpenAI-compatible);
LLM-generated session titles; rate limiting; SSRF DNS-rebinding mitigation (pin
resolved IP); backups/restore; tighter resource bounds.

Amend this document when contracts change; keep `AGENT_ARCHITECTURE.md` aligned
with runtime behavior.

---

## 16. Verification plan

**Phase 1 end-to-end:**
1. `make migrate` against a temp SQLite file → `schema_migrations` and all
   tables created (compare to `embed/schema.sql`).
2. `make dev` or `./swiflow serve` → server listens; `curl /api/health` → `{"status":"ok"}`.
3. Create a provider (OpenAI-compatible endpoint + key) via `POST /api/providers`;
   confirm `GET` omits `api_key`. Confirm `api_key_enc` in DB is non-empty
   ciphertext.
4. Create an agent via `POST /api/agents` referencing the provider.
5. `POST /api/sessions/<key>/chat` with a message → observe SSE `delta` chunks →
   `done`. Confirm `messages` rows persisted (user + assistant). Kill the server
   mid-stream and restart → confirm the user message was still persisted (F5.8).
6. Ask the agent to list files → `tool_call` + `tool_result` events; ask it to
   read `../../etc/passwd` → `tool_result` with `isError:true` (sandbox).
7. Start two chats on the same session concurrently → second receives
   `session busy`.
8. `POST /api/sessions/<key>/abort` during a long run → run stops, `error`/closed
   promptly.
9. Update the agent's `model` via `PUT /api/agents/{key}` → next chat uses the
   new model (invalidate cache).
10. `make dev` → Vue UI lists agents/providers, chats end-to-end with
    streaming, tool calls render.
11. `go vet ./... && go build ./...` clean.

**Unit-level (recommended):** `secure` (SSRF cases, sandbox traversal cases),
`llm` stream parsing (feed a recorded SSE byte stream, assert tool-call
accumulation incl. non-contiguous indices), `agent` loop (mock provider: assert
tool loop → **wrap-up** after 3 repeats; assert max-rounds safety fuse; assert
user message persisted before first LLM call), `store` (AppendMessage seq
monotonicity, round-trip encrypt/decrypt of provider key).

---

## 17. Provenance & licensing

- **`PROVENANCE.md`** (repo root): per-module statement that the module was
  implemented from this SPEC, naming any permissive reference consulted
  (hermes-agent MIT, ZeroClaw MIT/Apache-2.0).
- **`NOTICE.md`**: attribution + license text for any code copied from a
  permissive reference (expected empty — fresh Go).
- **`LICENSE`**: the project's own license, chosen by the owner (commercial
  product). A placeholder is added now; the owner selects the final terms.
- **Third-party obligation:** none attaches to Swiflow when the clean-room rules
  in §2 are followed. An independent reviewer should audit against this SPEC.

---

## Appendix A — Open questions for the owner

1. **`LICENSE` choice** for Swiflow (proprietary / MIT / Apache-2.0 / other).
2. **SSE busy behavior** (§10.4): busy → HTTP **202** queue; global gate full →
   **409**. Clarify protocol still deferred.
3. **`web_search`:** implemented (`duckduckgo` / `brave` / `searxng` via
   `tools.search_*`). Closed.
4. **Session title generation** (§7): Phase 1 uses truncated first user message;
   LLM-generated titles move to Phase 4. Confirm.
5. **Default provider/model** names (`openai` / `gpt-4o-mini`) — confirm or
   change.
6. **Clarify / hard verify gates** — still deferred; mid-run queue + draft
   confirm have landed (see `AGENT_ARCHITECTURE.md` §5–6 and §10).
