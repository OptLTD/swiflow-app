// Shell execution tool, gated by config. Spec §8. Only registered when
// config.Tools.ExecEnabled is true.
package tool

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

type execTool struct{ ws WorkspaceRoots }

func (t *execTool) Name() string        { return "exec_run" }
func (t *execTool) Description() string { return "Run a shell command in the workspace." }
func (t *execTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{"type": "string"},
			"timeout": map[string]any{"type": "integer", "default": 30},
		},
		"required": []string{"command"},
	}
}
func (t *execTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	command, _ := args["command"].(string)
	timeout := 30
	if to, ok := args["timeout"].(float64); ok && to > 0 {
		timeout = int(to)
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "sh", "-c", command)
	if t.ws.Base != "" {
		cmd.Dir = t.ws.Base
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	runErr := cmd.Run()
	result := out.String()
	const max = 256 * 1024
	if len(result) > max {
		result = result[:max] + "\n...[truncated]"
	}
	if runErr != nil {
		if cctx.Err() == context.DeadlineExceeded {
			return result + "\n[timeout]", fmt.Errorf("command timed out after %ds", timeout)
		}
		return result, fmt.Errorf("command failed: %w", runErr)
	}
	if result == "" {
		return "(no output)", nil
	}
	return result, nil
}

// RegisterExec registers the shell tool if enabled.
func RegisterExec(r *Registry, ws WorkspaceRoots, enabled bool) {
	if !enabled {
		return
	}
	r.Register(&execTool{ws: ws})
}
