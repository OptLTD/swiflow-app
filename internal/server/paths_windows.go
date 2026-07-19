//go:build windows

package server

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// openPathInOS opens a file or directory with the shell default association.
// Do not use CREATE_NO_WINDOW here — that flag breaks Explorer / GUI apps.
func openPathInOS(path string) error {
	abs := filepath.Clean(path)
	ptr, err := windows.UTF16PtrFromString(abs)
	if err != nil {
		return err
	}
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	if err := windows.ShellExecute(0, verb, ptr, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("ShellExecute open: %w", err)
	}
	return nil
}

// revealPathInOS selects the file in Explorer.
func revealPathInOS(path string) error {
	abs := filepath.Clean(path)
	// /select,<path> must be a single argument (no space after the comma).
	cmd := exec.Command("explorer.exe", "/select,"+abs)
	if err := cmd.Start(); err != nil {
		// Fallback: open the parent folder.
		return openPathInOS(filepath.Dir(abs))
	}
	return nil
}
