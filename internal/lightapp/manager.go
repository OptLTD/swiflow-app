// Package lightapp manages the lifecycle of AI-generated light applications.
// Each app is a directory under data/light-apps/<id>/ containing either a
// Python entry point (app.py) or a static index.html.
package lightapp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Runtime identifies the type of light app.
type Runtime string

// LaunchConfig holds the parameters for launching a light app.
type LaunchConfig struct {
	EntryPoint string
	Runtime    Runtime
	ExtraEnv   map[string]string
	// BaseDir overrides Manager.baseDir for this launch (per-tenant light-apps root).
	BaseDir string
}

const (
	RuntimePython  Runtime = "python"
	RuntimeStatic  Runtime = "static"
)

// AppState is the in-process runtime state of a running app.
type AppState struct {
	Port    int
	URL     string
	cmd     *exec.Cmd        // nil for static
	server  *http.Server     // non-nil for static
	cancel  context.CancelFunc
}

// Manager tracks running light apps keyed by app ID.
type Manager struct {
	mu      sync.Mutex
	running map[string]*AppState
	baseDir string // absolute path to data/light-apps
}

// NewManager creates a Manager rooted at baseDir (e.g. data/light-apps).
func NewManager(baseDir string) *Manager {
	return &Manager{
		running: map[string]*AppState{},
		baseDir: baseDir,
	}
}

// AppDir returns the directory for a given app ID under the manager default base.
func (m *Manager) AppDir(id string) string {
	return m.AppDirAt(m.baseDir, id)
}

// AppDirAt returns the directory for app ID under base (falls back to manager base).
func (m *Manager) AppDirAt(base, id string) string {
	if base == "" {
		base = m.baseDir
	}
	return filepath.Join(base, id)
}

// EnsureDir creates the app directory if it doesn't exist.
func (m *Manager) EnsureDir(id string) error {
	return m.EnsureDirAt(m.baseDir, id)
}

// EnsureDirAt creates the app directory under base.
func (m *Manager) EnsureDirAt(base, id string) error {
	return os.MkdirAll(m.AppDirAt(base, id), 0o755)
}

// Status returns "running" if the app is currently tracked, else "stopped".
func (m *Manager) Status(id string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.running[id]; ok {
		return "running"
	}
	return "stopped"
}

// RunningPort returns the port for a running app, or 0.
func (m *Manager) RunningPort(id string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.running[id]; ok {
		return s.Port
	}
	return 0
}

// Launch starts an app process. If already running, it is stopped and restarted
// so updated ExtraEnv / files take effect.
func (m *Manager) Launch(ctx context.Context, id string, cfg LaunchConfig) (url string, port int, err error) {
	m.Stop(id)

	appDir := m.AppDirAt(cfg.BaseDir, id)
	entryPoint := cfg.EntryPoint
	if entryPoint == "" {
		if cfg.Runtime == RuntimeStatic {
			entryPoint = "index.html"
		} else {
			entryPoint = "app.py"
		}
	}
	if !filepath.IsAbs(entryPoint) {
		entryPoint = filepath.Join(appDir, entryPoint)
	}
	if _, statErr := os.Stat(entryPoint); statErr != nil {
		return "", 0, fmt.Errorf("entry point missing: %s (write the app files before launch)", entryPoint)
	}

	port, err = freePort()
	if err != nil {
		return "", 0, fmt.Errorf("find free port: %w", err)
	}

	appCtx, cancel := context.WithCancel(context.Background())
	state := &AppState{Port: port, cancel: cancel}

	switch cfg.Runtime {
	case RuntimeStatic:
		state.server = serveStatic(appCtx, appDir, port, cfg.ExtraEnv)
	case RuntimePython:
		cmd, launchErr := launchPython(appCtx, appDir, entryPoint, port, cfg.ExtraEnv)
		if launchErr != nil {
			cancel()
			return "", 0, fmt.Errorf("launch python: %w", launchErr)
		}
		state.cmd = cmd
	default:
		cancel()
		return "", 0, fmt.Errorf("unknown runtime: %s", cfg.Runtime)
	}

	state.URL = fmt.Sprintf("http://127.0.0.1:%d", port)

	m.mu.Lock()
	m.running[id] = state
	m.mu.Unlock()

	// Watch process exit and auto-clean state (only if still this instance).
	go func() {
		if state.cmd != nil {
			_ = state.cmd.Wait()
		} else {
			<-appCtx.Done()
		}
		m.mu.Lock()
		if cur, ok := m.running[id]; ok && cur == state {
			delete(m.running, id)
		}
		m.mu.Unlock()
		slog.Info("light app stopped", "id", id, "port", port)
	}()

	slog.Info("light app launched", "id", id, "runtime", cfg.Runtime, "port", port)
	return state.URL, port, nil
}

