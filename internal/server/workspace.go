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

	"github.com/OptLTD/swiflow/internal/tenant"
	"github.com/OptLTD/swiflow/library/support"
	"github.com/OptLTD/swiflow/library/workspace"
)

func (s *Server) tenantWorkspace(r *http.Request) string {
	return s.cfg.RootsForTenant(tenant.ID(r.Context())).Workspace
}

const maxWorkspaceUpload = workspace.MaxFileSize

func writeWorkspaceOpErr(w http.ResponseWriter, status int, err error) {
	msg := err.Error()
	const prefix = "file too large: "
	if strings.HasPrefix(msg, prefix) {
		writeErr(w, status, ErrFileTooLarge, strings.TrimSpace(msg[len(prefix):]))
		return
	}
	writeErr(w, status, ErrInternalError, msg)
}

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
	full, err := support.SandboxPath(s.tenantWorkspace(r), dir)
	if err != nil {
		writeErr(w, http.StatusBadRequest, ErrInternalError, err.Error())
		return
	}
	infos, err := os.ReadDir(full)
	if err != nil {
		writeErr(w, http.StatusBadRequest, ErrInternalError, err.Error())
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
		writeErr(w, http.StatusBadRequest, ErrPathRequired)
		return
	}
	full, err := support.SandboxPath(s.tenantWorkspace(r), path)
	if err != nil {
		writeErr(w, http.StatusBadRequest, ErrInternalError, err.Error())
		return
	}
	info, err := os.Stat(full)
	if err != nil {
		writeErr(w, http.StatusBadRequest, ErrInternalError, err.Error())
		return
	}
	if info.IsDir() {
		writeErr(w, http.StatusBadRequest, ErrPathIsDirectory)
		return
	}
	data, err := os.ReadFile(full)
	if err != nil {
		writeErr(w, http.StatusBadRequest, ErrInternalError, err.Error())
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
		writeErr(w, http.StatusBadRequest, ErrPathRequired)
		return
	}
	payload, err := workspace.ReadBinaryFile(s.tenantWorkspace(r), path)
	if err != nil {
		writeErr(w, http.StatusBadRequest, ErrInternalError, err.Error())
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
		writeErr(w, http.StatusBadRequest, ErrInvalidMultipart)
		return
	}

	// Client "path" is ignored: uploads always land in the immutable uploads/ inbox.
	headers := r.MultipartForm.File["files"]
	if len(headers) == 0 {
		writeErr(w, http.StatusBadRequest, ErrNoFiles)
		return
	}

	uploaded := make([]uploadedFile, 0, len(headers))
	ws := s.tenantWorkspace(r)
	for _, fh := range headers {
		item, err := saveUploadedFile(ws, fh)
		if err != nil {
			writeWorkspaceOpErr(w, http.StatusBadRequest, err)
			return
		}
		uploaded = append(uploaded, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"path":     workspace.UploadsRoot,
		"uploaded": uploaded,
	})
}

func saveUploadedFile(workspaceDir string, fh *multipart.FileHeader) (uploadedFile, error) {
	origName, err := workspace.SafeUploadName(fh.Filename)
	if err != nil {
		return uploadedFile{}, err
	}
	rel, err := workspace.AllocUploadRel(origName)
	if err != nil {
		return uploadedFile{}, err
	}
	if fh.Size > maxWorkspaceUpload {
		return uploadedFile{}, fmt.Errorf("file too large: %s", origName)
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

	dst, err := os.OpenFile(full, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return uploadedFile{}, fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	n, err := io.Copy(dst, io.LimitReader(src, maxWorkspaceUpload+1))
	if err != nil {
		_ = os.Remove(full)
		return uploadedFile{}, fmt.Errorf("write file: %w", err)
	}
	if n > maxWorkspaceUpload {
		_ = os.Remove(full)
		return uploadedFile{}, fmt.Errorf("file too large: %s", origName)
	}

	return uploadedFile{Name: origName, Path: rel, Size: n}, nil
}
