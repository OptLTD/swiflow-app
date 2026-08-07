-- Swiflow — SQLite schema (idempotent CREATE IF NOT EXISTS).
-- Applied at startup via migrate.Apply before any upgrade scripts.

PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS sys_migration (
    version    VARCHAR(64) PRIMARY KEY,
    applied_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS sys_tenant (
    id            VARCHAR(36) PRIMARY KEY,
    name          VARCHAR(64) NOT NULL UNIQUE,
    enabled       SMALLINT NOT NULL DEFAULT 1,
    password_hash TEXT NOT NULL DEFAULT '',
    created_at    DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS sys_user (
    id         VARCHAR(36) PRIMARY KEY,
    tid        VARCHAR(64) NOT NULL DEFAULT 'default',
    username   VARCHAR(64) NOT NULL UNIQUE,
    enabled    SMALLINT NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS sys_settings (
    key        VARCHAR(128) PRIMARY KEY,
    value      TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS llm_provider (
    id           VARCHAR(36) PRIMARY KEY,
    tid          VARCHAR(64) NOT NULL DEFAULT 'default',
    name         VARCHAR(64) NOT NULL,
    display      VARCHAR(128),
    api_base     TEXT NOT NULL,
    api_key      BLOB NOT NULL,
    model        VARCHAR(128) NOT NULL DEFAULT '',
    enabled      SMALLINT NOT NULL DEFAULT 1,
    created_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE(tid, name)
);

CREATE TABLE IF NOT EXISTS agent_config (
    id           VARCHAR(36) PRIMARY KEY,
    tid          VARCHAR(64) NOT NULL DEFAULT 'default',
    key          VARCHAR(64) NOT NULL,
    display      VARCHAR(128),
    txt_model    VARCHAR(64) NOT NULL DEFAULT '',
    img_model    VARCHAR(64) NOT NULL DEFAULT '',
    prompt       TEXT NOT NULL DEFAULT '',
    charter      TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE(tid, key)
);

CREATE TABLE IF NOT EXISTS agent_session (
    id         VARCHAR(36) PRIMARY KEY,
    tid        VARCHAR(64) NOT NULL DEFAULT 'default',
    agent      VARCHAR(64) NOT NULL,
    title      VARCHAR(128),
    parent     VARCHAR(36) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS agent_message (
    id              VARCHAR(36) PRIMARY KEY,
    tid             VARCHAR(64) NOT NULL DEFAULT 'default',
    sid             VARCHAR(36) NOT NULL REFERENCES agent_session(id) ON DELETE CASCADE,
    seq             INTEGER NOT NULL,
    role            VARCHAR(32) NOT NULL CHECK (role IN ('system','user','assistant','tool')),
    content         TEXT NOT NULL DEFAULT '',
    thinking        TEXT NOT NULL DEFAULT '',
    tool_calls      TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(tool_calls)),
    tool_call_id    VARCHAR(64) NOT NULL DEFAULT '',
    tool_name       VARCHAR(128) NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE(sid, seq)
);
CREATE INDEX IF NOT EXISTS idx_agent_message_session ON agent_message(sid, seq);

CREATE TABLE IF NOT EXISTS agent_experience (
    id          VARCHAR(36) PRIMARY KEY,
    tid         VARCHAR(64) NOT NULL DEFAULT 'default',
    sid         VARCHAR(36) NOT NULL,
    agent       VARCHAR(64) NOT NULL,
    summary     TEXT NOT NULL DEFAULT '',
    outcome     VARCHAR(32) NOT NULL DEFAULT 'unknown',
    tags        TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(tags)),
    weight      INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
-- idx_agent_experience_agent created in migrate.applyCanonicalSchema after renames.

CREATE TABLE IF NOT EXISTS agent_todo (
    sid        VARCHAR(36) PRIMARY KEY,
    tid        VARCHAR(64) NOT NULL DEFAULT 'default',
    items      TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(items)),
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS agent_sched (
    id          VARCHAR(36) PRIMARY KEY,
    tid         VARCHAR(64) NOT NULL DEFAULT 'default',
    name        VARCHAR(64) NOT NULL,
    agent       VARCHAR(64) NOT NULL,
    message     TEXT NOT NULL,
    schedule    TEXT NOT NULL,
    enabled     SMALLINT NOT NULL DEFAULT 1,
    last_run_at DATETIME,
    created_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE(tid, name)
);

CREATE TABLE IF NOT EXISTS mcp_server (
    id           VARCHAR(36) PRIMARY KEY,
    tid          VARCHAR(64) NOT NULL DEFAULT 'default',
    name         VARCHAR(64) NOT NULL,
    type         VARCHAR(32) NOT NULL CHECK (type IN ('stdio', 'sse', 'streamable')),
    cmd          TEXT NOT NULL DEFAULT '',
    args         TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(args)),
    url          TEXT NOT NULL DEFAULT '',
    env          TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(env)),
    enabled      SMALLINT NOT NULL DEFAULT 1,
    created_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE(tid, name)
);

CREATE TABLE IF NOT EXISTS light_app (
    id           VARCHAR(36) PRIMARY KEY,
    tid          VARCHAR(64) NOT NULL DEFAULT 'default',
    name         VARCHAR(128) NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    runtime      VARCHAR(32) NOT NULL DEFAULT 'python',
    entry_point  TEXT NOT NULL DEFAULT '',
    status       VARCHAR(32) NOT NULL DEFAULT 'stopped',
    port         INTEGER NOT NULL DEFAULT 0,
    created_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