// Stop terminates a running app.
func (m *Manager) Stop(id string) bool {
	m.mu.Lock()
	state, ok := m.running[id]
	if ok {
		delete(m.running, id)
	}
	m.mu.Unlock()
	if !ok {
		return false
	}
	state.cancel()
	if state.cmd != nil && state.cmd.Process != nil {
		_ = state.cmd.Process.Kill()
	}
	if state.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = state.server.Shutdown(ctx)
	}
	return true
}

// StopAll stops all running apps.
func (m *Manager) StopAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.running))
	for id := range m.running {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		m.Stop(id)
	}
}

func launchPython(ctx context.Context, appDir, entryPoint string, port int, extraEnv map[string]string) (*exec.Cmd, error) {
	python := pythonBin()
	cmd := exec.CommandContext(ctx, python, entryPoint)
	cmd.Dir = appDir
	env := append(os.Environ(),
		fmt.Sprintf("PORT=%d", port),
		"PYTHONDONTWRITEBYTECODE=1",
	)
	for k, v := range extraEnv {
		env = append(env, k+"="+v)
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	// Wait briefly for the server to bind.
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return cmd, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	// Process started but didn't bind — still return cmd so caller can stop it.
	return cmd, nil
}

func serveStatic(ctx context.Context, dir string, port int, extraEnv map[string]string) *http.Server {
	envJS := buildEnvJS(extraEnv)
	fs := http.FileServer(http.Dir(dir))
	mux := http.NewServeMux()
	mux.HandleFunc("/__env.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(envJS))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if shouldInjectEnv(r.URL.Path) {
			serveHTMLWithEnv(w, r, dir, envJS)
			return
		}
		fs.ServeHTTP(w, r)
	})
	srv := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: mux,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Warn("static light app server", "error", err)
		}
	}()
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()
	return srv
}

// buildEnvJS returns a script that exposes window.swiflow.env(key).
// Missing or empty keys throw so apps and tests fail loudly.
func buildEnvJS(extraEnv map[string]string) string {
	if extraEnv == nil {
		extraEnv = map[string]string{}
	}
	raw, err := json.Marshal(extraEnv)
	if err != nil {
		raw = []byte("{}")
	}
	return `(function(){
  var data = ` + string(raw) + `;
  window.swiflow = {
    env: function(key) {
      if (key == null || key === "") {
        throw new Error("swiflow.env: key required");
      }
      if (!Object.prototype.hasOwnProperty.call(data, key)) {
        throw new Error('swiflow.env: missing "' + key + '" — set it in Light Apps → Environment Variables, then re-launch');
      }
      var v = data[key];
      if (v == null || v === "") {
        throw new Error('swiflow.env: "' + key + '" is empty — set it in Light Apps → Environment Variables, then re-launch');
      }
      return v;
    }
  };
})();
`
}

func shouldInjectEnv(path string) bool {
	if path == "/" || path == "" {
		return true
	}
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm")
}

const envMarker = "/*swiflow-env*/"

func injectEnvIntoHTML(html, envJS string) string {
	if strings.Contains(html, envMarker) {
		return html
	}
	injected := "<script>" + envMarker + "\n" + envJS + "</script>\n"
	lower := strings.ToLower(html)
	if i := strings.Index(lower, "<head>"); i >= 0 {
		insertAt := i + len("<head>")
		return html[:insertAt] + "\n" + injected + html[insertAt:]
	}
	if i := strings.Index(lower, "<body"); i >= 0 {
		return html[:i] + injected + html[i:]
	}
	return injected + html
}

func serveHTMLWithEnv(w http.ResponseWriter, r *http.Request, dir, envJS string) {
	rel := strings.TrimPrefix(r.URL.Path, "/")
	if rel == "" {
		rel = "index.html"
	}
	full := filepath.Join(dir, filepath.FromSlash(rel))
	data, err := os.ReadFile(full)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	html := injectEnvIntoHTML(string(data), envJS)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(html))
}

func freePort() (int, error) {
	for port := 30000; port <= 30100; port++ {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}
		l.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no free port available in range 30000-30100")
}

func pythonBin() string {
	for _, bin := range []string{"python3", "python"} {
		if path, err := exec.LookPath(bin); err == nil {
			return path
		}
	}
	return "python3"
}
