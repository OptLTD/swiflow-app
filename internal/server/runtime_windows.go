//go:build windows

package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const createNoWindow = 0x08000000

func hideConsoleWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}

// registryPATH reads the current Machine+User Path from the registry so tools
// installed after Swiflow started become visible without a process restart.
func registryPATH() string {
	machine := readRegPATH(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`)
	user := readRegPATH(registry.CURRENT_USER, `Environment`)
	return mergePATH(machine, user)
}

func readRegPATH(root registry.Key, path string) string {
	k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	v, _, err := k.GetStringValue("Path")
	if err != nil {
		return ""
	}
	return v
}

func isIgnoredRuntimePath(path string) bool {
	lower := strings.ToLower(filepath.Clean(path))
	// Windows Store python/python3 stubs are not real interpreters.
	if strings.Contains(lower, `\windowsapps\`) {
		return true
	}
	return false
}

func knownRuntimeCandidatesWindowsShim(name string) []string {
	dirs := knownRuntimePathDirs()
	out := make([]string, 0, len(dirs)*2)
	for _, d := range dirs {
		out = append(out,
			filepath.Join(d, name+".cmd"),
			filepath.Join(d, name+".exe"),
		)
	}
	// Official MSI / winget layout.
	for _, d := range []string{`C:\Program Files\nodejs`, filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "nodejs")} {
		if d == "" || d == string(filepath.Separator)+"Programs"+string(filepath.Separator)+"nodejs" {
			continue
		}
		out = append(out,
			filepath.Join(d, name+".cmd"),
			filepath.Join(d, name+".exe"),
		)
	}
	return out
}
