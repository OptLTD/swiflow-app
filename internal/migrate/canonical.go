package migrate

import (
	"context"
	"database/sql"
	"fmt"
)

// applyCanonicalSchema renames legacy columns and reshapes agent_config /
// llm_provider for existing databases. Idempotent: safe on fresh installs
// where schema.sql already has the canonical shape.
func applyCanonicalSchema(ctx context.Context, db *sql.DB) error {
	renames := []struct {
		table, from, to string
	}{
		{"llm_provider", "display_name", "display"},
		{"llm_provider", "api_key_enc", "api_key"},
		{"agent_config", "display_name", "display"},
		{"agent_config", "system_extra", "sys_prompt"},
		{"mcp_server", "transport", "type"},
		{"mcp_server", "command", "cmd"},
		{"mcp_server", "args_json", "args"},
		{"mcp_server", "env_json", "env"},
		{"agent_session", "agent_key", "agent"},
		{"agent_experience", "agent_key", "agent"},
		{"agent_sched", "agent_key", "agent"},
		{"agent_todo", "items_json", "items"},
		{"agent_message", "tool_calls_json", "tool_calls"},
	}
	for _, r := range renames {
		if err := renameColumnIfExists(ctx, db, r.table, r.from, r.to); err != nil {
			return err
		}
	}

	// mcp_server.display / display_name are unused UI labels — drop if present.
	if err := dropColumnIfExists(ctx, db, "mcp_server", "display_name"); err != nil {
		return err
	}
	if err := dropColumnIfExists(ctx, db, "mcp_server", "display"); err != nil {
		return err
	}

	if err := addColumnIfMissing(ctx, db, "llm_provider", "model", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, "agent_config", "txt_model", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, "agent_config", "img_model", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, "agent_session", "parent", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, "sys_tenant", "password_hash", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}

	// Migrate agent_config.provider/model → txt_model + llm_provider.model.
	hasProvider, err := columnExists(ctx, db, "agent_config", "provider")
	if err != nil {
		return err
	}
	if hasProvider {
		if _, err := db.ExecContext(ctx, `
			UPDATE agent_config SET txt_model = provider
			WHERE (txt_model = '' OR txt_model IS NULL) AND provider IS NOT NULL AND provider != ''
		`); err != nil {
			return fmt.Errorf("backfill txt_model: %w", err)
		}
		if _, err := db.ExecContext(ctx, `
			UPDATE llm_provider
			SET model = COALESCE((
				SELECT a.model FROM agent_config a
				WHERE a.provider = llm_provider.name AND a.model != ''
				LIMIT 1
			), model)
			WHERE model = '' OR model IS NULL
		`); err != nil {
			return fmt.Errorf("backfill provider model: %w", err)
		}
		if err := dropColumnIfExists(ctx, db, "agent_config", "provider"); err != nil {
			return err
		}
		if err := dropColumnIfExists(ctx, db, "agent_config", "model"); err != nil {
			return err
		}
	}

	// Refresh experience index name/columns if needed.
	if _, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS idx_agent_experience_agent`); err != nil {
		return err
	}
	if hasAgent, err := columnExists(ctx, db, "agent_experience", "agent"); err != nil {
		return err
	} else if hasAgent {
		if _, err := db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_agent_experience_agent
			ON agent_experience(agent, created_at)
		`); err != nil {
			return err
		}
	}
	return nil
}

func renameColumnIfExists(ctx context.Context, db *sql.DB, table, from, to string) error {
	hasFrom, err := columnExists(ctx, db, table, from)
	if err != nil {
		return err
	}
	if !hasFrom {
		return nil
	}
	hasTo, err := columnExists(ctx, db, table, to)
	if err != nil {
		return err
	}
	if hasTo {
		return nil
	}
	_, err = db.ExecContext(ctx, fmt.Sprintf(
		`ALTER TABLE %s RENAME COLUMN %s TO %s`, table, from, to,
	))
	if err != nil {
		return fmt.Errorf("rename %s.%s→%s: %w", table, from, to, err)
	}
	return nil
}

func addColumnIfMissing(ctx context.Context, db *sql.DB, table, col, decl string) error {
	ok, err := columnExists(ctx, db, table, col)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	_, err = db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, col, decl))
	if err != nil {
		return fmt.Errorf("add %s.%s: %w", table, col, err)
	}
	return nil
}

func dropColumnIfExists(ctx context.Context, db *sql.DB, table, col string) error {
	ok, err := columnExists(ctx, db, table, col)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	_, err = db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s`, table, col))
	if err != nil {
		return fmt.Errorf("drop %s.%s: %w", table, col, err)
	}
	return nil
}

func columnExists(ctx context.Context, db *sql.DB, table, col string) (bool, error) {
	// Table may not exist yet on partial init.
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table,
	).Scan(&n)
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == col {
			return true, nil
		}
	}
	return false, rows.Err()
}
