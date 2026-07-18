// Filesystem tools, workspace-sandboxed. Spec §8.
package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/OptLTD/swiflow/library/support"
)

// WorkspaceRoots holds the agent workspace base.
type WorkspaceRoots struct{ Base string }

type readFileTool struct{ ws WorkspaceRoots }

func (t *readFileTool) Name() string        { return "fs_read" }
func (t *readFileTool) Description() string {
	return "Read a UTF-8 text file from the workspace. Not for light apps — use light_app_read instead."
}
func (t *readFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "Path relative to the workspace (also accepts @/… where @/ is the workspace root)."},
		},
		"required": []string{"path"},
	}
}
func (t *readFileTool) Execute(_ context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	full, err := support.SandboxPath(t.ws.Base, path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	const max = 256 * 1024
	if len(data) > max {
		return string(data[:max]) + "\n...[truncated]", nil
	}
	return string(data), nil
}

type writeFileTool struct{ ws WorkspaceRoots }

func (t *writeFileTool) Name() string { return "fs_write" }
func (t *writeFileTool) Description() string {
	return "Write text to a workspace file (creates/overwrites). Not for light apps — use light_app_write instead."
}
func (t *writeFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string"},
			"content": map[string]any{"type": "string"},
		},
		"required": []string{"path", "content"},
	}
}
func (t *writeFileTool) Execute(_ context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	full, err := support.SandboxPath(t.ws.Base, path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("wrote %s", path), nil
}

type listFilesTool struct{ ws WorkspaceRoots }

func (t *listFilesTool) Name() string { return "fs_list" }
func (t *listFilesTool) Description() string {
	return "List entries in a workspace directory. Not for light apps — use light_app_list / light_app_ls instead."
}
func (t *listFilesTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "default": "."},
		},
	}
}
func (t *listFilesTool) Execute(_ context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}
	full, err := support.SandboxPath(t.ws.Base, path)
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		b.WriteString(name)
		b.WriteString("\n")
	}
	return b.String(), nil
}

type editFileTool struct{ ws WorkspaceRoots }

func (t *editFileTool) Name() string { return "fs_edit" }
func (t *editFileTool) Description() string {
	return "Replace a unique old string with a new string in a file."
}
func (t *editFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string"},
			"old":  map[string]any{"type": "string"},
			"new":  map[string]any{"type": "string"},
		},
		"required": []string{"path", "old", "new"},
	}
}
func (t *editFileTool) Execute(_ context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	oldS, _ := args["old"].(string)
	newS, _ := args["new"].(string)
	full, err := support.SandboxPath(t.ws.Base, path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	cnt := strings.Count(string(data), oldS)
	if cnt == 0 {
		return "", fmt.Errorf("old string not found in %s", path)
	}
	if cnt > 1 {
		return "", fmt.Errorf("old string not unique in %s (%d occurrences)", path, cnt)
	}
	updated := strings.Replace(string(data), oldS, newS, 1)
	if err := os.WriteFile(full, []byte(updated), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("edited %s", path), nil
}

// RegisterFS registers the filesystem tools with the registry.
func RegisterFS(r *Registry, ws WorkspaceRoots) {
	r.Register(&readFileTool{ws: ws})
	r.Register(&writeFileTool{ws: ws})
	r.Register(&listFilesTool{ws: ws})
	r.Register(&editFileTool{ws: ws})
}
