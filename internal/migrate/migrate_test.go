package migrate_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	_ "modernc.org/sqlite"

	"github.com/OptLTD/swiflow/embed"
	"github.com/OptLTD/swiflow/internal/migrate"
)

func TestApplyIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	upgrades, err := embed.UpgradesDir()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := migrate.Apply(ctx, db, embed.SchemaSQL, upgrades); err != nil {
			t.Fatalf("apply %d: %v", i, err)
		}
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n < 6 {
		t.Fatalf("expected tables, got %d", n)
	}
}

func TestApplyUpgradesInOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	fsys := fstest.MapFS{
		"0001_a.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE IF NOT EXISTS upgrade_a (id TEXT PRIMARY KEY);`)},
		"0002_b.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE IF NOT EXISTS upgrade_b (id TEXT PRIMARY KEY);`)},
	}
	if err := migrate.Apply(ctx, db, embed.SchemaSQL, fsys); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Apply(ctx, db, embed.SchemaSQL, fsys); err != nil {
		t.Fatal(err)
	}
	var versions []string
	rows, err := db.Query(`SELECT version FROM sys_migration ORDER BY version`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		rows.Scan(&v)
		versions = append(versions, v)
	}
	if strings.Join(versions, ",") != "0001_a.sql,0002_b.sql" {
		t.Fatalf("versions: %v", versions)
	}
}

func TestCanonicalRenameFromLegacyColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()

	legacy := `
CREATE TABLE sys_migration (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT (datetime('now')));
CREATE TABLE llm_provider (
    id TEXT PRIMARY KEY, tid TEXT NOT NULL DEFAULT 'default',
    name TEXT NOT NULL UNIQUE, display_name TEXT,
    api_base TEXT NOT NULL, api_key_enc BLOB NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE agent_config (
    id TEXT PRIMARY KEY, tid TEXT NOT NULL DEFAULT 'default',
    key TEXT NOT NULL UNIQUE, display_name TEXT,
    provider TEXT NOT NULL, model TEXT NOT NULL,
    system_extra TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE agent_session (
    id TEXT PRIMARY KEY, tid TEXT NOT NULL DEFAULT 'default',
    agent_key TEXT NOT NULL, title TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE agent_experience (
    id TEXT PRIMARY KEY, tid TEXT NOT NULL DEFAULT 'default',
    sid TEXT NOT NULL, agent_key TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '', outcome TEXT NOT NULL DEFAULT 'unknown',
    tags TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE agent_sched (
    id TEXT PRIMARY KEY, tid TEXT NOT NULL DEFAULT 'default',
    name TEXT NOT NULL UNIQUE, agent_key TEXT NOT NULL,
    message TEXT NOT NULL, schedule TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1, last_run_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE mcp_server (
    id TEXT PRIMARY KEY, tid TEXT NOT NULL DEFAULT 'default',
    name TEXT NOT NULL UNIQUE, display_name TEXT,
    transport TEXT NOT NULL, command TEXT NOT NULL DEFAULT '',
    args_json TEXT NOT NULL DEFAULT '[]', url TEXT NOT NULL DEFAULT '',
    env_json TEXT NOT NULL DEFAULT '{}', enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO llm_provider (id, name, display_name, api_base, api_key_enc, enabled)
VALUES ('p1', 'openai', 'OpenAI', 'https://api.openai.com/v1', x'00', 1);
INSERT INTO agent_config (id, key, display_name, provider, model)
VALUES ('a1', 'default', 'Default', 'openai', 'gpt-4o-mini');
`
	if _, err := db.ExecContext(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	upgrades, err := embed.UpgradesDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate.Apply(ctx, db, embed.SchemaSQL, upgrades); err != nil {
		t.Fatal(err)
	}

	var display, model, txt string
	if err := db.QueryRow(`SELECT display, model FROM llm_provider WHERE name='openai'`).Scan(&display, &model); err != nil {
		t.Fatal(err)
	}
	if display != "OpenAI" || model != "gpt-4o-mini" {
		t.Fatalf("provider: display=%q model=%q", display, model)
	}
	if err := db.QueryRow(`SELECT display, txt_model FROM agent_config WHERE key='default'`).Scan(&display, &txt); err != nil {
		t.Fatal(err)
	}
	if display != "Default" || txt != "openai" {
		t.Fatalf("agent: display=%q txt_model=%q", display, txt)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('agent_config') WHERE name IN ('provider','model')`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("legacy provider/model columns still present: %d", n)
	}
}
