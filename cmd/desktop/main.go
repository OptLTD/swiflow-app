// Package main is the Swiflow desktop application entrypoint using wails3.
// It starts the Swiflow backend in-process and opens a native window with the Vue UI.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	emb "github.com/OptLTD/swiflow/embed"
	"github.com/OptLTD/swiflow/internal/agent"
	"github.com/OptLTD/swiflow/internal/appdb"
	"github.com/OptLTD/swiflow/internal/config"
	"github.com/OptLTD/swiflow/internal/harness"
	"github.com/OptLTD/swiflow/internal/lightapp"
	"github.com/OptLTD/swiflow/internal/mcpclient"
	"github.com/OptLTD/swiflow/internal/observe"
	"github.com/OptLTD/swiflow/internal/schedule"
	"github.com/OptLTD/swiflow/internal/server"
	"github.com/OptLTD/swiflow/internal/skill"
	"github.com/OptLTD/swiflow/internal/tool"
	"github.com/OptLTD/swiflow/library/browser"
	"github.com/OptLTD/swiflow/library/httputil"
	"github.com/OptLTD/swiflow/library/window"
)

func main() {
	start := time.Now()
	ctx := context.Background()

	// 1. Load Swiflow config (.app launches with cwd=/ so use Application Support)
	cfgPath, err := ensureDesktopConfig()
	if err != nil {
		slog.Error("desktop config", "error", err)
		os.Exit(1)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		slog.Error("load config", "path", cfgPath, "error", err)
		os.Exit(1)
	}
	cfg = resolveDesktopPaths(cfg, filepath.Dir(cfgPath))
	// Document extract uses the vision/default provider from Settings; keep enabled on desktop.
	cfg.Tools.DocumentEnabled = true

	// File logs under data dir (same folder as swiflow.db).
	if _, err := observe.SetupFileLog(cfg.DataDir(), slog.LevelInfo); err != nil {
		slog.Warn("file log setup", "error", err)
	}

	// 2. Start Swiflow backend. MCP sync is deferred.
	shutdown := startSwiflowBackend(ctx, cfg)
	defer shutdown()
	slog.Info("desktop backend ready", "elapsed", time.Since(start).Round(time.Millisecond))

	// 3. Create wails3 application
	backendURL := fmt.Sprintf("http://%s", cfg.Addr())
	workspaceSvc := &Workspace{cfg: cfg}
	lightAppSvc := &LightAppService{}
	app := application.New(application.Options{
		Name:        "Swiflow",
		Description: "Self-hosted AI Agent Runtime",
		Services: []application.Service{
			application.NewService(workspaceSvc),
			application.NewService(lightAppSvc),
		},
		Assets: application.AssetOptions{
			Handler:    mustDesktopFrontendHandler(),
			Middleware: apiProxyMiddleware(backendURL),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	lightAppSvc.app = app
	app.SetIcon(emb.AppIconPNG)

	// 4. Create main window.
	// macOS: fusion title bar (traffic lights kept, content inset).
	// Windows: frameless so HeadTabBar replaces the native title bar.
	winOpts := application.WebviewWindowOptions{
		URL: "/", Title: "Swiflow", EnableFileDrop: true,
		Width: 1200, Height: 800, MinWidth: 800, MinHeight: 600,
		BackgroundColour: application.NewRGB(255, 255, 255),
	}
	switch runtime.GOOS {
	case "darwin":
		winOpts.Mac = application.MacWindow{
			TitleBar: application.MacTitleBarHiddenInset,
		}
	case "windows":
		winOpts.Frameless = true
	}
	win := app.Window.NewWithOptions(winOpts)
	bindWorkspaceFileDrop(win, cfg)
	slog.Info("desktop window created", "elapsed", time.Since(start).Round(time.Millisecond))

	// 5. Run
	if err := app.Run(); err != nil {
		slog.Error("app run", "error", err)
		os.Exit(1)
	}
}

// appDataDir returns the persistent desktop data directory.
// macOS: ~/Library/Application Support/Swiflow
func appDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Swiflow"), nil
	case "windows":
		if base := os.Getenv("APPDATA"); base != "" {
			return filepath.Join(base, "Swiflow"), nil
		}
		return filepath.Join(home, "AppData", "Roaming", "Swiflow"), nil
	default:
		if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
			return filepath.Join(base, "swiflow"), nil
		}
		return filepath.Join(home, ".config", "swiflow"), nil
	}
}

