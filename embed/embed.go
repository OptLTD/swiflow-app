// Package embed holds assets compiled into the Swiflow binary: database schema,
// incremental upgrades, built-in skills, desktop frontend, and app icons.
package embed

import (
	"embed"
	"io/fs"
)

// SchemaSQL is the SQLite schema (idempotent CREATE IF NOT EXISTS).
//
//go:embed schema.sql
var SchemaSQL string

// SchemaPostgresSQL is the PostgreSQL schema (idempotent CREATE IF NOT EXISTS).
//
//go:embed schema.pg.sql
var SchemaPostgresSQL string

// upgrades embeds incremental SQL files under upgrades/ (NNNN_*.sql).
// SQLite-oriented; Postgres greenfield installs use SchemaPostgresSQL only for now.
//
//go:embed all:upgrades
var upgrades embed.FS

// InitSkillsFS holds embedded built-in skills (init-skills/<slug>/SKILL.md).
//
//go:embed all:init-skills
var InitSkillsFS embed.FS

// FrontendDist holds the built Vue UI for the wails3 desktop app.
// Vite writes webui build output directly into embed/frontend/.
//
//go:embed all:frontend
var FrontendDist embed.FS

// AppIconPNG is the desktop application icon (Dock / taskbar).
//
//go:embed icons/appicon.png
var AppIconPNG []byte

// UpgradesDir returns the embedded upgrades/ directory as an fs.FS.
func UpgradesDir() (fs.FS, error) {
	return fs.Sub(upgrades, "upgrades")
}

// GetFrontendDist returns the embedded frontend/ directory as an fs.FS.
func GetFrontendDist() (fs.FS, error) {
	return fs.Sub(FrontendDist, "frontend")
}
