// Package main is the Swiflow desktop application entrypoint using wails3.
// It starts the Swiflow backend in-process and opens a native window with the Vue UI.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	emb "github.com/OptLTD/swiflow/embed"
	"github.com/OptLTD/swiflow/internal/agent"
	"github.com/OptLTD/swiflow/internal/browser"
	"github.com/OptLTD/swiflow/internal/config"
	"github.com/OptLTD/swiflow/internal/mcpclient"
	"github.com/OptLTD/swiflow/internal/migrate"
	"github.com/OptLTD/swiflow/internal/schedule"
	"github.com/OptLTD/swiflow/internal/seed"
	"github.com/OptLTD/swiflow/internal/server"
	"github.com/OptLTD/swiflow/internal/sesshub"
	"github.com/OptLTD/swiflow/internal/skill"
	"github.com/OptLTD/swiflow/internal/store/sqlite"
	"github.com/OptLTD/swiflow/internal/tool"
)

func main() {
	ctx := context.Background()

	// 1. Load Swiflow config
	cfgPath := findConfig()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	// 2. Start Swiflow backend in background (skip auth in desktop mode)
	cfg.SkipAuth = true
	shutdown := startSwiflowBackend(ctx, cfg)
	defer shutdown()

	// 3. Create wails3 application
	backendURL := fmt.Sprintf("http://%s", cfg.Addr())
	app := application.New(application.Options{
		Name:        "Swiflow",
		Description: "Self-hosted AI Agent Runtime",
		Services:    []application.Service{},
		Assets: application.AssetOptions{
			Handler:    mustDesktopFrontendHandler(),
			Middleware: apiProxyMiddleware(backendURL),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// 4. Create main window (macOS fusion title bar: no system header, traffic lights kept)
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		URL: "/", Title: "Swiflow", Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHiddenInset,
		},
		Width: 1200, Height: 800, MinWidth: 800, MinHeight: 600,
		BackgroundColour: application.NewRGB(255, 255, 255),
	})

	// 5. Run
	if err := app.Run(); err != nil {
		slog.Error("app run", "error", err)
		os.Exit(1)
	}
}

// findConfig locates config.json: check SWIFLOW_CONFIG env, then current dir, then parent dir.
func findConfig() string {
	if p := os.Getenv("SWIFLOW_CONFIG"); p != "" {
		return p
	}
	if _, err := os.Stat("config.json"); err == nil {
		return "config.json"
	}
	if _, err := os.Stat("../config.json"); err == nil {
		return "../config.json"
	}
	return "config.json"
}

// startSwiflowBackend starts the Swiflow HTTP server in a goroutine and waits for it to be ready.
func startSwiflowBackend(ctx context.Context, cfg config.Config) func() {
	for _, dir := range []string{filepath.Dir(cfg.DBPath), cfg.WorkspaceDir, cfg.UserSkillsDir} {
		if dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
	}

	st, err := sqlite.Open(cfg.DBPath, cfg.EncryptionKey)
	if err != nil {
		slog.Error("open db", "error", err)
		os.Exit(1)
	}

	// Auto-migrate
	upgrades, err := emb.UpgradesDir()
	if err != nil {
		slog.Error("upgrades fs", "error", err)
		os.Exit(1)
	}
	if err := migrate.Apply(ctx, st.DB(), emb.SchemaSQL, upgrades); err != nil {
		slog.Error("migrate", "error", err)
		os.Exit(1)
	}
	if err := seed.EnsureDefaults(ctx, st); err != nil {
		slog.Error("seed", "error", err)
		os.Exit(1)
	}

	skillsCat := skill.NewCatalog(cfg.InitSkillsDir, cfg.UserSkillsDir)

	toolsReg := tool.NewRegistry()
	tool.RegisterFS(toolsReg, tool.WorkspaceRoots{Base: cfg.WorkspaceDir})
	tool.RegisterWeb(toolsReg)
	tool.RegisterExec(toolsReg, tool.WorkspaceRoots{Base: cfg.WorkspaceDir}, cfg.Tools.ExecEnabled)
	tool.RegisterSkill(toolsReg, skillsCat, st)

	browserPool := browser.NewPool(cfg.Tools.BrowserHeadless)
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
	if err := mcpMgr.Sync(ctx); err != nil {
		slog.Warn("mcp initial sync", "error", err)
	}

	runner := agent.NewRunner(agent.RunnerDeps{
		Store:              st,
		Tools:              toolsReg,
		Skills:             skillsCat,
		Workspace:          cfg.WorkspaceDir,
		MaxHistoryMessages: cfg.MaxHistoryMsgs,
	})

	events := sesshub.New()

	cronSched := schedule.New(st, runner, events)
	tool.RegisterSchedule(toolsReg, st, cronSched)
	if err := cronSched.Start(ctx); err != nil {
		slog.Warn("cron start", "error", err)
	}

	srv := server.New(cfg, st, runner, toolsReg, skillsCat, mcpMgr, cronSched, events)
	httpServer := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("swiflow backend listening", "addr", cfg.Addr())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()

	// Wait for backend to be ready
	for i := 0; i < 50; i++ {
		resp, err := http.Get(fmt.Sprintf("http://%s/api/health", cfg.Addr()))
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			slog.Info("swiflow backend ready")
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}

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
			resp, err := http.DefaultClient.Do(proxyReq)
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
