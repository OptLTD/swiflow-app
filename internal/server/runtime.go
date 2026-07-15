package server

import (
	"context"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type runtimeBinary struct {
	Found   bool   `json:"found"`
	Path    string `json:"path"`
	Version string `json:"version"`
}

func (s *Server) getRuntime(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"python3": detectRuntimeBinary([]string{"python3", "python"}),
		"node":    detectRuntimeBinary([]string{"node"}),
	})
}

func detectRuntimeBinary(names []string) runtimeBinary {
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		return runtimeBinary{
			Found:   true,
			Path:    path,
			Version: probeVersion(path),
		}
	}
	return runtimeBinary{}
}

func probeVersion(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	return strings.TrimSpace(line)
}
