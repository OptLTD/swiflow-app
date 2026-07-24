package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/OptLTD/swiflow/internal/agent"
	"github.com/OptLTD/swiflow/internal/llmclient"
	"github.com/OptLTD/swiflow/library/support"
)

func evalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "CLI eval harness (loads config.json like serve)",
	}
	cmd.AddCommand(evalAsyncImgExtractCmd())
	cmd.AddCommand(evalSubagentExitCmd())
	return cmd
}

func evalAsyncImgExtractCmd() *cobra.Command {
	var casesDir string
	var agentKey string
	var live bool
	var timeoutSec int
	cmd := &cobra.Command{
		Use:   "async-img-extract",
		Short: "Eval batch image extract (scripted soft-async or --live real LLM)",
		Long: `Boots the same runtime as serve (-c config.json), workspace = case image folder.

Default (scripted): drives soft-async OCR overlap + premature-stop re-ask.

--live: real chat model; after ~2 soft-async OCR on main with remaining >3s,
content_extract is denied and main should call subagent_spawn for the rest.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if live {
				return runEvalLiveDelegate(casesDir, agentKey, time.Duration(timeoutSec)*time.Second)
			}
			return runEvalAsyncImgExtract(casesDir, agentKey)
		},
	}
	cmd.Flags().StringVar(&casesDir, "cases", "data/testcases/async-img-extract", "directory of sample images")
	cmd.Flags().StringVar(&agentKey, "agent", "default", "agent key")
	cmd.Flags().BoolVar(&live, "live", false, "use real LLM from config/DB (test subagent_spawn routing)")
	cmd.Flags().IntVar(&timeoutSec, "timeout", 300, "max wall-clock seconds for --live (0 = no limit)")
	return cmd
}

func runEvalLiveDelegate(casesDir, agentKey string, wallTimeout time.Duration) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	absCases, err := filepath.Abs(casesDir)
	if err != nil {
		return err
	}
	files, err := listCaseImages(absCases)
	if err != nil {
		return err
	}
	if len(files) < 3 {
		return fmt.Errorf("live delegate eval needs ≥3 images, found %d in %s", len(files), absCases)
	}
	cfg.WorkspaceDir = absCases
	cfg.Tools.DocumentEnabled = true
	if cfg.Tools.DocumentTimeout < 180 {
		cfg.Tools.DocumentTimeout = 180 // glm-ocr layout_parsing can exceed 120s under load
	}

	ctx := context.Background()
	if wallTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, wallTimeout)
		defer cancel()
		fmt.Printf("live eval wall timeout: %s\n", wallTimeout.Round(time.Second))
	}
	rt, err := openRuntime(ctx, cfg)
	if err != nil {
		return err
	}
	defer rt.Close()

	if _, _, _, err := rt.Store.ProviderCreds(ctx, "default"); err != nil {
		return fmt.Errorf("default provider required for --live: %w", err)
	}
	if _, _, _, err := rt.Store.ProviderCreds(ctx, "vision"); err != nil {
		slog.Warn("vision provider missing; child OCR may fail", "error", err)
	}

	sessionID := "eval-live-" + support.NewID()[:8]
	// Prefer the production-shaped prompt (table + upload fence).
	userMsg := "把这些单据整理成表格\n\n[UPLOAD FILES START]\n"
	for _, f := range files {
		userMsg += "@/" + f + "\n"
	}
	userMsg += "[UPLOAD FILES END]\n"
	slog.Info("eval.live_delegate.start", "session", sessionID, "files", len(files), "agent", agentKey)
	fmt.Printf("user message (%d files):\n%s\n", len(files), userMsg)

	var (
		mu             sync.Mutex
		parentExtract  int
		parentDelegate int
		toolOrder      []string
		sawDone        bool
		lastAssistant  string
		runErr         string
		delegateArgs   []map[string]any
		delegateGoals  []string
	)

	t0 := time.Now()
	err = rt.Runner.Run(ctx, sessionID, agentKey, userMsg, func(ev agent.Event) {
		mu.Lock()
		defer mu.Unlock()
		switch ev.Type {
		case "tool_call":
			toolOrder = append(toolOrder, ev.Name)
			fmt.Printf("→ parent tool_call %s\n", ev.Name)
			if ev.Name == "content_extract" {
				parentExtract++
				if b, err := json.Marshal(ev.Arguments); err == nil {
					fmt.Printf("  args: %s\n", trimOneLine(string(b), 240))
				}
			}
			if ev.Name == "subagent_spawn" {
				parentDelegate++
				delegateArgs = append(delegateArgs, ev.Arguments)
				if g, _ := ev.Arguments["goal"].(string); g != "" {
					delegateGoals = append(delegateGoals, g)
				}
				if b, err := json.Marshal(ev.Arguments); err == nil {
					fmt.Printf("  args: %s\n", trimOneLine(string(b), 500))
				}
			}
		case "tool_result":
			fmt.Printf("  ← %s result (%d bytes)%s\n", ev.Name, len(ev.Result), map[bool]string{true: " ERROR", false: ""}[ev.IsError])
			if ev.Name == "subagent_spawn" && ev.Result != "" {
				fmt.Printf("  summary snippet: %s\n", trimOneLine(ev.Result, 280))
			}
			if ev.Result == agent.SoftAsyncPlaceholder {
				fmt.Printf("  … soft-async placeholder\n")
			}
		case "delta":
			lastAssistant += ev.Content
		case "done":
			sawDone = true
		case "error":
			runErr = ev.Error
			fmt.Printf("! error: %s\n", ev.Error)
		}
	})
	elapsed := time.Since(t0)

	childExtract := 0
	childSessions := 0
	sessions, _ := rt.Store.ListSessions(ctx)
	for _, s := range sessions {
		if !strings.HasPrefix(s.ID, "sub-"+sessionID) {
			continue
		}
		childSessions++
		msgs, merr := rt.Store.ListMessages(ctx, s.ID)
		if merr != nil {
			continue
		}
		n := 0
		var childTools []string
		for _, m := range msgs {
			if m.Role == "assistant" && m.ToolName != "" {
				childTools = append(childTools, m.ToolName)
			}
			if m.Role == "tool" && m.ToolName == "content_extract" {
				n++
			}
		}
		fmt.Printf("child session %s: content_extract tool msgs=%d tools=%v\n", s.ID, n, childTools)
		childExtract += n
	}

	xlsx, _ := filepath.Glob(filepath.Join(absCases, "*.xlsx"))
	fmt.Println()
	fmt.Println("=== live observations ===")
	fmt.Printf("elapsed:              %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("parent tool order:    %v\n", toolOrder)
	fmt.Printf("parent subagent_spawn: %d\n", parentDelegate)
	fmt.Printf("parent content_extract: %d\n", parentExtract)
	fmt.Printf("child sessions:       %d\n", childSessions)
	fmt.Printf("child content_extract msgs: %d\n", childExtract)
	fmt.Printf("workspace xlsx:       %v\n", xlsx)
	fmt.Printf("done:                 %v\n", sawDone)
	fmt.Printf("final:                %s\n", trimOneLine(lastAssistant, 240))

	var fails []string
	if parentDelegate == 0 {
		fails = append(fails, "main agent never called subagent_spawn after slow async handoff")
	}
	if parentExtract > 3 {
		fails = append(fails, fmt.Sprintf("main content_extract=%d; after handoff should stop stacking (want ≤3)", parentExtract))
	}
	if parentDelegate > 0 {
		delIdx := -1
		for i, n := range toolOrder {
			if n == "subagent_spawn" {
				delIdx = i
				break
			}
		}
		for i := delIdx + 1; i < len(toolOrder); i++ {
			if toolOrder[i] == "content_extract" {
				fails = append(fails, "parent called content_extract after subagent_spawn")
				break
			}
		}
	}
	// Full handoff quality: one batch goal covering remaining paths; no invented path/tools args.
	for i, args := range delegateArgs {
		if _, ok := args["path"]; ok {
			fails = append(fails, fmt.Sprintf("subagent_spawn[%d] invented path arg", i))
		}
		if _, ok := args["tools"]; ok {
			fails = append(fails, fmt.Sprintf("subagent_spawn[%d] passed tools whitelist (removed)", i))
		}
		goal, _ := args["goal"].(string)
		atCount := strings.Count(goal, "@/")
		// After cost probe (~2), remaining should be listed; require ≥3 @/ in goal for this 7-file case.
		if len(files) >= 5 && atCount < 3 {
			fails = append(fails, fmt.Sprintf("subagent_spawn[%d] goal has %d @/ paths (want batch ≥3); goal=%s", i, atCount, trimOneLine(goal, 160)))
		}
	}
	if parentDelegate > 1 {
		fails = append(fails, fmt.Sprintf("subagent_spawn called %d times; want one full handoff", parentDelegate))
	}
	if !sawDone && err == nil && runErr == "" {
		fails = append(fails, "missing done event")
	}
	if err != nil {
		fails = append(fails, "runner error: "+err.Error())
	}
	if runErr != "" {
		fails = append(fails, "stream error: "+runErr)
	}
	if ctx.Err() == context.DeadlineExceeded {
		fails = append(fails, fmt.Sprintf("wall timeout (%s) — increase --timeout if OCR batch still running", wallTimeout.Round(time.Second)))
	}

	if len(fails) > 0 {
		fmt.Println()
		fmt.Println("FAIL")
		for _, f := range fails {
			fmt.Println(" -", f)
		}
		return fmt.Errorf("live eval failed (%d checks)", len(fails))
	}
	fmt.Println()
	fmt.Println("PASS (main delegated batch; no parent extract after handoff)")
	return nil
}

func runEvalAsyncImgExtract(casesDir, agentKey string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	absCases, err := filepath.Abs(casesDir)
	if err != nil {
		return err
	}
	entries, err := listCaseImages(absCases)
	if err != nil {
		return err
	}
	if len(entries) < 4 {
		return fmt.Errorf("need ≥4 images in %s, found %d", absCases, len(entries))
	}
	// Use case dir as workspace so @/name resolves like production uploads.
	cfg.WorkspaceDir = absCases
	if !cfg.Tools.DocumentEnabled {
		cfg.Tools.DocumentEnabled = true
	}

	ctx := context.Background()
	rt, err := openRuntime(ctx, cfg)
	if err != nil {
		return err
	}
	defer rt.Close()

	// Confirm vision creds resolve (same path content_extract uses).
	if _, _, _, err := rt.Store.ProviderCreds(ctx, "vision"); err != nil {
		if _, _, _, err2 := rt.Store.ProviderCreds(ctx, "default"); err2 != nil {
			return fmt.Errorf("no vision/default provider creds in DB (config=%s): vision=%v default=%v", cfgFileOrDefault(), err, err2)
		}
		slog.Warn("vision provider missing; content_extract will fall back to default")
	}

	files := entries
	slog.Info("eval.async_img_extract.start", "cases", absCases, "files", len(files), "agent", agentKey)

	steps := make([]*llmclient.ChatResponse, 0, len(files)+3)
	for i, name := range files {
		steps = append(steps, &llmclient.ChatResponse{
			FinishReason: "tool_calls",
			ToolCalls: []llmclient.ToolCall{{
				ID:   fmt.Sprintf("ex-%d", i+1),
				Name: "content_extract",
				Arguments: map[string]any{
					"path":   "@/" + name,
					"prompt": "提取单据关键字段：车号/车牌、净重或装货量、单位、日期。用中文 key。",
				},
			}},
		})
	}
	steps = append(steps,
		&llmclient.ChatResponse{Content: "已整理部分文件，剩余仍在提取中，我将继续。", FinishReason: "stop"},
		&llmclient.ChatResponse{Content: "全部提取完成，可以整理成 Excel。", FinishReason: "stop"},
	)

	prov := &evalScriptedProvider{steps: steps}
	// Agent.txt_model is "default" in this env.
	rt.Runner.SetProvider("default", prov)

	sessionID := "eval-async-" + support.NewID()[:8]
	userMsg := buildUploadMessage(files)

	var (
		mu            sync.Mutex
		placeholders  int
		realResults   int
		toolStarts    []time.Time
		toolEnds      []time.Time
		llmCalls      int
		sawDone       bool
		lastAssistant string
	)

	t0 := time.Now()
	err = rt.Runner.Run(ctx, sessionID, agentKey, userMsg, func(ev agent.Event) {
		mu.Lock()
		defer mu.Unlock()
		switch ev.Type {
		case "tool_call":
			if ev.Name == "content_extract" {
				toolStarts = append(toolStarts, time.Now())
				fmt.Printf("→ tool_call content_extract #%d\n", len(toolStarts))
			}
		case "tool_result":
			if ev.Name != "content_extract" {
				return
			}
			if ev.Result == agent.SoftAsyncPlaceholder {
				placeholders++
				fmt.Printf("  … placeholder (soft-async)\n")
				return
			}
			realResults++
			toolEnds = append(toolEnds, time.Now())
			snippet := ev.Result
			if len(snippet) > 120 {
				snippet = snippet[:120] + "…"
			}
			fmt.Printf("  ✓ real result #%d (%d bytes): %s\n", realResults, len(ev.Result), snippet)
		case "delta":
			lastAssistant += ev.Content
		case "done":
			sawDone = true
		case "error":
			fmt.Printf("! error: %s\n", ev.Error)
		}
	})
	elapsed := time.Since(t0)
	llmCalls = prov.calls()

	fmt.Println()
	fmt.Println("=== observations ===")
	fmt.Printf("elapsed:        %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("llm_calls:      %d\n", llmCalls)
	fmt.Printf("placeholders:   %d\n", placeholders)
	fmt.Printf("real_results:   %d / %d files\n", realResults, len(files))
	fmt.Printf("tool_starts:    %d\n", len(toolStarts))
	fmt.Printf("done:           %v\n", sawDone)
	fmt.Printf("final_assistant: %s\n", trimOneLine(lastAssistant, 160))

	var fails []string
	if placeholders == 0 {
		fails = append(fails, "expected soft-async placeholder (≥1)")
	}
	if realResults < len(files) {
		fails = append(fails, fmt.Sprintf("expected %d real OCR results, got %d", len(files), realResults))
	}
	if !sawDone {
		fails = append(fails, "missing done event")
	}
	// Premature stop must trigger re-ask: scripted steps = N extracts + 2 stops → ≥ N+2 LLM calls
	wantMinCalls := len(files) + 2
	if llmCalls < wantMinCalls {
		fails = append(fails, fmt.Sprintf("expected ≥%d LLM calls (re-ask after premature stop), got %d", wantMinCalls, llmCalls))
	}
	if !strings.Contains(lastAssistant, "全部提取完成") {
		fails = append(fails, "final assistant text should be the post-await scripted reply, not the premature stop")
	}
	// Overlap: all starts should happen well before all ends; wall clock should beat naive serial.
	if len(toolStarts) >= 2 && len(toolEnds) >= 2 {
		startSpan := toolStarts[len(toolStarts)-1].Sub(toolStarts[0])
		fmt.Printf("start_span:     %s (time from first to last tool_call)\n", startSpan.Round(time.Millisecond))
		if startSpan > 30*time.Second {
			fails = append(fails, "tool starts looked serial (start_span too large)")
		}
	}
	if err != nil {
		fails = append(fails, "runner error: "+err.Error())
	}

	if len(fails) > 0 {
		fmt.Println()
		fmt.Println("FAIL")
		for _, f := range fails {
			fmt.Println(" -", f)
		}
		return fmt.Errorf("eval failed (%d checks)", len(fails))
	}
	fmt.Println()
	fmt.Println("PASS")
	return nil
}

func listCaseImages(dir string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		low := strings.ToLower(n)
		if strings.HasSuffix(low, ".png") || strings.HasSuffix(low, ".jpg") || strings.HasSuffix(low, ".jpeg") || strings.HasSuffix(low, ".webp") {
			out = append(out, n)
		}
	}
	return out, nil
}

func buildUploadMessage(files []string) string {
	var b strings.Builder
	b.WriteString("看下这几个,并整理成excel\n")
	b.WriteString("[UPLOAD FILES START]\n")
	for _, f := range files {
		b.WriteString("@/")
		b.WriteString(f)
		b.WriteByte('\n')
	}
	b.WriteString("[UPLOAD FILES END]\n")
	return b.String()
}

func cfgFileOrDefault() string {
	if cfgFile != "" {
		return cfgFile
	}
	if v := os.Getenv("SWIFLOW_CONFIG"); v != "" {
		return v
	}
	return "config.json"
}

func trimOneLine(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

// evalScriptedProvider returns canned ChatResponses in order (same idea as agent tests).
type evalScriptedProvider struct {
	mu    sync.Mutex
	steps []*llmclient.ChatResponse
	i     int
}

func (p *evalScriptedProvider) Name() string         { return "openai" }
func (p *evalScriptedProvider) DefaultModel() string { return "eval-script" }
func (p *evalScriptedProvider) calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.i
}
func (p *evalScriptedProvider) Chat(ctx context.Context, req llmclient.ChatRequest) (*llmclient.ChatResponse, error) {
	return p.next(ctx)
}
func (p *evalScriptedProvider) ChatStream(ctx context.Context, req llmclient.ChatRequest, onChunk func(llmclient.StreamChunk)) (*llmclient.ChatResponse, error) {
	resp, err := p.next(ctx)
	if err != nil {
		return nil, err
	}
	if resp.Content != "" && onChunk != nil {
		onChunk(llmclient.StreamChunk{Content: resp.Content})
	}
	return resp, nil
}
func (p *evalScriptedProvider) next(ctx context.Context) (*llmclient.ChatResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.i >= len(p.steps) {
		return &llmclient.ChatResponse{Content: "unexpected extra LLM call", FinishReason: "stop"}, nil
	}
	resp := p.steps[p.i]
	p.i++
	// Deep-copy so mutations are safe
	b, _ := json.Marshal(resp)
	var out llmclient.ChatResponse
	_ = json.Unmarshal(b, &out)
	return &out, nil
}
