package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/OptLTD/swiflow/internal/appdb"
	"github.com/OptLTD/swiflow/internal/config"
	"github.com/OptLTD/swiflow/internal/store/sqlstore"
)

func migrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Initialize database schema and apply upgrades",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if cfg.DBDriver == "" || cfg.DBDriver == sqlstore.DialectSQLite || cfg.DBDriver == "sqlite3" {
				if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
					return fmt.Errorf("create db dir: %w", err)
				}
			}
			st, err := appdb.MigrateAndOpen(context.Background(), cfg, true)
			if err != nil {
				return err
			}
			defer st.Close()
			if err := appdb.EnsureDefaults(context.Background(), st); err != nil {
				return fmt.Errorf("seed: %w", err)
			}
			if st.Driver() == sqlstore.DialectPostgres {
				slog.Info("schema applied", "driver", st.Driver(), "dsn", redactDSN(cfg.DBDSN))
			} else {
				slog.Info("schema applied", "driver", st.Driver(), "db", cfg.DBPath)
			}
			return nil
		},
	}
}

func loadConfig() (config.Config, error) {
	path := cfgFile
	if path == "" {
		path = os.Getenv("SWIFLOW_CONFIG")
	}
	if path == "" {
		path = "config.json"
	}
	return config.Load(path)
}

func redactDSN(dsn string) string {
	// Avoid logging passwords; keep scheme/host only when possible.
	if dsn == "" {
		return ""
	}
	at := -1
	for i := 0; i < len(dsn); i++ {
		if dsn[i] == '@' {
			at = i
			break
		}
	}
	if at < 0 {
		return "(set)"
	}
	scheme := dsn
	for i := 0; i < len(dsn)-2; i++ {
		if dsn[i] == ':' && dsn[i+1] == '/' && dsn[i+2] == '/' {
			scheme = dsn[:i+3]
			break
		}
	}
	return scheme + "***" + dsn[at:]
}
