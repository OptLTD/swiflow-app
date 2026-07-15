// Package appdb opens and migrates the configured persistence backend.
package appdb

import (
	"context"
	"fmt"
	"strings"

	"github.com/OptLTD/swiflow/embed"
	"github.com/OptLTD/swiflow/internal/config"
	"github.com/OptLTD/swiflow/internal/migrate"
	"github.com/OptLTD/swiflow/internal/store/pg"
	"github.com/OptLTD/swiflow/internal/store/sqlite"
	"github.com/OptLTD/swiflow/internal/store/sqlstore"
)

// Open opens a Store based on cfg.DBDriver (sqlite|postgres).
func Open(cfg config.Config) (*sqlstore.Store, error) {
	driver := strings.ToLower(strings.TrimSpace(cfg.DBDriver))
	if driver == "" {
		driver = sqlstore.DialectSQLite
	}
	switch driver {
	case sqlstore.DialectSQLite, "sqlite3":
		return sqlite.Open(cfg.DBPath, cfg.EncryptionKey)
	case sqlstore.DialectPostgres, "postgresql", "pgx":
		dsn := strings.TrimSpace(cfg.DBDSN)
		if dsn == "" {
			return nil, fmt.Errorf("db_dsn is required when db_driver=%s", driver)
		}
		return pg.Open(dsn, cfg.EncryptionKey)
	default:
		return nil, fmt.Errorf("unsupported db_driver %q (want sqlite or postgres)", cfg.DBDriver)
	}
}

// ApplySchema applies the dialect-appropriate schema to an opened store.
func ApplySchema(ctx context.Context, st *sqlstore.Store) error {
	switch st.Driver() {
	case sqlstore.DialectPostgres:
		return migrate.ApplyPostgres(ctx, st.DB(), embed.SchemaPostgresSQL)
	case sqlstore.DialectSQLite:
		upgrades, err := embed.UpgradesDir()
		if err != nil {
			return fmt.Errorf("upgrades fs: %w", err)
		}
		return migrate.Apply(ctx, st.DB(), embed.SchemaSQL, upgrades)
	default:
		return fmt.Errorf("unsupported store driver %q", st.Driver())
	}
}

// MigrateAndOpen opens the configured store and optionally applies schema.
func MigrateAndOpen(ctx context.Context, cfg config.Config, autoMigrate bool) (*sqlstore.Store, error) {
	st, err := Open(cfg)
	if err != nil {
		return nil, err
	}
	if !autoMigrate {
		return st, nil
	}
	if err := ApplySchema(ctx, st); err != nil {
		_ = st.Close()
		return nil, err
	}
	return st, nil
}
