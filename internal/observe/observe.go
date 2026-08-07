// Package observe provides lightweight structured logging for agent runs.
package observe

import (
	"log/slog"
	"time"
)

// RoundStart logs the beginning of an LLM/tool round.
func RoundStart(session string, round int) {
	slog.Info("agent.round_start", "session", session, "round", round)
}

// RoundEnd logs round completion.
func RoundEnd(session string, round int, hadTools bool) {
	slog.Info("agent.round_end", "session", session, "round", round, "had_tools", hadTools)
}

// ToolStart logs the beginning of a tool invocation.
func ToolStart(session, name string) {
	slog.Info("agent.tool_start", "session", session, "tool", name)
}

// ToolEnd logs a tool invocation result.
func ToolEnd(session, name string, d time.Duration, err error) {
	if err != nil {
		slog.Warn("agent.tool", "session", session, "tool", name, "ms", d.Milliseconds(), "error", err.Error())
		return
	}
	slog.Info("agent.tool", "session", session, "tool", name, "ms", d.Milliseconds())
}

// BusyReject logs a per-session busy conflict (caller may still queue).
func BusyReject(session string) {
	slog.Info("agent.busy_reject", "session", session)
}

// ConcurrentReject logs a global concurrency gate rejection.
func ConcurrentReject(session string, inFlight, max int) {
	slog.Warn("agent.concurrent_reject", "session", session, "in_flight", inFlight, "max", max)
}

// Abort logs a session abort.
func Abort(session string) {
	slog.Info("agent.abort", "session", session)
}

// Queued logs a mid-run enqueue.
func Queued(session string, position int) {
	slog.Info("agent.queued", "session", session, "position", position)
}

// RunStart logs the beginning of an agent run.
func RunStart(session string, child bool, maxRounds int, chatModel string) {
	slog.Info("agent.run_start", "session", session, "child", child, "max_rounds", maxRounds, "chat_model", chatModel)
}

// RunEnd logs the end of an agent run.
func RunEnd(session, status string, round, maxRounds int, child bool) {
	slog.Info("agent.run_end", "session", session, "status", status, "round", round, "max_rounds", maxRounds, "child", child)
}

// ReflectEnter logs entry into a reflection checkpoint.
func ReflectEnter(session string, round int, trigger string) {
	slog.Info("agent.reflect_enter", "session", session, "round", round, "trigger", trigger)
}

// ReflectExit logs leaving a reflection checkpoint.
func ReflectExit(session string, round int, outcome string) {
	slog.Info("agent.reflect_exit", "session", session, "round", round, "outcome", outcome)
}

// ClaimRejected logs when a premature done is blocked for reflection.
func ClaimRejected(session, reason string) {
	slog.Info("agent.claim_rejected", "session", session, "reason", reason)
}

// CharterInjected logs Ways of working injection into the system prompt.
func CharterInjected(session, agent string, bytes int, emptySeed bool) {
	slog.Info("agent.charter_injected", "session", session, "agent", agent, "bytes", bytes, "empty_seed", emptySeed)
}

// CharterUpdated logs a working-charter change.
func CharterUpdated(session, agent, source string) {
	slog.Info("agent.charter_updated", "session", session, "agent", agent, "source", source)
}

// LLMStillWaiting logs periodic heartbeat while an LLM stream is in flight.
func LLMStillWaiting(session string, round int, model string, elapsed time.Duration, ctxRemain string) {
	slog.Info("agent.llm_still_waiting",
		"session", session, "round", round, "model", model,
		"elapsed", elapsed.Round(time.Second).String(), "ctx_remain", ctxRemain)
}

// SoftAsyncPlaceholder logs when a tool returns the soft-async placeholder.
func SoftAsyncPlaceholder(session, tool, callID string, pending int) {
	slog.Info("agent.soft_async_placeholder",
		"session", session, "tool", tool, "call_id", callID, "pending", pending)
}

// SoftAsyncDone logs when a background soft-async job finishes.
func SoftAsyncDone(session, tool, callID string, ms int64, pending int, isErr bool) {
	slog.Info("agent.soft_async_done",
		"session", session, "tool", tool, "call_id", callID,
		"ms", ms, "pending", pending, "error", isErr)
}

// AwaitAll logs blocking until pending soft-async jobs finish.
func AwaitAll(session, reason string, count int) {
	slog.Info("agent.await_all", "session", session, "reason", reason, "pending", count)
}

// AwaitSlot logs waiting for a parallel soft-async slot.
func AwaitSlot(session, tool string, inFlight, cap int) {
	slog.Info("agent.await_slot", "session", session, "tool", tool, "in_flight", inFlight, "cap", cap)
}

// Stall logs forced wrap-up due to repeated tools or async re-ask cap.
func Stall(session, reason string, round int) {
	slog.Warn("agent.stall", "session", session, "reason", reason, "round", round)
}

// SoftAsyncReask logs when the model tried to stop while async work was pending.
func SoftAsyncReask(session string, round, reask, pending int) {
	slog.Info("agent.soft_async_reask", "session", session, "round", round, "reask", reask, "pending", pending)
}

// DelegateStart logs sub-agent handoff.
func DelegateStart(parent, child string, maxRounds int) {
	slog.Info("agent.delegate_start", "parent", parent, "child", child, "max_rounds", maxRounds)
}

// DelegateEnd logs sub-agent completion.
func DelegateEnd(parent, child string, ms int64, err error) {
	if err != nil {
		slog.Warn("agent.delegate_end", "parent", parent, "child", child, "ms", ms, "error", err.Error())
		return
	}
	slog.Info("agent.delegate_end", "parent", parent, "child", child, "ms", ms)
}
