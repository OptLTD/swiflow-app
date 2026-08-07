package server

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/OptLTD/swiflow/internal/observe"
	"github.com/OptLTD/swiflow/internal/tenant"
)

// getPaths returns absolute storage paths used by the runtime.
func (s *Server) getPaths(w http.ResponseWriter, r *http.Request) {
	dataDir := s.cfg.DataDir()
	logFile := observe.LogFileAbsPath()
	if logFile == "" {
		logFile = filepath.Join(dataDir, observe.RelLogFile)
	}
	ws := s.cfg.RootsForTenant(tenant.ID(r.Context())).Workspace
	writeJSON(w, http.StatusOK, map[string]string{
		"data_dir":      dataDir,
		"workspace_dir": ws,
		"log_file":      logFile,
	})
}

// openDataDir opens the persistent data directory in the OS file manager.
func (s *Server) openDataDir(w http.ResponseWriter, r *http.Request) {
	dir := s.cfg.DataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrInternalError, err.Error())
		return
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, ErrInternalError, err.Error())
		return
	}
	if err := openPathInOS(abs); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrInternalError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "path": abs})
}

// openLogFile opens (or reveals) the application log file in the OS.
func (s *Server) openLogFile(w http.ResponseWriter, r *http.Request) {
	path := observe.LogFileAbsPath()
	if path == "" {
		path = filepath.Join(s.cfg.DataDir(), observe.RelLogFile)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, ErrInternalError, err.Error())
		return
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrInternalError, err.Error())
		return
	}
	// Ensure the file exists so the OS can open/reveal it.
	if _, err := os.Stat(abs); err != nil {
		f, createErr := os.OpenFile(abs, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if createErr != nil {
			writeErr(w, http.StatusInternalServerError, ErrInternalError, createErr.Error())
			return
		}
		_ = f.Close()
	}
	if err := revealPathInOS(abs); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrInternalError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "path": abs})
}
