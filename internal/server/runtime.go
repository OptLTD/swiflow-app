package server

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	emb "github.com/OptLTD/swiflow/embed"
)

type runtimeBinary struct {
	Found   bool   `json:"found"`
	Path    string `json:"path"`
	Version string `json:"version"`
}

type runtimeInstallState struct {
	mu      sync.Mutex
	running map[string]bool // name -> installing
}

var runtimeInstalls = &runtimeInstallState{running: map[string]bool{}}

func (s *Server) getRuntime(w http.ResponseWriter, _ *http.Request) {
	refreshProcessPATH()
	writeJSON(w, http.StatusOK, map[string]any{
		"python3":    detectRuntimeBinary([]string{"python3", "python"}),
		"uv":         detectRuntimeBinary([]string{"uv"}),
		"uvx":        detectRuntimeBinary([]string{"uvx"}),
		"node":       detectRuntimeBinary([]string{"node"}),
		"npx":        detectRuntimeBinary([]string{"npx"}),
		"os":         runtime.GOOS,
		"installing": runtimeInstalls.snapshot(),
	})
}

func (s *Server) installRuntime(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"` // uvx-py | js-npx
		Mode string `json:"mode"` // mainland | standard
	}
	if !bindJSON(w, r, &in) {
		return
	}
	name := strings.TrimSpace(in.Name)
	if name != "uvx-py" && name != "js-npx" {
		writeErr(w, http.StatusBadRequest, ErrInvalidRuntimeInstallName)
		return
	}
	mode := strings.TrimSpace(in.Mode)
	if mode == "" {
		mode = "mainland"
	}
	if mode != "mainland" && mode != "standard" {
		writeErr(w, http.StatusBadRequest, ErrInvalidRuntimeInstallMode)
		return
	}

	runtimeInstalls.mu.Lock()
	if runtimeInstalls.running[name] {
		runtimeInstalls.mu.Unlock()
		writeErr(w, http.StatusConflict, ErrInstallAlreadyRunning)
		return
	}
	runtimeInstalls.running[name] = true
	runtimeInstalls.mu.Unlock()

	scriptDir := filepath.Join(filepath.Dir(s.cfg.DBPath), "install-scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		runtimeInstalls.clear(name)
		writeErr(w, http.StatusInternalServerError, ErrInternalError, err.Error())
		return
	}

	scriptName, cmd, args, err := runtimeInstallCmd(name, mode, scriptDir)
	if err != nil {
		runtimeInstalls.clear(name)
		writeErr(w, http.StatusBadRequest, ErrInternalError, err.Error())
		return
	}
	bin, err := emb.GetScript(scriptName)
	if err != nil || len(bin) == 0 {
		runtimeInstalls.clear(name)
		writeErr(w, http.StatusInternalServerError, ErrScriptNotFound, scriptName)
		return
	}
	scriptPath := filepath.Join(scriptDir, scriptName)
	if err := os.WriteFile(scriptPath, bin, 0o755); err != nil {
		runtimeInstalls.clear(name)
		writeErr(w, http.StatusInternalServerError, ErrInternalError, err.Error())
		return
	}

	logPath := scriptPath + ".log"
	logFile, err := os.Create(logPath)
	if err != nil {
		runtimeInstalls.clear(name)
		writeErr(w, http.StatusInternalServerError, ErrInternalError, err.Error())
		return
	}

	c := exec.Command(cmd, args...)
	c.Dir = scriptDir
	c.Stdout = logFile
	c.Stderr = logFile
	hideConsoleWindow(c)
	if err := c.Start(); err != nil {
		_ = logFile.Close()
		runtimeInstalls.clear(name)
		writeErr(w, http.StatusInternalServerError, ErrInstallStartFailed, err.Error())
		return
	}

	go func() {
		defer logFile.Close()
		_ = c.Wait()
		runtimeInstalls.clear(name)
	}()

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "started",
		"name":    name,
		"mode":    mode,
		"script":  scriptName,
		"log":     logPath,
		"message": "installation started; poll GET /api/runtime until binaries appear",
	})
}

