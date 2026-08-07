// Package workspace provides helpers for importing files into the agent workspace.
package workspace

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OptLTD/swiflow/library/support"
)

const (
	MaxFileSize = 64 << 20 // 64 MiB
	// UploadsRoot is the immutable inbox for user uploads. Chat history cites
	// paths under this directory; agents should copy (not move) when organizing.
	UploadsRoot = "uploads"
)

// ImportedFile describes a file written into the workspace.
type ImportedFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// AllocUploadRel returns a unique workspace-relative path under uploads/:
// uploads/YYYYMMDD/<id12>_<safeName>.
func AllocUploadRel(originalName string) (string, error) {
	name, err := SafeUploadName(originalName)
	if err != nil {
		return "", err
	}
	id := strings.ReplaceAll(support.NewID(), "-", "")
	if len(id) > 12 {
		// Prefer the trailing bits (less timestamp-heavy on UUIDv7).
		id = id[len(id)-12:]
	}
	day := time.Now().UTC().Format("20060102")
	return filepath.ToSlash(filepath.Join(UploadsRoot, day, id+"_"+name)), nil
}

// ImportFiles copies files from absolute source paths into the immutable
// uploads/ inbox (paths are unique; originals stay put for history refs).
func ImportFiles(workspaceDir string, sourcePaths []string) ([]ImportedFile, error) {
	if len(sourcePaths) == 0 {
		return nil, fmt.Errorf("no files")
	}

	uploaded := make([]ImportedFile, 0, len(sourcePaths))
	for _, src := range sourcePaths {
		item, err := importOne(workspaceDir, src)
		if err != nil {
			return uploaded, err
		}
		uploaded = append(uploaded, item)
	}
	return uploaded, nil
}

func importOne(workspaceDir, src string) (ImportedFile, error) {
	origName, err := SafeUploadName(filepath.Base(src))
	if err != nil {
		return ImportedFile{}, err
	}
	rel, err := AllocUploadRel(origName)
	if err != nil {
		return ImportedFile{}, err
	}

	dest, err := support.SandboxPath(workspaceDir, rel)
	if err != nil {
		return ImportedFile{}, err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return ImportedFile{}, fmt.Errorf("mkdir failed: %w", err)
	}

	info, err := os.Stat(src)
	if err != nil {
		return ImportedFile{}, fmt.Errorf("stat source: %w", err)
	}
	if info.IsDir() {
		return ImportedFile{}, fmt.Errorf("directories not supported: %s", origName)
	}
	if info.Size() > MaxFileSize {
		return ImportedFile{}, fmt.Errorf("file too large: %s", origName)
	}

	in, err := os.Open(src)
	if err != nil {
		return ImportedFile{}, fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return ImportedFile{}, fmt.Errorf("create dest: %w", err)
	}
	defer out.Close()

	n, err := io.Copy(out, io.LimitReader(in, MaxFileSize+1))
	if err != nil {
		_ = os.Remove(dest)
		return ImportedFile{}, fmt.Errorf("copy file: %w", err)
	}
	if n > MaxFileSize {
		_ = os.Remove(dest)
		return ImportedFile{}, fmt.Errorf("file too large: %s", origName)
	}

	return ImportedFile{Name: origName, Path: rel, Size: n}, nil
}

// SafeUploadName validates a single file name for workspace writes.
func SafeUploadName(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = filepath.Base(filepath.ToSlash(name))
	if name == "" || name == "." || name == ".." {
		return "", fmt.Errorf("invalid filename")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", fmt.Errorf("invalid filename")
	}
	if strings.HasPrefix(name, ".") {
		return "", fmt.Errorf("hidden files not allowed")
	}
	return name, nil
}
