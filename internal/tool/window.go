package tool

import (
	"context"
	"fmt"
	"os"

	"github.com/OptLTD/swiflow/library/support"
	"github.com/OptLTD/swiflow/library/window"
)

type windowTools struct {
	bridge *window.Bridge
	ws     WorkspaceRoots
}

func (t *windowTools) call(ctx context.Context, op string, args map[string]any) (string, error) {
	rc, ok := RunContextFrom(ctx)
	if !ok || rc.SessionID == "" {
		return "", fmt.Errorf("ui client unavailable")
	}
	return t.bridge.Request(ctx, rc.SessionID, op, args)
}

type windowOpenedTool struct{ base *windowTools }

func (t *windowOpenedTool) Name() string { return "window_opened" }
func (t *windowOpenedTool) Description() string {
	return "List file tabs currently open in the user's Swiflow window (not Home/Explore/Settings)."
}
func (t *windowOpenedTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}
func (t *windowOpenedTool) Execute(ctx context.Context, _ map[string]any) (string, error) {
	return t.base.call(ctx, t.Name(), map[string]any{})
}

type windowActiveTool struct{ base *windowTools }

func (t *windowActiveTool) Name() string { return "window_active" }
func (t *windowActiveTool) Description() string {
	return "Get the file tab currently focused in the user's Swiflow window."
}
func (t *windowActiveTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}
func (t *windowActiveTool) Execute(ctx context.Context, _ map[string]any) (string, error) {
	return t.base.call(ctx, t.Name(), map[string]any{})
}

type windowOpenTool struct{ base *windowTools }

func (t *windowOpenTool) Name() string { return "window_open" }
func (t *windowOpenTool) Description() string {
	return "Open (or focus) a workspace file in the user's Swiflow window."
}
func (t *windowOpenTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path relative to the workspace.",
			},
		},
		"required": []string{"path"},
	}
}
func (t *windowOpenTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path required")
	}
	rel := support.NormalizeWorkspaceRel(path)
	full, err := support.SandboxPath(t.base.ws.Base, rel)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(full)
	if err != nil {
		return "", err
	}
	if fi.IsDir() {
		return "", fmt.Errorf("%s is a directory", rel)
	}
	return t.base.call(ctx, t.Name(), map[string]any{"path": rel})
}

// RegisterWindow registers window_* UI tools. bridge may be nil (tools still register but fail at execute).
func RegisterWindow(r *Registry, bridge *window.Bridge, ws WorkspaceRoots) {
	if bridge == nil {
		bridge = window.NewBridge()
	}
	base := &windowTools{bridge: bridge, ws: ws}
	r.Register(&windowOpenedTool{base: base})
	r.Register(&windowActiveTool{base: base})
	r.Register(&windowOpenTool{base: base})
}
