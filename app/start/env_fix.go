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
	cmd := exec.Command(shell, "-ilc",
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

// Fix reads shell configuration and sets PATH.
func Fix() error {
	return FixVars([]string{"PATH"})
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
