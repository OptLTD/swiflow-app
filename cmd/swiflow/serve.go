package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/OptLTD/swiflow/internal/agent"
	"github.com/OptLTD/swiflow/internal/appdb"
	"github.com/OptLTD/swiflow/internal/harness"
	"github.com/OptLTD/swiflow/internal/lightapp"
	"github.com/OptLTD/swiflow/internal/mcpclient"
	"github.com/OptLTD/swiflow/internal/observe"
	"github.com/OptLTD/swiflow/internal/schedule"
	"github.com/OptLTD/swiflow/internal/server"
	"github.com/OptLTD/swiflow/internal/skill"
	"github.com/OptLTD/swiflow/internal/store/sqlstore"
	"github.com/OptLTD/swiflow/internal/tool"
	"github.com/OptLTD/swiflow/library/browser"
	"github.com/OptLTD/swiflow/library/window"
)

var autoMigrate bool

func serveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Swiflow HTTP server",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runServe()
		},
	}
	cmd.Flags().BoolVar(&autoMigrate, "migrate", true, "apply schema and upgrades before serving")
	return cmd
}

func runServe() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	if _, err := observe.SetupFileLog(cfg.DataDir(), level); err != nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
		slog.Warn("file log setup", "error", err)
	}

	dirs := []string{cfg.WorkspaceDir, cfg.UserSkillsDir, cfg.LightAppsDir}
	if cfg.DBDriver == "" || cfg.DBDriver == sqlstore.DialectSQLite || cfg.DBDriver == "sqlite3" {
		dirs = append(dirs, filepath.Dir(cfg.DBPath))
	}
	for _, dir := range dirs {
		if dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
	}

	st, err := appdb.MigrateAndOpen(context.Background(), cfg, autoMigrate)
	if err != nil {
		return err
	}
	defer st.Close()

	if autoMigrate {
		if err := appdb.EnsureDefaults(context.Background(), st); err != nil {
			return fmt.Errorf("seed: %w", err)
		}
	}

	skillsCat := skill.NewCatalog(cfg.InitSkillsDir, cfg.UserSkillsDir)

	toolsReg := tool.NewRegistry()
	tool.RegisterFS(toolsReg, tool.WorkspaceRoots{Base: cfg.WorkspaceDir})
	webOpts := &tool.WebOptions{
		SearchProvider: cfg.Tools.SearchProvider,
		SearchBaseURL:  cfg.Tools.SearchBaseURL,
		SearchAPIKey:   cfg.Tools.SearchAPIKey,
	}
	if webOpts.SearchProvider == "" {
		webOpts.SearchProvider = "duckduckgo"
	}
	server.LoadSearchSettings(context.Background(), st, webOpts)
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
	defer browserPool.Close()
	webOpts.BrowserPool = browserPool
	webOpts.BrowserEnabled = cfg.Tools.BrowserEnabled
	tool.RegisterBrowser(toolsReg, tool.WorkspaceRoots{Base: cfg.WorkspaceDir}, browserPool, tool.BrowserOptions{
		Enabled:  cfg.Tools.BrowserEnabled,
		Headless: cfg.Tools.BrowserHeadless,
	})

	// Apply persisted tool policy.
	if pol, err := st.ListToolPolicy(context.Background()); err == nil {
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
	if err := mcpMgr.Sync(context.Background()); err != nil {
		slog.Warn("mcp initial sync", "error", err)
	}
	defer mcpMgr.Close()

	events := server.NewSessionHub()
	tracker := harness.NewTracker(events, st)
	defer tracker.Close()

	runner := agent.NewRunner(agent.RunnerDeps{
		Store:              st,
		Tools:              toolsReg,
		Skills:             skillsCat,
		Workspace:          cfg.WorkspaceDir,
		MaxHistoryMessages: cfg.MaxHistoryMsgs,
		Publish:            tracker,
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
	if err := cronSched.Start(context.Background()); err != nil {
		slog.Warn("cron start", "error", err)
	}
	defer cronSched.Stop()

	lightMgr := lightapp.NewManager(cfg.LightAppsDir)
	defer lightMgr.StopAll()
	tool.RegisterLightAppTools(toolsReg, tool.LightAppRoots{Base: cfg.LightAppsDir}, st, lightMgr, winBridge)

	srv := server.New(cfg, st, runner, toolsReg, skillsCat, mcpMgr, cronSched, events, winBridge, webOpts, tracker, lightMgr)
	httpServer := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("swiflow listening", "addr", cfg.Addr())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(), 15*time.Second,
	)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}
