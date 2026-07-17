package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/OptLTD/swiflow/internal/agent"
	"github.com/OptLTD/swiflow/internal/appdb"
	"github.com/OptLTD/swiflow/internal/config"
	"github.com/OptLTD/swiflow/internal/harness"
	"github.com/OptLTD/swiflow/internal/observe"
	"github.com/OptLTD/swiflow/internal/skill"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/store/sqlstore"
	"github.com/OptLTD/swiflow/internal/tool"
)

// runtimeBundle is the same core wiring as `serve`, without HTTP/cron/MCP.
type runtimeBundle struct {
	Cfg    config.Config
	Store  store.Store
	Tools  *tool.Registry
	Runner *agent.Runner
	close  func()
}

func openRuntime(ctx context.Context, cfg config.Config) (*runtimeBundle, error) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	if _, err := observe.SetupFileLog(cfg.WorkspaceDir, level); err != nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
		slog.Warn("file log setup", "error", err)
	}

	dirs := []string{cfg.WorkspaceDir, cfg.UserSkillsDir}
	if cfg.DBDriver == "" || cfg.DBDriver == sqlstore.DialectSQLite || cfg.DBDriver == "sqlite3" {
		dirs = append(dirs, filepath.Dir(cfg.DBPath))
	}
	for _, dir := range dirs {
		if dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
	}

	st, err := appdb.MigrateAndOpen(ctx, cfg, true)
	if err != nil {
		return nil, err
	}
	if err := appdb.EnsureDefaults(ctx, st); err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("seed: %w", err)
	}

	skillsCat := skill.NewCatalog(cfg.InitSkillsDir, cfg.UserSkillsDir)
	docTimeout := time.Duration(cfg.Tools.DocumentTimeout) * time.Second
	if docTimeout <= 0 {
		docTimeout = 120 * time.Second
	}
	toolsReg := tool.NewRegistry()
	tool.RegisterFS(toolsReg, tool.WorkspaceRoots{Base: cfg.WorkspaceDir})
	tool.RegisterDocument(toolsReg, tool.WorkspaceRoots{Base: cfg.WorkspaceDir}, st, tool.DocumentOptions{
		Enabled:   cfg.Tools.DocumentEnabled,
		BaseURL:   cfg.Tools.DocumentBaseURL,
		APIKey:    cfg.Tools.DocumentAPIKey,
		Model:     cfg.Tools.DocumentModel,
		Timeout:   docTimeout,
		Workspace: cfg.WorkspaceDir,
	})
	tool.RegisterTodo(toolsReg, st)
	tool.RegisterExperience(toolsReg, st)
	tool.RegisterSkill(toolsReg, skillsCat, st)

	if pol, err := st.ListToolPolicy(ctx); err == nil {
		for _, p := range pol {
			toolsReg.SetEnabled(p.ToolName, p.Enabled)
		}
	}

	tracker := harness.NewTracker(nil, st)

	runner := agent.NewRunner(agent.RunnerDeps{
		Store:              st,
		Tools:              toolsReg,
		Skills:             skillsCat,
		Workspace:          cfg.WorkspaceDir,
		MaxHistoryMessages: cfg.MaxHistoryMsgs,
		Publish:            tracker,
		MaxConcurrentRuns:  cfg.MaxConcurrentRuns,
		ToolTimeoutSec:     cfg.ToolTimeoutSec,
		// Align the document_extract call timeout with the provider timeout
		// (+margin) so the provider's own HTTP timeout errors first.
		ToolTimeouts: map[string]time.Duration{
			tool.ToolDocumentExtract: docTimeout + 30*time.Second,
		},
		DisableThinking: cfg.DisableThinking,
	})
	tool.RegisterDelegate(toolsReg, runner)

	return &runtimeBundle{
		Cfg:    cfg,
		Store:  st,
		Tools:  toolsReg,
		Runner: runner,
		close: func() {
			tracker.Close()
			_ = st.Close()
		},
	}, nil
}

func (b *runtimeBundle) Close() {
	if b != nil && b.close != nil {
		b.close()
	}
}
