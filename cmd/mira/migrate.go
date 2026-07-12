package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"mira/initial"
	"mira/internal/config"
	"mira/internal/migrate"
	"mira/internal/seed"
	"mira/internal/store/sqlite"
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
			upgrades, err := initial.UpgradesDir()
			if err != nil {
				return fmt.Errorf("upgrades fs: %w", err)
			}
			if err := migrate.Apply(context.Background(), st.DB(), initial.SchemaSQL, upgrades); err != nil {
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
