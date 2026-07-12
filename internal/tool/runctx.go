package tool

import "context"

type runCtxKey struct{}

// RunContext carries per-run metadata for tools (current session, agent).
type RunContext struct {
	SessionKey string
	AgentKey   string
}

// WithRunContext attaches run metadata to ctx for tool execution.
func WithRunContext(ctx context.Context, rc RunContext) context.Context {
	return context.WithValue(ctx, runCtxKey{}, rc)
}

// RunContextFrom reads run metadata injected by the agent runner.
func RunContextFrom(ctx context.Context) (RunContext, bool) {
	rc, ok := ctx.Value(runCtxKey{}).(RunContext)
	return rc, ok
}
