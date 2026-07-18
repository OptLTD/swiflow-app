// Package lightapp manages the lifecycle of AI-generated light applications.
// Each app is a directory under data/light-apps/<id>/ containing either a
// Python entry point (app.py) or a static index.html.
package lightapp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

// AppDir returns the directory for a given app ID.
func (m *Manager) AppDir(id string) string {
	return filepath.Join(m.baseDir, id)
}

// EnsureDir creates the app directory if it doesn't exist.
func (m *Manager) EnsureDir(id string) error {
	return os.MkdirAll(m.AppDir(id), 0o755)
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

// Launch starts an app process. It is idempotent: if already running, returns its URL.
func (m *Manager) Launch(ctx context.Context, id string, cfg LaunchConfig) (url string, port int, err error) {
	m.mu.Lock()
	if s, ok := m.running[id]; ok {
		m.mu.Unlock()
		return s.URL, s.Port, nil
	}
	m.mu.Unlock()

	port, err = freePort()
	if err != nil {
		return "", 0, fmt.Errorf("find free port: %w", err)
	}

	appDir := m.AppDir(id)
	entryPoint := cfg.EntryPoint
	if !filepath.IsAbs(entryPoint) {
		entryPoint = filepath.Join(appDir, entryPoint)
	}

	appCtx, cancel := context.WithCancel(context.Background())
	state := &AppState{Port: port, cancel: cancel}

	switch cfg.Runtime {
	case RuntimeStatic:
		state.server = serveStatic(appCtx, appDir, port)
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

	// Watch process exit and auto-clean state.
	go func() {
		if state.cmd != nil {
			_ = state.cmd.Wait()
		} else {
			<-appCtx.Done()
		}
		m.mu.Lock()
		delete(m.running, id)
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

func serveStatic(ctx context.Context, dir string, port int) *http.Server {
	srv := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: http.FileServer(http.Dir(dir)),
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
