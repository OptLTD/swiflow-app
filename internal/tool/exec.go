// Shell execution tool (group:runtime), gated by config. Spec §8.
package tool

import (
	"context"
	"fmt"
)

const ToolExec = "exec"

var runtimeToolNames = []string{ToolExec}

// RuntimeToolNames returns workspace execution tool names.
func RuntimeToolNames() []string {
	return append([]string(nil), runtimeToolNames...)
}

// IsRuntimeTool reports whether name is the workspace exec tool.
func IsRuntimeTool(name string) bool {
	return name == ToolExec
}

type execTool struct {
	ws      WorkspaceRoots
	allowed bool
}

func (t *execTool) Name() string { return ToolExec }
func (t *execTool) Description() string {
	return "Run a shell command in the workspace (sh -c). Do NOT use exec to browse or edit data/light-apps — use light_app_list/read/write/ls instead."
}
func (t *execTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "Shell command to run in the workspace directory.",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout in seconds (1–3600, default 60).",
				"default":     60,
			},
		},
		"required": []string{"command"},
	}
}
func (t *execTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if !t.allowed {
		return "", fmt.Errorf("exec is disabled (set tools.exec_enabled or SWIFLOW_EXEC=true)")
	}
	command, _ := args["command"].(string)
	if command == "" {
		return "", fmt.Errorf("command is required")
	}
	return runCommand(ctx, t.ws, "sh", []string{"-c", command}, parseTimeout(args))
}

// RegisterExec registers the exec tool. It appears in the admin UI even when
// disabled; execution and LLM exposure stay off until enabled.
func RegisterExec(r *Registry, ws WorkspaceRoots, enabled bool) {
	r.Register(&execTool{ws: ws, allowed: enabled})
	if !enabled {
		r.SetEnabled(ToolExec, false)
	}
}
