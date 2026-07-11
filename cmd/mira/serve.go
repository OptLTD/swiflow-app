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

	"mira/initial"
	"mira/internal/agent"
	"mira/internal/migrate"
	"mira/internal/server"
	"mira/internal/skill"
	"mira/internal/store/sqlite"
	"mira/internal/tool"
)

var autoMigrate bool

func serveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Mira HTTP server",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runServe()
		},
	}
	cmd.Flags().BoolVar(&autoMigrate, "migrate", false, "apply schema and upgrades before serving")
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
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	for _, dir := range []string{filepath.Dir(cfg.DBPath), cfg.WorkspaceDir, cfg.UserSkillsDir} {
		if dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
	}

	st, err := sqlite.Open(cfg.DBPath, cfg.EncryptionKey)
	if err != nil {
		return err
	}
	defer st.Close()

	if autoMigrate {
		upgrades, err := initial.UpgradesDir()
		if err != nil {
			return fmt.Errorf("upgrades fs: %w", err)
		}
		if err := migrate.Apply(context.Background(), st.DB(), initial.SchemaSQL, upgrades); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	skillsCat := skill.NewCatalog(cfg.InitSkillsDir, cfg.UserSkillsDir)

	toolsReg := tool.NewRegistry()
	tool.RegisterFS(toolsReg, tool.WorkspaceRoots{Base: cfg.WorkspaceDir})
	tool.RegisterWeb(toolsReg)
	tool.RegisterExec(toolsReg, tool.WorkspaceRoots{Base: cfg.WorkspaceDir}, cfg.Tools.ExecEnabled)
	tool.RegisterSkill(toolsReg, skillsCat)

	// Apply persisted tool policy.
	if pol, err := st.ListToolPolicy(context.Background()); err == nil {
		for _, p := range pol {
			toolsReg.SetEnabled(p.ToolName, p.Enabled)
		}
	}

	runner := agent.NewRunner(agent.RunnerDeps{
		Store:     st,
		Tools:     toolsReg,
		Skills:    skillsCat,
		Workspace: cfg.WorkspaceDir,
	})

	srv := server.New(cfg, st, runner, toolsReg, skillsCat)
	httpServer := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("mira listening", "addr", cfg.Addr())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}
