-- Phase 2: MCP server configurations.

CREATE TABLE IF NOT EXISTS mcp_servers (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL UNIQUE,
    display_name TEXT,
    transport    TEXT NOT NULL CHECK (transport IN ('stdio', 'sse', 'streamable')),
    command      TEXT NOT NULL DEFAULT '',
    args_json    TEXT NOT NULL DEFAULT '[]',
    url          TEXT NOT NULL DEFAULT '',
    env_json     TEXT NOT NULL DEFAULT '{}',
    enabled      INTEGER NOT NULL DEFAULT 1,
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now'))
);
