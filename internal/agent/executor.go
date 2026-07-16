// Package agent — bounded concurrent tool executor. Spec §6.7.
//
// One turn's tool calls run here: independent tools execute concurrently (capped
// by maxParallelTools) with a per-call timeout; serial tools (delegate_task /
// clarify / window_*) run sequentially. All calls are awaited before the loop
// starts the next LLM turn, and results are appended by the single loop writer.
package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/OptLTD/swiflow/internal/llmclient"
	"github.com/OptLTD/swiflow/internal/observe"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/tool"
	"github.com/OptLTD/swiflow/library/support"
)

const maxParallelTools = 4

type toolOutcome struct {
	tc     llmclient.ToolCall
	result string
	isErr  bool
}

// toolNeedsSerial reports tools that must run alone/sequentially: sub-agent runs
// and UI bridges must fully finish before the parent continues.
func toolNeedsSerial(name string) bool {
	return name == "clarify" || name == "delegate_task" || strings.HasPrefix(name, "window_")
}

// toolTimeout is the single source of truth for per-tool timeouts.
func (r *Runner) toolTimeout(name string) time.Duration {
	if r.deps.ToolTimeouts != nil {
		if d, ok := r.deps.ToolTimeouts[name]; ok && d > 0 {
			return d
		}
	}
	switch name {
	case "clarify", "delegate_task":
		// Interactive / nested runs are naturally long; bound them generously.
		return 15 * time.Minute
	}
	if r.deps.ToolTimeoutSec > 0 {
		return time.Duration(r.deps.ToolTimeoutSec) * time.Second
	}
	return defaultToolTimeout
}

// executeToolCalls runs one turn's tool calls concurrently (independent tools) or
// sequentially (when any is serial), emits tool_call/tool_result events, persists
// each tool result, and returns outcomes in call order.
func (r *Runner) executeToolCalls(
	runCtx, persistCtx context.Context,
	sessionID, agentKey string,
	calls []llmclient.ToolCall,
	publisher func(Event),
) []toolOutcome {
	for _, tc := range calls {
		publisher(Event{Type: "tool_call", ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments})
	}

	serial := false
	for _, tc := range calls {
		if toolNeedsSerial(tc.Name) {
			serial = true
			break
		}
	}

	outcomes := make([]toolOutcome, len(calls))
	runOne := func(i int, tc llmclient.ToolCall) {
		outcomes[i] = r.runToolOne(runCtx, sessionID, agentKey, tc, publisher)
	}

	if serial || len(calls) == 1 {
		for i, tc := range calls {
			runOne(i, tc)
		}
	} else {
		sem := make(chan struct{}, maxParallelTools)
		var wg sync.WaitGroup
		for i, tc := range calls {
			wg.Add(1)
			go func(i int, tc llmclient.ToolCall) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				runOne(i, tc)
			}(i, tc)
		}
		wg.Wait()
	}

	for _, o := range outcomes {
		publisher(Event{Type: "tool_result", ID: o.tc.ID, Name: o.tc.Name, Result: o.result, IsError: o.isErr})
		toolMsg := store.Message{
			ID:         support.NewID(),
			Role:       "tool",
			Content:    o.result,
			ToolCallId: o.tc.ID,
			ToolName:   o.tc.Name,
		}
		if _, err := r.deps.Store.AppendMessage(persistCtx, sessionID, toolMsg); err != nil {
			slog.Error("persist tool message", "error", err)
		}
	}
	return outcomes
}

// runToolOne executes a single tool call with its own timeout and returns a
// normalized outcome (truncated result, error captured as result text).
func (r *Runner) runToolOne(runCtx context.Context, sessionID, agentKey string, tc llmclient.ToolCall, publisher func(Event)) toolOutcome {
	t0 := time.Now()
	observe.ToolStart(sessionID, tc.Name)
	finish := func(result string, execErr error) toolOutcome {
		observe.ToolEnd(sessionID, tc.Name, time.Since(t0), execErr)
		if execErr != nil {
			return toolOutcome{tc: tc, result: truncateToolResult(formatToolError(result, execErr)), isErr: true}
		}
		return toolOutcome{tc: tc, result: truncateToolResult(result), isErr: false}
	}

	if err := runCtx.Err(); err != nil {
		return finish("", err)
	}
	if r.deps.Tools == nil {
		return finish("", fmt.Errorf("tools unavailable"))
	}
	if r.deps.Store != nil && !r.deps.Store.ToolEnabled(runCtx, tc.Name) {
		return finish("", fmt.Errorf("tool disabled: %s", tc.Name))
	}

	timeout := r.toolTimeout(tc.Name)
	tctx, cancel := context.WithTimeout(runCtx, timeout)
	defer cancel()
	// Live progress: serial tools (delegate_task) run on this goroutine, so it is
	// safe to publish tool_progress directly onto the foreground stream.
	emitProgress := func(p tool.ToolProgress) {
		if publisher == nil {
			return
		}
		publisher(Event{
			Type: "tool_progress",
			ID:   tc.ID, Content: p.Content,
			Name: tc.Name, Child: p.Child,
		})
	}
	result, err := r.deps.Tools.Execute(
		tool.WithRunContext(tctx, tool.RunContext{
			SessionID: sessionID, Agent: agentKey,
			ToolCallID: tc.ID, Emit: emitProgress,
		}),
		tc.Name, tc.Arguments,
	)
	return finish(result, err)
}
