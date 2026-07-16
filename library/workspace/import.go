// Package workspace provides helpers for importing files into the agent workspace.
package workspace

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/OptLTD/swiflow/library/support"
)

const MaxFileSize = 64 << 20 // 64 MiB

// ImportedFile describes a file written into the workspace.
type ImportedFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// ImportFiles copies files from absolute source paths into a workspace directory.
func ImportFiles(workspaceDir, relDir string, sourcePaths []string) ([]ImportedFile, error) {
	if len(sourcePaths) == 0 {
		return nil, fmt.Errorf("no files")
	}
	destDir, err := support.SandboxPath(workspaceDir, relDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir failed: %w", err)
	}

	uploaded := make([]ImportedFile, 0, len(sourcePaths))
	for _, src := range sourcePaths {
		item, err := importOne(workspaceDir, relDir, src)
		if err != nil {
			return uploaded, err
		}
		uploaded = append(uploaded, item)
	}
	return uploaded, nil
}

func importOne(workspaceDir, relDir, src string) (ImportedFile, error) {
	name, err := SafeUploadName(filepath.Base(src))
	if err != nil {
		return ImportedFile{}, err
	}

	rel := name
	if relDir != "." {
		rel = filepath.ToSlash(filepath.Join(relDir, name))
	}
	dest, err := support.SandboxPath(workspaceDir, rel)
	if err != nil {
		return ImportedFile{}, err
	}

	info, err := os.Stat(src)
	if err != nil {
		return ImportedFile{}, fmt.Errorf("stat source: %w", err)
	}
	if info.IsDir() {
		return ImportedFile{}, fmt.Errorf("directories not supported: %s", name)
	}
	if info.Size() > MaxFileSize {
		return ImportedFile{}, fmt.Errorf("file too large: %s", name)
	}

	in, err := os.Open(src)
	if err != nil {
		return ImportedFile{}, fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return ImportedFile{}, fmt.Errorf("create dest: %w", err)
	}
	defer out.Close()

	n, err := io.Copy(out, io.LimitReader(in, MaxFileSize+1))
	if err != nil {
		return ImportedFile{}, fmt.Errorf("copy file: %w", err)
	}
	if n > MaxFileSize {
		_ = os.Remove(dest)
		return ImportedFile{}, fmt.Errorf("file too large: %s", name)
	}

	return ImportedFile{Name: name, Path: rel, Size: n}, nil
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
