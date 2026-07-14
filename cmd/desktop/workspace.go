package main

import (
	"github.com/OptLTD/swiflow/internal/config"
	"github.com/OptLTD/swiflow/internal/workspace"
)

// Workspace exposes workspace file reads to the desktop frontend via Wails IPC.
// This avoids WKWebView blocking binary fetches under wails://.
type Workspace struct {
	cfg config.Config
}

func (s *Workspace) DownloadFile(path string) (*workspace.BinaryPayload, error) {
	return workspace.ReadBinaryFile(s.cfg.WorkspaceDir, path)
}
