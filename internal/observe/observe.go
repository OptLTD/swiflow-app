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
