package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDatabaseDSN(t *testing.T) {
	cases := []struct {
		in             string
		driver, sqlite string
		pg             string
	}{
		{"sqlite://./data/swiflow.db", "sqlite", "./data/swiflow.db", ""},
		{"sqlite:./data/swiflow.db", "sqlite", "./data/swiflow.db", ""},
		{"postgres://u:p@localhost:5432/db?sslmode=disable", "postgres", "", "postgres://u:p@localhost:5432/db?sslmode=disable"},
		{"postgresql://localhost/db", "postgres", "", "postgresql://localhost/db"},
	}
	for _, c := range cases {
		driver, path, pg, err := ParseDatabaseDSN(c.in)
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if driver != c.driver || path != c.sqlite || pg != c.pg {
			t.Fatalf("%q → (%q,%q,%q) want (%q,%q,%q)", c.in, driver, path, pg, c.driver, c.sqlite, c.pg)
		}
	}
	if _, _, _, err := ParseDatabaseDSN("./data/swiflow.db"); err == nil {
		t.Fatal("bare path must be rejected")
	}
}

func TestLoadHostAddressAndDatabaseDSN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw := `{
	  "host_address": "0.0.0.0:9000",
	  "database_dsn": "sqlite://./tmp/app.db",
	  "workspace_dir": "./ws"
	}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "0.0.0.0" || cfg.Port != 9000 {
		t.Fatalf("listen=%s:%d", cfg.Host, cfg.Port)
	}
	if cfg.Addr() != "0.0.0.0:9000" {
		t.Fatalf("Addr=%q", cfg.Addr())
	}
	if cfg.DBDriver != "sqlite" || cfg.DBPath != "./tmp/app.db" {
		t.Fatalf("db=%s %s", cfg.DBDriver, cfg.DBPath)
	}
}

func TestLoadPostgresDatabaseDSN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw := `{
	  "host_address": "127.0.0.1:8000",
	  "database_dsn": "postgres://postgres:postgres@127.0.0.1:5432/swiflow?sslmode=disable"
	}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBDriver != "postgres" {
		t.Fatalf("driver=%s", cfg.DBDriver)
	}
	if cfg.DBDSN != "postgres://postgres:postgres@127.0.0.1:5432/swiflow?sslmode=disable" {
		t.Fatalf("dsn=%s", cfg.DBDSN)
	}
}

func TestPartialContextKeepsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw := `{
	  "host_address": "127.0.0.1:8000",
	  "database_dsn": "sqlite://./data/x.db",
	  "context": { "disable_thinking": false }
	}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Context.DisableThinking {
		t.Fatal("disable_thinking should be false from file")
	}
	if cfg.Context.MaxHistoryMsgs != 100 || cfg.Context.MaxContextChars != 120_000 {
		t.Fatalf("defaults lost: %+v", cfg.Context)
	}
}
