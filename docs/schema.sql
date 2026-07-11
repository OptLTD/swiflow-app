-- Mira — Phase 1 schema (SQLite).
-- Canonical schema referenced by docs/SPEC.md §5. Source of truth for DDL is
-- initial/schema.sql (embedded into the binary). Designed to be portable to
-- PostgreSQL in Phase 3 (TEXT ids hold UUIDv7; INTEGER booleans become BOOLEAN;
-- datetime('now') becomes NOW()).

PRAGMA foreign_keys = ON;

-- Tracks applied upgrade scripts in initial/upgrades/ (not schema.sql itself).
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- LLM provider endpoints (OpenAI-compatible). API keys encrypted at rest.
CREATE TABLE IF NOT EXISTS providers (
    id           TEXT PRIMARY KEY,                 -- UUIDv7
    name         TEXT NOT NULL UNIQUE,             -- e.g. "openai", "local"
    display_name TEXT,
    api_base     TEXT NOT NULL,                    -- e.g. https://api.openai.com/v1
    api_key_enc  BLOB NOT NULL,                    -- AES-256-GCM ciphertext (nonce||ct)
    enabled      INTEGER NOT NULL DEFAULT 1,       -- 0/1
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Agent configurations. provider references providers.name.
CREATE TABLE IF NOT EXISTS agents (
    id           TEXT PRIMARY KEY,                 -- UUIDv7
    key          TEXT NOT NULL UNIQUE,             -- e.g. "default"
    display_name TEXT,
    provider     TEXT NOT NULL,                    -- references providers.name
    model        TEXT NOT NULL,                    -- e.g. "gpt-4o-mini"
    system_extra TEXT NOT NULL DEFAULT '',         -- extra system prompt text
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Conversation threads. agent_key references agents.key.
CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,                   -- UUIDv7
    key        TEXT NOT NULL UNIQUE,               -- client-supplied session key
    agent_key  TEXT NOT NULL,                      -- references agents.key
    title      TEXT,                               -- nullable; auto-set from first user msg
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Normalized message history (one row per turn). Enables Phase 2 search.
CREATE TABLE IF NOT EXISTS messages (
    id              TEXT PRIMARY KEY,              -- UUIDv7
    session_id      TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    seq             INTEGER NOT NULL,              -- monotonic per session, from 1
    role            TEXT NOT NULL CHECK (role IN ('system','user','assistant','tool')),
    content         TEXT NOT NULL DEFAULT '',      -- text content
    thinking        TEXT NOT NULL DEFAULT '',      -- reasoning text (assistant)
    tool_calls_json TEXT NOT NULL DEFAULT '',      -- JSON array; assistant tool-call requests
    tool_call_id    TEXT NOT NULL DEFAULT '',      -- for role='tool': the call id being answered
    tool_name       TEXT NOT NULL DEFAULT '',      -- for role='tool': the tool name
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(session_id, seq)
);
CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id, seq);

-- Per-tool enable/disable. Absent row => default (enabled).
CREATE TABLE IF NOT EXISTS tool_policy (
    tool_name TEXT PRIMARY KEY,
    enabled   INTEGER NOT NULL DEFAULT 1           -- 0/1
);

-- Disabled skills (global in Phase 1; per-tenant in Phase 3). Absent => enabled.
CREATE TABLE IF NOT EXISTS skill_disabled (
    slug TEXT PRIMARY KEY
);
