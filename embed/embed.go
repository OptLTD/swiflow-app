// Package embed holds assets compiled into the Mira binary: database schema,
// incremental upgrades, built-in skills, and the desktop frontend.
package embed

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

// InitSkillsFS holds embedded built-in skills (init-skills/<slug>/SKILL.md).
//
//go:embed all:init-skills
var InitSkillsFS embed.FS

// DesktopFrontend holds the built Vue UI for the wails3 desktop app.
// Vite writes webui build output directly into embed/frontend/.
//
//go:embed all:frontend
var DesktopFrontend embed.FS

// UpgradesDir returns the embedded upgrades/ directory as an fs.FS.
func UpgradesDir() (fs.FS, error) {
	return fs.Sub(upgrades, "upgrades")
}

// DesktopFrontendDist returns the embedded frontend/ directory as an fs.FS.
func DesktopFrontendDist() (fs.FS, error) {
	return fs.Sub(DesktopFrontend, "frontend")
}
