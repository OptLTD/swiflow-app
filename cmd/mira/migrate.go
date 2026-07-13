package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/OptLTD/swiflow/embed"
	"github.com/OptLTD/swiflow/internal/config"
	"github.com/OptLTD/swiflow/internal/migrate"
	"github.com/OptLTD/swiflow/internal/seed"
	"github.com/OptLTD/swiflow/internal/store/sqlite"
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
			if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
				return fmt.Errorf("create db dir: %w", err)
			}
			st, err := sqlite.Open(cfg.DBPath, cfg.EncryptionKey)
			if err != nil {
				return err
			}
			defer st.Close()
			upgrades, err := embed.UpgradesDir()
			if err != nil {
				return fmt.Errorf("upgrades fs: %w", err)
			}
			if err := migrate.Apply(context.Background(), st.DB(), embed.SchemaSQL, upgrades); err != nil {
				return fmt.Errorf("migrate: %w", err)
			}
			if err := seed.EnsureDefaults(context.Background(), st); err != nil {
				return fmt.Errorf("seed: %w", err)
			}
			slog.Info("schema applied", "db", cfg.DBPath)
			return nil
		},
	}
}

func loadConfig() (config.Config, error) {
	path := cfgFile
	if path == "" {
		path = os.Getenv("MIRA_CONFIG")
	}
	if path == "" {
		path = "config.json"
	}
	return config.Load(path)
}
