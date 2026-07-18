// Filesystem tools scoped to the light-apps directory (data/light-apps/).
// Paths are relative to that root, so the agent addresses files as
// "<app_id>/app.py" without being able to escape above it.
package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/OptLTD/swiflow/internal/lightapp"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/library/support"
	"github.com/pkg/browser"
)

// LightAppRoots holds the base directory for light apps (e.g. data/light-apps).
type LightAppRoots struct{ Base string }

// -- light_app_read --

type lightAppReadTool struct{ la LightAppRoots }

func (t *lightAppReadTool) Name() string { return "light_app_read" }
func (t *lightAppReadTool) Description() string {
	return "Read a file from a light app directory. Path must be \"<app_id>/<file>\"."
}
func (t *lightAppReadTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path within data/light-apps/, e.g. \"abc123/app.py\".",
			},
		},
		"required": []string{"path"},
	}
}
func (t *lightAppReadTool) Execute(_ context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	full, err := support.SandboxPath(t.la.Base, path)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(full)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(b), nil
}

// -- light_app_write --

type lightAppWriteTool struct{ la LightAppRoots }

func (t *lightAppWriteTool) Name() string { return "light_app_write" }
func (t *lightAppWriteTool) Description() string {
	return "Write (create or overwrite) a file in a light app directory. Path must be \"<app_id>/<file>\"."
}
func (t *lightAppWriteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path within data/light-apps/, e.g. \"abc123/app.py\".",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Full UTF-8 content to write.",
			},
		},
		"required": []string{"path", "content"},
	}
}
func (t *lightAppWriteTool) Execute(_ context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	full, err := support.SandboxPath(t.la.Base, path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return fmt.Sprintf("wrote %s (%d bytes)", path, len(content)), nil
}

// -- light_app_ls --

type lightAppLsTool struct{ la LightAppRoots }

func (t *lightAppLsTool) Name() string { return "light_app_ls" }
func (t *lightAppLsTool) Description() string {
	return "List files in a light app directory. Pass \"<app_id>\" or \"<app_id>/<subdir>\"."
}
func (t *lightAppLsTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path within data/light-apps/ to list, e.g. \"abc123\".",
			},
		},
		"required": []string{"path"},
	}
}
func (t *lightAppLsTool) Execute(_ context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	full, err := support.SandboxPath(t.la.Base, path)
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return "", fmt.Errorf("ls %s: %w", path, err)
	}
	var lines []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		lines = append(lines, name)
	}
	return strings.Join(lines, "\n"), nil
}

// -- light_app_create --

type lightAppCreateTool struct {
	st store.Store
	la LightAppRoots
}

func (t *lightAppCreateTool) Name() string { return "light_app_create" }
func (t *lightAppCreateTool) Description() string {
	return "Register a new light app in the database and create its directory under data/light-apps/. Returns the app object including its id."
}
func (t *lightAppCreateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":        map[string]any{"type": "string", "description": "Human-readable name for the app."},
			"description": map[string]any{"type": "string", "description": "Short description of what the app does."},
			"runtime": map[string]any{
				"type":        "string",
				"enum":        []string{"python", "static"},
				"description": "\"python\" for Flask/FastAPI apps, \"static\" for plain HTML/JS.",
			},
			"entry_point": map[string]any{
				"type":        "string",
				"description": "Main file name, e.g. \"app.py\" or \"index.html\". Defaults to runtime default.",
			},
		},
		"required": []string{"name", "runtime"},
	}
}
func (t *lightAppCreateTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return "", fmt.Errorf("name required")
	}
	runtime, _ := args["runtime"].(string)
	if runtime != "python" && runtime != "static" {
		return "", fmt.Errorf("runtime must be python or static")
	}
	entryPoint, _ := args["entry_point"].(string)
	if entryPoint == "" {
		if runtime == "python" {
			entryPoint = "app.py"
		} else {
			entryPoint = "index.html"
		}
	}
	description, _ := args["description"].(string)
	id := support.NewID()
	// Create app directory.
	if err := os.MkdirAll(filepath.Join(t.la.Base, id), 0o755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	a := &store.LightApp{
		ID:          id,
		Name:        name,
		Description: description,
		Runtime:     runtime,
		EntryPoint:  entryPoint,
		Status:      "stopped",
	}
	if err := t.st.CreateLightApp(ctx, a); err != nil {
		return "", fmt.Errorf("create: %w", err)
	}
	out, _ := json.Marshal(a)
	return string(out), nil
}

// -- light_app_launch --

type lightAppLaunchTool struct {
	st  store.Store
	mgr LaunchManager
}

// LaunchManager is the subset of lightapp.Manager used by the tool.
type LaunchManager interface {
	Launch(ctx context.Context, id string, cfg lightapp.LaunchConfig) (url string, port int, err error)
	Status(id string) string
}

func (t *lightAppLaunchTool) Name() string { return "light_app_launch" }
func (t *lightAppLaunchTool) Description() string {
	return "Launch a light app and return its local URL. The app must already be created via light_app_create."
}
func (t *lightAppLaunchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{"type": "string", "description": "App ID returned by light_app_create."},
		},
		"required": []string{"id"},
	}
}
func (t *lightAppLaunchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	id, _ := args["id"].(string)
	if id == "" {
		return "", fmt.Errorf("id required")
	}
	a, err := t.st.GetLightAppByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("app not found: %w", err)
	}
	if t.mgr == nil {
		return "", fmt.Errorf("launch manager unavailable")
	}
	url, port, err := t.mgr.Launch(ctx, id, lightapp.LaunchConfig{
		EntryPoint: a.EntryPoint,
		Runtime:    lightapp.Runtime(a.Runtime),
	})
	if err != nil {
		return "", fmt.Errorf("launch: %w", err)
	}
	_ = t.st.UpdateLightApp(ctx, id, map[string]any{"status": "running", "port": port})
	out, _ := json.Marshal(map[string]any{"url": url, "port": port})
	return string(out), nil
}

// -- light_app_open --

type lightAppOpenTool struct{}

func (t *lightAppOpenTool) Name() string { return "light_app_open" }
func (t *lightAppOpenTool) Description() string {
	return "Open a light app URL in the desktop browser. Call this after light_app_launch with the url it returned."
}
func (t *lightAppOpenTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The http URL to open, e.g. \"http://127.0.0.1:30000\".",
			},
		},
		"required": []string{"url"},
	}
}
func (t *lightAppOpenTool) Execute(_ context.Context, args map[string]any) (string, error) {
	raw, _ := args["url"].(string)
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("url must be a valid http or https URL")
	}
	if err := browser.OpenURL(parsed.String()); err != nil {
		return "", fmt.Errorf("open: %w", err)
	}
	return "opened " + parsed.String(), nil
}

// RegisterLightAppFS registers the light_app_read/write/ls tools.
func RegisterLightAppFS(r *Registry, la LightAppRoots) {
	r.Register(&lightAppReadTool{la: la})
	r.Register(&lightAppWriteTool{la: la})
	r.Register(&lightAppLsTool{la: la})
}

// RegisterLightAppTools registers all light-app agent tools.
func RegisterLightAppTools(r *Registry, la LightAppRoots, st store.Store, mgr LaunchManager) {
	r.Register(&lightAppReadTool{la: la})
	r.Register(&lightAppWriteTool{la: la})
	r.Register(&lightAppLsTool{la: la})
	r.Register(&lightAppCreateTool{st: st, la: la})
	r.Register(&lightAppLaunchTool{st: st, mgr: mgr})
	r.Register(&lightAppOpenTool{})
}