func runtimeInstallCmd(name, mode, scriptDir string) (scriptName, cmd string, args []string, err error) {
	switch runtime.GOOS {
	case "windows":
		scriptName = "win-" + name + ".ps1"
		scriptPath := filepath.Join(scriptDir, scriptName)
		// powershell.exe + Hidden avoids console flash under Wails (-H windowsgui).
		cmd = "powershell.exe"
		args = []string{
			"-NoProfile",
			"-NonInteractive",
			"-WindowStyle", "Hidden",
			"-ExecutionPolicy", "Bypass",
			"-File", scriptPath,
			"-mode", mode,
		}
	case "darwin", "linux":
		// Archive ships bash scripts as mac-*; reuse for linux.
		scriptName = "mac-" + name + ".sh"
		scriptPath := filepath.Join(scriptDir, scriptName)
		cmd = "sh"
		args = []string{scriptPath, mode}
	default:
		return "", "", nil, errUnsupportedOS(runtime.GOOS)
	}
	return scriptName, cmd, args, nil
}

type unsupportedOSError string

func (e unsupportedOSError) Error() string { return "unsupported os: " + string(e) }
func errUnsupportedOS(osName string) error  { return unsupportedOSError(osName) }

func (st *runtimeInstallState) clear(name string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.running, name)
}

func (st *runtimeInstallState) snapshot() map[string]bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	out := make(map[string]bool, len(st.running))
	for k, v := range st.running {
		out[k] = v
	}
	return out
}

func detectRuntimeBinary(names []string) runtimeBinary {
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		if isIgnoredRuntimePath(path) {
			continue
		}
		return runtimeBinary{
			Found:   true,
			Path:    path,
			Version: probeVersion(path),
		}
	}
	// Fall back to well-known install locations (PATH may lag after install).
	for _, name := range names {
		for _, cand := range knownRuntimeCandidates(name) {
			if st, err := os.Stat(cand); err != nil || st.IsDir() {
				continue
			}
			if isIgnoredRuntimePath(cand) {
				continue
			}
			return runtimeBinary{
				Found:   true,
				Path:    cand,
				Version: probeVersion(cand),
			}
		}
	}
	return runtimeBinary{}
}

func probeVersion(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	hideConsoleWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// uvx/npx sometimes use -V
		cmd = exec.CommandContext(ctx, path, "-V")
		hideConsoleWindow(cmd)
		out, err = cmd.CombinedOutput()
		if err != nil {
			return ""
		}
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	return strings.TrimSpace(line)
}

// refreshProcessPATH reloads PATH so newly installed tools are visible without
// restarting the desktop process. Platform-specific details live in runtime_*.go.
func refreshProcessPATH() {
	parts := []string{os.Getenv("PATH"), registryPATH()}
	parts = append(parts, knownRuntimePathDirs()...)
	merged := mergePATH(parts...)
	if merged != "" {
		_ = os.Setenv("PATH", merged)
		// Windows LookPath also checks Path.
		_ = os.Setenv("Path", merged)
	}
}

func mergePATH(parts ...string) string {
	sep := string(os.PathListSeparator)
	seen := map[string]bool{}
	out := make([]string, 0, 32)
	for _, part := range parts {
		for _, entry := range strings.Split(part, sep) {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			key := entry
			if runtime.GOOS == "windows" {
				key = strings.ToLower(entry)
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, entry)
		}
	}
	return strings.Join(out, sep)
}

func knownRuntimePathDirs() []string {
	home, _ := os.UserHomeDir()
	localApp := os.Getenv("LOCALAPPDATA")
	dirs := []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, ".cargo", "bin"),
	}
	if localApp != "" {
		dirs = append(dirs,
			filepath.Join(localApp, "Programs", "nodejs"),
		)
	}
	if runtime.GOOS == "windows" {
		dirs = append(dirs,
			`C:\Program Files\nodejs`,
			`C:\Program Files (x86)\nodejs`,
		)
	}
	if runtime.GOOS == "darwin" {
		dirs = append(dirs, "/usr/local/bin", "/opt/homebrew/bin")
	}
	existing := make([]string, 0, len(dirs))
	for _, d := range dirs {
		if d == "" {
			continue
		}
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			existing = append(existing, d)
		}
	}
	return existing
}

func knownRuntimeCandidates(name string) []string {
	exe := name
	if runtime.GOOS == "windows" {
		exe = name + ".exe"
		// npm/npx are usually .cmd shims on Windows.
		if name == "npm" || name == "npx" {
			return knownRuntimeCandidatesWindowsShim(name)
		}
	}
	dirs := knownRuntimePathDirs()
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		out = append(out, filepath.Join(d, exe))
	}
	return out
}
