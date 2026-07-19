package server

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OptLTD/swiflow/library/support"
	"github.com/OptLTD/swiflow/library/workspace"
)

const maxWorkspaceUpload = workspace.MaxFileSize

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
	full, err := support.SandboxPath(s.cfg.WorkspaceDir, dir)
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
	full, err := support.SandboxPath(s.cfg.WorkspaceDir, path)
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

func (s *Server) downloadFile(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Path string `json:"path"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	path := strings.TrimSpace(in.Path)
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path required"})
		return
	}
	payload, err := workspace.ReadBinaryFile(s.cfg.WorkspaceDir, path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

type uploadedFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func (s *Server) uploadWorkspace(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxWorkspaceUpload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}

	dir := strings.TrimSpace(r.FormValue("path"))
	if dir == "" {
		dir = "."
	}
	destDir, err := support.SandboxPath(s.cfg.WorkspaceDir, dir)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mkdir failed"})
		return
	}

	headers := r.MultipartForm.File["files"]
	if len(headers) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no files"})
		return
	}

	uploaded := make([]uploadedFile, 0, len(headers))
	for _, fh := range headers {
		item, err := saveUploadedFile(s.cfg.WorkspaceDir, dir, fh)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		uploaded = append(uploaded, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"path":     dir,
		"uploaded": uploaded,
	})
}

func saveUploadedFile(workspaceDir, dir string, fh *multipart.FileHeader) (uploadedFile, error) {
	name, err := workspace.SafeUploadName(fh.Filename)
	if err != nil {
		return uploadedFile{}, err
	}
	if fh.Size > maxWorkspaceUpload {
		return uploadedFile{}, fmt.Errorf("file too large: %s", name)
	}

	rel := name
	if dir != "." {
		rel = filepath.ToSlash(filepath.Join(dir, name))
	}
	full, err := support.SandboxPath(workspaceDir, rel)
	if err != nil {
		return uploadedFile{}, err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return uploadedFile{}, fmt.Errorf("mkdir failed: %w", err)
	}

	src, err := fh.Open()
	if err != nil {
		return uploadedFile{}, fmt.Errorf("open upload: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(full, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return uploadedFile{}, fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	n, err := io.Copy(dst, io.LimitReader(src, maxWorkspaceUpload+1))
	if err != nil {
		return uploadedFile{}, fmt.Errorf("write file: %w", err)
	}
	if n > maxWorkspaceUpload {
		_ = os.Remove(full)
		return uploadedFile{}, fmt.Errorf("file too large: %s", name)
	}

	return uploadedFile{Name: name, Path: rel, Size: n}, nil
}