// ensureDesktopConfig finds an existing config or writes a default under Application Support.
// Launch order: SWIFLOW_CONFIG → ./config.json → ../config.json → AppSupport/config.json
// Data paths are always under AppSupport/data/{workspace,user-skills,swiflow.db}.
func ensureDesktopConfig() (string, error) {
	if p := os.Getenv("SWIFLOW_CONFIG"); p != "" {
		return p, nil
	}
	for _, p := range []string{"config.json", "../config.json"} {
		if abs, err := filepath.Abs(p); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs, nil
			}
		}
	}

	dir, err := appDataDir()
	if err != nil {
		return "", err
	}
	dataDir := filepath.Join(dir, "data")
	for _, sub := range []string{"workspace", "user-skills"} {
		if err := os.MkdirAll(filepath.Join(dataDir, sub), 0o755); err != nil {
			return "", err
		}
	}

	cfgPath := filepath.Join(dir, "config.json")
	if _, err := os.Stat(cfgPath); err == nil {
		return cfgPath, nil
	}

	cfg := map[string]any{
		"db_path": filepath.Join(dataDir, "swiflow.db"),
		"host":    "127.0.0.1", "port": 18765,
		"workspace_dir":    filepath.Join(dataDir, "workspace"),
		"user_skills_dir":  filepath.Join(dataDir, "user-skills"),
		"light_apps_dir":   filepath.Join(dataDir, "light-apps"),
		"allowed_origins":  []string{"*"},
		"max_history_msgs": 100, "tools": map[string]any{
			"exec_enabled":     true,
			"browser_enabled":  true,
			"browser_headless": true,
			"document_enabled": true,
		},
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(cfgPath, raw, 0o600); err != nil {
		return "", err
	}
	slog.Info("created desktop config", "path", cfgPath)
	return cfgPath, nil
}

// resolveDesktopPaths turns relative data paths into absolute paths under the config directory.
func resolveDesktopPaths(cfg config.Config, baseDir string) config.Config {
	abs := func(p string) string {
		if p == "" || filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(baseDir, p)
	}
	// Prefer rule-based layout under AppSupport when paths are still defaults/relative.
	cfg.DBPath = abs(cfg.DBPath)
	cfg.WorkspaceDir = abs(cfg.WorkspaceDir)
	cfg.UserSkillsDir = abs(cfg.UserSkillsDir)
	cfg.InitSkillsDir = abs(cfg.InitSkillsDir)
	return cfg
}

