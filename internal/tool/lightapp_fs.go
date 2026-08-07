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
	"github.com/OptLTD/swiflow/library/window"
)

// LightAppRoots holds the base directory for light apps (e.g. data/light-apps).
type LightAppRoots struct{ Base string }

// -- light_app_read --

type lightAppReadTool struct{ la LightAppRoots }

func (t *lightAppReadTool) Name() string { return "light_app_read" }
func (t *lightAppReadTool) Description() string {
	return "Read a file from a light app. Path: \"<app_id>/<file>\". Do NOT use fs_read/exec for light-app files."
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
func (t *lightAppReadTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	full, err := support.SandboxPath(LightAppsBase(ctx, t.la.Base), path)
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
	return "Write a file in a light app. Path: \"<app_id>/<file>\". Do NOT use fs_write/exec for light-app files."
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
func (t *lightAppWriteTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	full, err := support.SandboxPath(LightAppsBase(ctx, t.la.Base), path)
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
	return "List files in a light app directory. Pass \"<app_id>\" or \"<app_id>/<subdir>\". Do NOT use fs_list/exec."
}
func (t *lightAppLsTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "App-relative path, e.g. \"abc123\" or \"abc123/assets\".",
			},
		},
		"required": []string{"path"},
	}
}
func (t *lightAppLsTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path required (use light_app_list to discover app ids)")
	}
	full, err := support.SandboxPath(LightAppsBase(ctx, t.la.Base), path)
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

// -- light_app_list --

type lightAppListTool struct{ st store.Store }

func (t *lightAppListTool) Name() string { return "light_app_list" }
func (t *lightAppListTool) Description() string {
	return "List registered light apps (id, name, runtime, status). Use this to find an existing app before light_app_read/write. Do NOT use fs_list/exec on data/light-apps."
}
func (t *lightAppListTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}
func (t *lightAppListTool) Execute(ctx context.Context, _ map[string]any) (string, error) {
	apps, err := t.st.ListLightApps(ctx)
	if err != nil {
		return "", fmt.Errorf("list: %w", err)
	}
	out, _ := json.Marshal(apps)
	return string(out), nil
}

// -- light_app_create --

type lightAppCreateTool struct {
	st store.Store
	la LightAppRoots
}

func (t *lightAppCreateTool) Name() string { return "light_app_create" }
func (t *lightAppCreateTool) Description() string {
	return "Register a new light app and create its directory. Call ONLY after clarify confirmed purpose, runtime, data storage, and acceptance spec. Next write SPEC.md then sources. Returns the app object including id."
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
	base := LightAppsBase(ctx, t.la.Base)
	// Create app directory.
	if err := os.MkdirAll(filepath.Join(base, id), 0o755); err != nil {
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
	extraEnv, _ := t.st.ListLightAppEnv(ctx)
	url, port, err := t.mgr.Launch(ctx, id, lightapp.LaunchConfig{
		EntryPoint: a.EntryPoint,
		Runtime:    lightapp.Runtime(a.Runtime),
		ExtraEnv:   extraEnv,
		BaseDir:    LightAppsBase(ctx, ""),
	})
	if err != nil {
		return "", fmt.Errorf("launch: %w", err)
	}
	_ = t.st.UpdateLightApp(ctx, id, map[string]any{"status": "running", "port": port})
	out, _ := json.Marshal(map[string]any{"url": url, "port": port, "name": a.Name})
	return string(out), nil
}

// -- light_app_open --

type lightAppOpenTool struct {
	bridge *window.Bridge
}

func (t *lightAppOpenTool) Name() string { return "light_app_open" }
func (t *lightAppOpenTool) Description() string {
	return "Open a light app for the user in a desktop child window (usage/handoff). Call ONLY after browser self-test against SPEC.md has passed. Pass url from light_app_launch and optional title."
}
func (t *lightAppOpenTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The http URL to open, e.g. \"http://127.0.0.1:30000\".",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Optional window title (app name).",
			},
		},
		"required": []string{"url"},
	}
}
func (t *lightAppOpenTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	raw, _ := args["url"].(string)
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("url must be a valid http or https URL")
	}
	title, _ := args["title"].(string)
	if title == "" {
		title = "Light App"
	}
	if t.bridge == nil {
		return "", fmt.Errorf("ui client unavailable")
	}
	rc, ok := RunContextFrom(ctx)
	if !ok || rc.SessionID == "" {
		return "", fmt.Errorf("ui client unavailable")
	}
	return t.bridge.Request(ctx, rc.SessionID, "light_app_open", map[string]any{
		"url":   parsed.String(),
		"title": title,
	})
}

// RegisterLightAppFS registers the light_app_read/write/ls tools.
func RegisterLightAppFS(r *Registry, la LightAppRoots) {
	r.Register(&lightAppReadTool{la: la})
	r.Register(&lightAppWriteTool{la: la})
	r.Register(&lightAppLsTool{la: la})
}

// RegisterLightAppTools registers all light-app agent tools.
func RegisterLightAppTools(r *Registry, la LightAppRoots, st store.Store, mgr LaunchManager, bridge *window.Bridge) {
	r.Register(&lightAppListTool{st: st})
	r.Register(&lightAppReadTool{la: la})
	r.Register(&lightAppWriteTool{la: la})
	r.Register(&lightAppLsTool{la: la})
	r.Register(&lightAppCreateTool{st: st, la: la})
	r.Register(&lightAppLaunchTool{st: st, mgr: mgr})
	r.Register(&lightAppOpenTool{bridge: bridge})
}
