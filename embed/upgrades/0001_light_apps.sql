-- Light Apps table
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