// startSwiflowBackend starts the Swiflow HTTP server in a goroutine and waits for it to be ready.
// MCP sync runs in the background so a hanging MCP server cannot block the first paint.
func startSwiflowBackend(ctx context.Context, cfg config.Config) func() {
	start := time.Now()
	for _, dir := range []string{filepath.Dir(cfg.DBPath), cfg.WorkspaceDir, cfg.UserSkillsDir, cfg.LightAppsDir} {
		if dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
	}

	st, err := appdb.MigrateAndOpen(ctx, cfg, true)
	if err != nil {
		slog.Error("open/migrate db", "error", err)
		os.Exit(1)
	}
	if err := appdb.EnsureDefaults(ctx, st); err != nil {
		slog.Error("seed", "error", err)
		os.Exit(1)
	}
	slog.Info("desktop db ready", "elapsed", time.Since(start).Round(time.Millisecond))

	skillsCat := skill.NewCatalog(cfg.InitSkillsDir, cfg.UserSkillsDir)

	toolsReg := tool.NewRegistry()
	tool.RegisterFS(toolsReg, tool.WorkspaceRoots{Base: cfg.WorkspaceDir})
	webOpts := &tool.WebOptions{
		SearchProvider: cfg.Tools.SearchProvider,
		SearchAPIKey:   cfg.Tools.SearchAPIKey,
		SearchBaseURL:  cfg.Tools.SearchBaseURL,
	}
	if webOpts.SearchProvider == "" {
		webOpts.SearchProvider = "duckduckgo"
	}
	server.LoadSearchSettings(ctx, st, webOpts)
	tool.RegisterWeb(toolsReg, tool.WorkspaceRoots{Base: cfg.WorkspaceDir}, webOpts)
	tool.RegisterExec(toolsReg, tool.WorkspaceRoots{Base: cfg.WorkspaceDir}, cfg.Tools.ExecEnabled)
	tool.RegisterSkill(toolsReg, skillsCat, st)
	docTimeout := time.Duration(cfg.Tools.DocumentTimeout) * time.Second
	if docTimeout <= 0 {
		docTimeout = 120 * time.Second
	}
	tool.RegisterContentExtract(toolsReg, tool.WorkspaceRoots{Base: cfg.WorkspaceDir}, st, tool.DocumentOptions{
		Enabled:   cfg.Tools.DocumentEnabled,
		BaseURL:   cfg.Tools.DocumentBaseURL,
		APIKey:    cfg.Tools.DocumentAPIKey,
		Model:     cfg.Tools.DocumentModel,
		Timeout:   docTimeout,
		Workspace: cfg.WorkspaceDir,
	})

	winBridge := window.NewBridge()
	tool.RegisterWindow(toolsReg, winBridge, tool.WorkspaceRoots{Base: cfg.WorkspaceDir})

	browserPool := browser.NewPool(cfg.Tools.BrowserHeadless)
	webOpts.BrowserPool = browserPool
	webOpts.BrowserEnabled = cfg.Tools.BrowserEnabled
	tool.RegisterBrowser(toolsReg, tool.WorkspaceRoots{Base: cfg.WorkspaceDir}, browserPool, tool.BrowserOptions{
		Enabled:  cfg.Tools.BrowserEnabled,
		Headless: cfg.Tools.BrowserHeadless,
	})

	// Apply persisted tool policy
	if pol, err := st.ListToolPolicy(ctx); err == nil {
		for _, p := range pol {
			toolsReg.SetEnabled(p.ToolName, p.Enabled)
		}
	}
	if !cfg.Tools.ExecEnabled {
		for _, name := range tool.RuntimeToolNames() {
			toolsReg.SetEnabled(name, false)
		}
	}
	if !cfg.Tools.BrowserEnabled {
		for _, name := range tool.BrowserToolNames() {
			toolsReg.SetEnabled(name, false)
		}
	}

	mcpMgr := mcpclient.NewManager(st, toolsReg)

	events := server.NewSessionHub()
	tracker := harness.NewTracker(events, st)

	runner := agent.NewRunner(agent.RunnerDeps{
		Store: st, Tools: toolsReg, Skills: skillsCat,
		Publish: tracker, Workspace: cfg.WorkspaceDir,

		MaxHistoryMessages: cfg.MaxHistoryMsgs,
		MaxConcurrentRuns:  cfg.MaxConcurrentRuns,
		ToolTimeoutSec:     cfg.ToolTimeoutSec,
		ToolTimeouts: map[string]time.Duration{
			tool.ToolContentExtract: docTimeout + 30*time.Second,
		},
		DisableThinking: cfg.DisableThinking,
	})

	cronSched := schedule.New(st, runner, tracker)
	tool.RegisterSchedule(toolsReg, st, cronSched)
	tool.RegisterExperience(toolsReg, st)
	tool.RegisterTodo(toolsReg, st)
	tool.RegisterDelegate(toolsReg, runner)
	tool.RegisterClarify(toolsReg, winBridge)
	if err := cronSched.Start(ctx); err != nil {
		slog.Warn("cron start", "error", err)
	}

	lightMgr := lightapp.NewManager(cfg.LightAppsDir)
	tool.RegisterLightAppTools(toolsReg, tool.LightAppRoots{Base: cfg.LightAppsDir}, st, lightMgr, winBridge)

	srv := server.New(cfg, st, runner, toolsReg, skillsCat, mcpMgr, cronSched, events, winBridge, webOpts, tracker, lightMgr)
	httpServer := &http.Server{
		Addr: cfg.Addr(), Handler: srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("swiflow backend listening", "addr", cfg.Addr())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()

	// Localhost must bypass system proxy (common Windows startup hang).
	healthClient := httputil.DirectClient(500 * time.Millisecond)
	ready := false
	for i := 0; i < 50; i++ {
		resp, err := healthClient.Get(fmt.Sprintf("http://%s/api/health", cfg.Addr()))
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			ready = true
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !ready {
		slog.Warn("swiflow backend health check timed out", "elapsed", time.Since(start).Round(time.Millisecond))
	} else {
		slog.Info("swiflow backend ready", "elapsed", time.Since(start).Round(time.Millisecond))
	}

	// Connect MCP servers after the UI can show; each connect may take up to 60s.
	go func() {
		mcpStart := time.Now()
		if err := mcpMgr.Sync(context.Background()); err != nil {
			slog.Warn("mcp initial sync", "error", err, "elapsed", time.Since(mcpStart).Round(time.Millisecond))
			return
		}
		slog.Info("mcp initial sync done", "elapsed", time.Since(mcpStart).Round(time.Millisecond))
	}()

	return func() {
		cronSched.Stop()
		mcpMgr.Close()
		browserPool.Close()
		httpServer.Shutdown(context.Background())
		st.Close()
	}
}

// mustDesktopFrontendHandler returns the desktop frontend as an http.Handler.
// In development, set FRONTEND_DEVSERVER_URL (e.g. http://localhost:5173) so
// AssetFileServerFS reverse-proxies to Vite HMR instead of embedded assets.
func mustDesktopFrontendHandler() http.Handler {
	dist, err := emb.GetFrontendDist()
	if err != nil {
		slog.Error("load desktop frontend", "error", err)
		os.Exit(1)
	}
	return application.AssetFileServerFS(dist)
}

// apiProxyMiddleware returns a wails3 Middleware that proxies /api/* requests
// to the Swiflow backend, letting all other requests fall through to the static file handler.
func apiProxyMiddleware(backendURL string) application.Middleware {
	// Localhost must bypass system proxy; Timeout 0 keeps SSE streams open.
	client := httputil.DirectClient(0)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}
			// Proxy to Swiflow backend
			proxyURL := backendURL + r.URL.Path
			if r.URL.RawQuery != "" {
				proxyURL += "?" + r.URL.RawQuery
			}
			proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, proxyURL, r.Body)
			if err != nil {
				http.Error(w, "proxy error", http.StatusBadGateway)
				return
			}
			// Copy headers
			for k, vs := range r.Header {
				for _, v := range vs {
					proxyReq.Header.Add(k, v)
				}
			}
			resp, err := client.Do(proxyReq)
			if err != nil {
				http.Error(w, "backend unreachable", http.StatusBadGateway)
				return
			}
			defer resp.Body.Close()
			// Copy response headers
			for k, vs := range resp.Header {
				for _, v := range vs {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(resp.StatusCode)
			// For SSE responses, flush as we write
			if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
				flusher, ok := w.(http.Flusher)
				if ok {
					buf := make([]byte, 4096)
					for {
						n, err := resp.Body.Read(buf)
						if n > 0 {
							w.Write(buf[:n])
							flusher.Flush()
						}
						if err != nil {
							break
						}
					}
					return
				}
			}
			io.Copy(w, resp.Body)
		})
	}
}
