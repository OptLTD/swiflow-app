//go:build !windows

package server

import (
	"os/exec"
	"path/filepath"
	"runtime"
)

func openPathInOS(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

func revealPathInOS(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	default:
		return openPathInOS(filepath.Dir(path))
	}
}
