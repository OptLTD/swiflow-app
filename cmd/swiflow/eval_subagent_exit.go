package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/OptLTD/swiflow/internal/agent"
	"github.com/OptLTD/swiflow/library/support"
)

func evalSubagentExitCmd() *cobra.Command {
	var (
		casesDir   string
		agentKey   string
		fileN      int
		maxRounds  int
		timeoutSec int
	)
	cmd := &cobra.Command{
		Use:   "subagent-exit",
		Short: "Live test: run sub-agent alone (5 images) and verify it exits",
		Long: `Boots runtime from config.json, invokes RunOpts directly on a child session
(no parent agent, no delegate_task wrapper). Uses real LLM/OCR keys from DB.

Purpose: verify sub-agent exits within max_rounds (done or budget) without being
killed by an outer eval wall clock. Child gets its own context timeout.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runEvalSubagentExit(casesDir, agentKey, fileN, maxRounds, time.Duration(timeoutSec)*time.Second)
		},
	}
	cmd.Flags().StringVar(&casesDir, "cases", "data/testcases/async-img-extract", "directory of sample images")
	cmd.Flags().StringVar(&agentKey, "agent", "default", "agent key")
	cmd.Flags().IntVar(&fileN, "files", 5, "number of images to include (first N in cases dir)")
	cmd.Flags().IntVar(&maxRounds, "max-rounds", 10, "sub-agent round budget")
	cmd.Flags().IntVar(&timeoutSec, "timeout", 300, "child-only wall clock seconds (0 = no limit; safety net, not parent eval timeout)")
	return cmd
}

func runEvalSubagentExit(casesDir, agentKey string, fileN, maxRounds int, childWall time.Duration) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	absCases, err := filepath.Abs(casesDir)
	if err != nil {
		return err
	}
	all, err := listCaseImages(absCases)
	if err != nil {
		return err
	}
	if len(all) < fileN {
		return fmt.Errorf("need ≥%d images in %s, found %d", fileN, absCases, len(all))
	}
	files := all[:fileN]
	cfg.WorkspaceDir = absCases
	cfg.Tools.DocumentEnabled = true
	if cfg.Tools.DocumentTimeout < 180 {
		cfg.Tools.DocumentTimeout = 180
	}

	rt, err := openRuntime(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer rt.Close()

	if _, _, _, err := rt.Store.ProviderCreds(context.Background(), "default"); err != nil {
		return fmt.Errorf("default provider required: %w", err)
	}
	if _, _, _, err := rt.Store.ProviderCreds(context.Background(), "vision"); err != nil {
		slog.Warn("vision provider missing; OCR may fail", "error", err)
	}

	goal := buildSubagentExitGoal(files)
	sessionID := "eval-sub-" + support.NewID()[:8]

	fmt.Println("=== subagent-exit (child only) ===")
	fmt.Printf("session:     %s\n", sessionID)
	fmt.Printf("files:       %d\n", len(files))
	fmt.Printf("max_rounds:  %d\n", maxRounds)
	if childWall > 0 {
		fmt.Printf("child wall:  %s (independent context; not parent eval timeout)\n", childWall.Round(time.Second))
	}
	fmt.Printf("goal:\n%s\n", goal)

	childCtx := context.Background()
	var cancel context.CancelFunc
	if childWall > 0 {
		childCtx, cancel = context.WithTimeout(context.Background(), childWall)
		defer cancel()
	}

	var (
		toolOrder   []string
		sawDone     bool
		runErr      string
		lastDelta   string
	)
	t0 := time.Now()

	runErrVal := rt.Runner.RunOpts(childCtx, sessionID, agentKey, goal, func(ev agent.Event) {
		switch ev.Type {
		case "tool_call":
			toolOrder = append(toolOrder, ev.Name)
			fmt.Printf("→ tool_call %s\n", ev.Name)
		case "tool_result":
			tag := ""
			if ev.IsError {
				tag = " ERROR"
			}
			if ev.Result == agent.SoftAsyncPlaceholder {
				tag = " (placeholder)"
			}
			fmt.Printf("  ← %s (%d bytes)%s\n", ev.Name, len(ev.Result), tag)
		case "delta":
			lastDelta += ev.Content
		case "done":
			sawDone = true
			fmt.Println("  ✓ done")
		case "error":
			runErr = ev.Error
			fmt.Printf("  ! error: %s\n", ev.Error)
		}
	}, agent.RunOpts{
		MaxRounds: maxRounds,
		DenyTools: map[string]bool{"delegate_task": true, "clarify": true},
	})
	elapsed := time.Since(t0)

	msgs, _ := rt.Store.ListMessages(context.Background(), sessionID)
	assistantTurns := 0
	extractMsgs := 0
	var lastAssistant string
	for _, m := range msgs {
		if m.Role == "assistant" {
			assistantTurns++
			if strings.TrimSpace(m.Content) != "" {
				lastAssistant = m.Content
			}
		}
		if m.Role == "tool" && m.ToolName == "document_extract" {
			extractMsgs++
		}
	}
	if strings.TrimSpace(lastAssistant) == "" {
		lastAssistant = strings.TrimSpace(rt.Runner.LastAssistantContent(context.Background(), sessionID))
	}
	if strings.TrimSpace(lastAssistant) == "" {
		lastAssistant = strings.TrimSpace(lastDelta)
	}

	xlsx, _ := filepath.Glob(filepath.Join(absCases, "*.xlsx"))

	fmt.Println()
	fmt.Println("=== observations ===")
	fmt.Printf("elapsed:           %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("run error:         %v\n", runErrVal)
	fmt.Printf("stream error:      %q\n", runErr)
	fmt.Printf("done event:        %v\n", sawDone)
	fmt.Printf("tool order:        %v\n", toolOrder)
	fmt.Printf("assistant turns:   %d (LLM rounds ≈ this)\n", assistantTurns)
	fmt.Printf("document_extract:  %d tool msgs\n", extractMsgs)
	fmt.Printf("workspace xlsx:    %v\n", xlsx)
	fmt.Printf("final summary:     %s\n", trimOneLine(lastAssistant, 240))
	if childCtx.Err() != nil {
		fmt.Printf("child ctx:         %v\n", childCtx.Err())
	}

	var fails []string
	if runErrVal != nil {
		fails = append(fails, "RunOpts error: "+runErrVal.Error())
	}
	if runErr != "" {
		fails = append(fails, "stream error: "+runErr)
	}
	if !sawDone {
		fails = append(fails, "missing done event (sub-agent did not exit cleanly)")
	}
	if childCtx.Err() == context.DeadlineExceeded {
		fails = append(fails, fmt.Sprintf("child wall timeout (%s) before natural exit — increase --timeout or reduce max-rounds", childWall.Round(time.Second)))
	}
	if assistantTurns > maxRounds+2 {
		fails = append(fails, fmt.Sprintf("assistant turns=%d > max_rounds=%d+2", assistantTurns, maxRounds))
	}

	if len(fails) > 0 {
		fmt.Println()
		fmt.Println("FAIL")
		for _, f := range fails {
			fmt.Println(" -", f)
		}
		os.Exit(1)
	}
	fmt.Println()
	fmt.Println("PASS (sub-agent exited: done event, within budget)")
	return nil
}

func buildSubagentExitGoal(files []string) string {
	var b strings.Builder
	b.WriteString("Batch job: OCR every file below and write one Excel @/subagent-exit-test.xlsx.\n")
	b.WriteString("Use document_extract per image; sensible columns (单据编号, 车牌, 装货量, 卸货量, 日期).\n")
	b.WriteString("When the xlsx exists, reply with ONLY the @/ path and row count — then stop.\n\n")
	for _, f := range files {
		b.WriteString("@/")
		b.WriteString(f)
		b.WriteByte('\n')
	}
	return b.String()
}
