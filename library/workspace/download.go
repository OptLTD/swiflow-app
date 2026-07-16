package workspace

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/OptLTD/swiflow/library/support"
)

// BinaryPayload is a JSON-safe workspace file payload for preview clients.
type BinaryPayload struct {
	Path     string `json:"path"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
	Size     int    `json:"size"`
}

// ReadBinaryFile reads a workspace file and returns a base64-encoded payload.
func ReadBinaryFile(workspaceDir, path string) (*BinaryPayload, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("path required")
	}
	full, err := support.SandboxPath(workspaceDir, path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory")
	}
	if info.Size() > MaxFileSize {
		return nil, fmt.Errorf("file too large")
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, err
	}
	return &BinaryPayload{
		Path:     path,
		Encoding: "base64",
		Content:  base64.StdEncoding.EncodeToString(data),
		Size:     len(data),
	}, nil
}
