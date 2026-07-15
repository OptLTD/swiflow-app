-- Swiflow — PostgreSQL schema (idempotent CREATE IF NOT EXISTS).
-- Applied at startup via migrate.ApplyPostgres.

CREATE TABLE IF NOT EXISTS sys_migration (
    version    VARCHAR(64) PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sys_tenant (
    id         VARCHAR(36) PRIMARY KEY,
    name       VARCHAR(64) NOT NULL UNIQUE,
    enabled    SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sys_user (
    id         VARCHAR(36) PRIMARY KEY,
    tid        VARCHAR(64) NOT NULL DEFAULT 'default',
    username   VARCHAR(64) NOT NULL UNIQUE,
    enabled    SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sys_settings (
    key        VARCHAR(128) PRIMARY KEY,
    value      TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS llm_provider (
    id           VARCHAR(36) PRIMARY KEY,
    tid          VARCHAR(64) NOT NULL DEFAULT 'default',
    name         VARCHAR(64) NOT NULL UNIQUE,
    display      VARCHAR(128),
    api_base     TEXT NOT NULL,
    api_key      BYTEA NOT NULL,
    model        VARCHAR(128) NOT NULL DEFAULT '',
    enabled      SMALLINT NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agent_config (
    id           VARCHAR(36) PRIMARY KEY,
    tid          VARCHAR(64) NOT NULL DEFAULT 'default',
    key          VARCHAR(64) NOT NULL UNIQUE,
    display      VARCHAR(128),
    txt_model    VARCHAR(64) NOT NULL DEFAULT '',
    img_model    VARCHAR(64) NOT NULL DEFAULT '',
    sys_prompt   TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agent_session (
    id         VARCHAR(36) PRIMARY KEY,
    tid        VARCHAR(64) NOT NULL DEFAULT 'default',
    agent      VARCHAR(64) NOT NULL,
    title      VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agent_message (
    id              VARCHAR(36) PRIMARY KEY,
    tid             VARCHAR(64) NOT NULL DEFAULT 'default',
    sid             VARCHAR(36) NOT NULL REFERENCES agent_session(id) ON DELETE CASCADE,
    seq             INTEGER NOT NULL,
    role            VARCHAR(32) NOT NULL CHECK (role IN ('system','user','assistant','tool')),
    content         TEXT NOT NULL DEFAULT '',
    thinking        TEXT NOT NULL DEFAULT '',
    tool_calls      JSONB NOT NULL DEFAULT '[]'::jsonb,
    tool_call_id    VARCHAR(64) NOT NULL DEFAULT '',
    tool_name       VARCHAR(128) NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
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
    tags        JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_agent_experience_agent ON agent_experience(agent, created_at);

CREATE TABLE IF NOT EXISTS agent_todo (
    sid        VARCHAR(36) PRIMARY KEY,
    tid        VARCHAR(64) NOT NULL DEFAULT 'default',
    items      JSONB NOT NULL DEFAULT '[]'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agent_sched (
    id          VARCHAR(36) PRIMARY KEY,
    tid         VARCHAR(64) NOT NULL DEFAULT 'default',
    name        VARCHAR(64) NOT NULL UNIQUE,
    agent       VARCHAR(64) NOT NULL,
    message     TEXT NOT NULL,
    schedule    TEXT NOT NULL,
    enabled     SMALLINT NOT NULL DEFAULT 1,
    last_run_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS mcp_server (
    id           VARCHAR(36) PRIMARY KEY,
    tid          VARCHAR(64) NOT NULL DEFAULT 'default',
    name         VARCHAR(64) NOT NULL UNIQUE,
    "type"       VARCHAR(32) NOT NULL CHECK ("type" IN ('stdio', 'sse', 'streamable')),
    cmd          TEXT NOT NULL DEFAULT '',
    args         JSONB NOT NULL DEFAULT '[]'::jsonb,
    url          TEXT NOT NULL DEFAULT '',
    env          JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled      SMALLINT NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
