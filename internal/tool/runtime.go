package tool

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

const runtimeOutputMax = 256 * 1024

func parseTimeout(args map[string]any) int {
	timeout := 60
	if to, ok := args["timeout"].(float64); ok && to > 0 {
		timeout = int(to)
	}
	if timeout < 1 {
		timeout = 1
	}
	if timeout > 3600 {
		timeout = 3600
	}
	return timeout
}

func runCommand(ctx context.Context, ws WorkspaceRoots, bin string, binArgs []string, timeout int) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, bin, binArgs...)
	base := WorkspaceBase(ctx, ws.Base)
	if base != "" {
		cmd.Dir = base
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	runErr := cmd.Run()
	result := truncateOutput(out.String())
	if runErr != nil {
		if cctx.Err() == context.DeadlineExceeded {
			return result + "\n[timeout]", fmt.Errorf("timed out after %ds", timeout)
		}
		return result, fmt.Errorf("failed: %w", runErr)
	}
	if result == "" {
		return "(no output)", nil
	}
	return result, nil
}

func truncateOutput(s string) string {
	if len(s) > runtimeOutputMax {
		return s[:runtimeOutputMax] + "\n...[truncated]"
	}
	return s
}
