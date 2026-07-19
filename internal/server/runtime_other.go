//go:build !windows

package server

import "os/exec"

func hideConsoleWindow(cmd *exec.Cmd) {}

func registryPATH() string { return "" }

func isIgnoredRuntimePath(path string) bool {
	_ = path
	return false
}

func knownRuntimeCandidatesWindowsShim(name string) []string {
	_ = name
	return nil
}
