package start

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// FixVars loads environment variables from a login interactive shell
// and sets them into the current process. If vars is empty, all
// variables are imported; otherwise only the listed ones are updated.
func FixVars(vars []string) error {
	// Windows already provides adequate env to GUI apps; do nothing.
	if runtime.GOOS == "windows" {
		return nil
	}

	// Choose default shell by platform; respect SHELL if present.
	defaultShell := "/bin/sh"
	if runtime.GOOS == "darwin" {
		defaultShell = "/bin/zsh"
	}
	shell := os.Getenv("SHELL")
	if strings.TrimSpace(shell) == "" {
		shell = defaultShell
	}

	// Build command: run login+interactive shell, echo delimiters around env output.
	delim := "_SHELL_ENV_DELIMITER_"
	cmd := exec.Command(shell, "-lc",
		"echo -n '"+delim+"'; env; echo -n '"+delim+"'; exit",
	)
	// Prevent Oh My Zsh auto-update or similar prompts from blocking.
	cmd.Env = append(os.Environ(), "DISABLE_AUTO_UPDATE=true")

	// Use user's home as working directory to mimic typical login context.
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		cmd.Dir = home
	}

	out, err := cmd.Output()
	if err != nil {
		return err
	}

	// Extract env output between delimiters.
	stdout := string(out)
	parts := strings.Split(stdout, delim)
	if len(parts) < 3 {
		return errors.New("invalid shell output: delimiter not found")
	}
	raw := parts[1]

	// Strip ANSI escape sequences from output.
	clean := stripANSI(raw)

	// Build quick lookup set for requested vars.
	wantAll := len(vars) == 0
	wanted := map[string]struct{}{}
	for _, v := range vars {
		if v != "" {
			wanted[v] = struct{}{}
		}
	}

	// Parse lines VAR=VALUE and set into current process.
	for _, line := range strings.Split(clean, "\n") {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, val := kv[0], kv[1]
		if wantAll {
			_ = os.Setenv(key, val)
			continue
		}
		if _, ok := wanted[key]; ok {
			_ = os.Setenv(key, val)
		}
	}

	return nil
}

// FixAllVars reads shell configuration and sets all variables.
func FixAllVars() error {
	return FixVars(nil)
}

// stripANSI removes common ANSI escape sequences from a string.
// This helps when shells print colored prompts or messages.
func stripANSI(s string) string {
	// Regex matches CSI sequences like \x1b[...m or other final letters.
	re := regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
	return re.ReplaceAllString(s, "")
}

// Fix reads shell configuration and sets PATH.
func FixPath() error {
	// Prefer OS-native sources to avoid login shell side effects.
	// Strategy:
	// 1) macOS: try `launchctl getenv PATH` for GUI apps
	// 2) macOS: try `/usr/libexec/path_helper -s` and parse PATH="..."
	// 3) Fallback: `shell -lc 'printenv PATH'` (non-login)

	if runtime.GOOS == "windows" {
		return nil
	}

	// Attempt macOS GUI PATH via launchctl
	if runtime.GOOS == "darwin" {
		if path, err := getPathFromLaunchctl(); err == nil && strings.TrimSpace(path) != "" {
			return setCleanPath(path)
		}
		if path, err := getPathFromPathHelper(); err == nil && strings.TrimSpace(path) != "" {
			return setCleanPath(path)
		}
	}

	// Fallback: non-login shell PATH
	if path, err := getPathFromShellNonLogin(); err == nil && strings.TrimSpace(path) != "" {
		return setCleanPath(path)
	}
	return nil
}

// getPathFromLaunchctl reads PATH from launchctl for GUI apps (macOS only).
// This avoids spawning a shell and is the recommended source for GUI processes.
func getPathFromLaunchctl() (string, error) {
	out, err := exec.Command("launchctl", "getenv", "PATH").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// getPathFromPathHelper parses PATH from `/usr/libexec/path_helper -s` output.
// Example: PATH="/usr/local/bin:/usr/bin"; export PATH;
func getPathFromPathHelper() (string, error) {
	out, err := exec.Command("/usr/libexec/path_helper", "-s").Output()
	if err != nil {
		return "", err
	}
	text := string(out)
	re := regexp.MustCompile(`PATH=\"([^\"]+)\"`)
	m := re.FindStringSubmatch(text)
	if len(m) == 2 {
		return strings.TrimSpace(m[1]), nil
	}
	return "", errors.New("PATH not found in path_helper output")
}

// getPathFromShellNonLogin fetches PATH using a non-login shell to reduce side effects.
func getPathFromShellNonLogin() (string, error) {
	defaultShell := "/bin/sh"
	if runtime.GOOS == "darwin" {
		defaultShell = "/bin/zsh"
	}
	shell := os.Getenv("SHELL")
	if strings.TrimSpace(shell) == "" {
		shell = defaultShell
	}
	cmd := exec.Command(shell, "-lc", "printenv PATH")
	// Prevent interactive prompts from blocking.
	cmd.Env = append(os.Environ(), "DISABLE_AUTO_UPDATE=true")
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		cmd.Dir = home
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// setCleanPath sanitizes and sets PATH: removes empty/current entries, dedups,
// and filters out non-existent directories to avoid breaking dev runners.
func setCleanPath(path string) error {
	parts := strings.Split(path, ":")
	clean := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "." {
			continue
		}
		if fi, err := os.Stat(p); err != nil || !fi.IsDir() {
			// Skip non-existent or non-directory entries
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		clean = append(clean, p)
	}
	if len(clean) == 0 {
		return nil
	}
	return os.Setenv("PATH", strings.Join(clean, ":"))
}
