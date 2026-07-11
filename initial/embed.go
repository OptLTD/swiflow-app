// Package initial embeds the database schema and incremental upgrade scripts.
package initial

import (
	"embed"
	"io/fs"
)

// SchemaSQL is the Phase 1 SQLite schema (idempotent CREATE IF NOT EXISTS).
//
//go:embed schema.sql
var SchemaSQL string

// upgrades embeds incremental SQL files under upgrades/ (NNNN_*.sql).
//
//go:embed all:upgrades
var upgrades embed.FS

// UpgradesDir returns the embedded upgrades/ directory as an fs.FS.
func UpgradesDir() (fs.FS, error) {
	return fs.Sub(upgrades, "upgrades")
}
