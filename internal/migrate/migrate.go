// Package migrate applies the embedded schema and incremental upgrades.
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Apply runs SQLite schemaSQL (idempotent CREATE IF NOT EXISTS), applies
// canonical column renames for legacy DBs, then any unapplied upgrades.
func Apply(ctx context.Context, db *sql.DB, schemaSQL string, upgradesFS fs.FS) error {
	if err := execInTx(ctx, db, schemaSQL); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	if err := applyCanonicalSchema(ctx, db); err != nil {
		return fmt.Errorf("canonical schema: %w", err)
	}
	if upgradesFS == nil {
		return nil
	}
	if err := applyUpgrades(ctx, db, upgradesFS, false); err != nil {
		return err
	}
	return nil
}

// ApplyPostgres applies the Postgres schema, then reconciles columns that were
// added to existing tables after their initial creation (CREATE TABLE IF NOT
// EXISTS never alters an existing table). Postgres supports ADD COLUMN IF NOT
// EXISTS, so each reconcile statement is idempotent.
func ApplyPostgres(ctx context.Context, db *sql.DB, schemaSQL string) error {
	if err := execInTx(ctx, db, schemaSQL); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	reconcile := []string{
		`ALTER TABLE agent_session ADD COLUMN IF NOT EXISTS parent VARCHAR(36) NOT NULL DEFAULT ''`,
	}
	for _, stmt := range reconcile {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("reconcile: %w", err)
		}
	}
	return nil
}

func applyUpgrades(ctx context.Context, db *sql.DB, fsys fs.FS, postgres bool) error {
	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return fmt.Errorf("read upgrades: %w", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		if applied[name] {
			continue
		}
		content, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if err := execInTx(ctx, db, string(content)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		q := `INSERT INTO sys_migration(version) VALUES (?)`
		if postgres {
			q = sqlx.Rebind(sqlx.DOLLAR, q)
		}
		if _, err := db.ExecContext(ctx, q, name); err != nil {
			return fmt.Errorf("record %s: %w", name, err)
		}
	}
	return nil
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM sys_migration`)
	if err != nil {
		return nil, fmt.Errorf("read sys_migration: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func execInTx(ctx context.Context, db *sql.DB, sqlText string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for _, stmt := range splitStatements(sqlText) {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func splitStatements(sqlText string) []string {
	var out []string
	var cur strings.Builder
	for _, line := range strings.Split(sqlText, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "--") || trim == "" {
			continue
		}
		cur.WriteString(line)
		cur.WriteString("\n")
		if strings.HasSuffix(trim, ";") {
			s := strings.TrimSpace(cur.String())
			if s != "" {
				out = append(out, s)
			}
			cur.Reset()
		}
	}
	if s := strings.TrimSpace(cur.String()); s != "" {
		out = append(out, s)
	}
	return out
}
