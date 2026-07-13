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
	rows, err := db.Query(`SELECT version FROM schema_migrations ORDER BY version`)
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
