package server

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OptLTD/swiflow/internal/secure"
)

type workspaceEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size,omitempty"`
	ModTime string `json:"mod_time,omitempty"`
}

func (s *Server) listWorkspace(w http.ResponseWriter, r *http.Request) {
	dir := strings.TrimSpace(r.URL.Query().Get("path"))
	if dir == "" {
		dir = "."
	}
	full, err := secure.SandboxPath(s.cfg.WorkspaceDir, dir)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	infos, err := os.ReadDir(full)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	entries := make([]workspaceEntry, 0, len(infos)+1)
	if dir != "." {
		parent := filepath.Dir(dir)
		if parent == "." {
			parent = "."
		}
		entries = append(entries, workspaceEntry{
			Name:  "..",
			Path:  parent,
			IsDir: true,
		})
	}

	for _, info := range infos {
		name := info.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		entryPath := filepath.ToSlash(filepath.Join(dir, name))
		if dir == "." {
			entryPath = name
		}
		e := workspaceEntry{
			Name:  name,
			Path:  entryPath,
			IsDir: info.IsDir(),
		}
		if fi, err := info.Info(); err == nil {
			if !fi.IsDir() {
				e.Size = fi.Size()
			}
			e.ModTime = fi.ModTime().UTC().Format(time.RFC3339)
		}
		entries = append(entries, e)
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Name == ".." {
			return true
		}
		if entries[j].Name == ".." {
			return false
		}
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"path":    dir,
		"entries": entries,
	})
}

func (s *Server) readWorkspaceFile(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path required"})
		return
	}
	full, err := secure.SandboxPath(s.cfg.WorkspaceDir, path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	info, err := os.Stat(full)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if info.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path is a directory"})
		return
	}
	data, err := os.ReadFile(full)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	const max = 512 * 1024
	truncated := false
	if len(data) > max {
		data = data[:max]
		truncated = true
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":      path,
		"content":   string(data),
		"truncated": truncated,
	})
}
