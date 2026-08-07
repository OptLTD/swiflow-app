package tool

import "context"

type runCtxKey struct{}

// ToolProgress is a lightweight live-progress signal a long-running tool can emit
// mid-execution (e.g. subagent_spawn streaming its subagent's latest action). It is
// defined here (not in the agent package) to avoid an import cycle.
type ToolProgress struct {
	Child   string // subagent session key, if any
	Content string // latest action summary (tool name / text snippet)
}

// RunContext carries per-run metadata for tools (current session, agent).
type RunContext struct {
	SessionID string
	Agent     string
	// Tid is the tenant id for this run (empty = default / LocalMode).
	Tid string
	// Workspace is the tenant workspace root for sandboxed file tools.
	// When non-empty, tools prefer it over their registration-time Base.
	Workspace string
	// SkillsDir is the tenant user-skills root (drafts live under it).
	SkillsDir string
	// LightAppsDir is the tenant light-apps root.
	LightAppsDir string
	// ToolCallID is the id of the current tool call (set by the executor).
	ToolCallID string
	// Emit, when non-nil, forwards live progress to the foreground stream. Only
	// invoked by tools during a run (e.g. subagent_spawn progress forwarded to parent UI).
	// so it is safe to call without extra synchronization.
	Emit func(ToolProgress)
}

// WorkspaceBase returns RunContext.Workspace when set, otherwise fallback.
func WorkspaceBase(ctx context.Context, fallback string) string {
	if rc, ok := RunContextFrom(ctx); ok && rc.Workspace != "" {
		return rc.Workspace
	}
	return fallback
}

// SkillsBase returns RunContext.SkillsDir when set, otherwise fallback.
func SkillsBase(ctx context.Context, fallback string) string {
	if rc, ok := RunContextFrom(ctx); ok && rc.SkillsDir != "" {
		return rc.SkillsDir
	}
	return fallback
}

// LightAppsBase returns RunContext.LightAppsDir when set, otherwise fallback.
func LightAppsBase(ctx context.Context, fallback string) string {
	if rc, ok := RunContextFrom(ctx); ok && rc.LightAppsDir != "" {
		return rc.LightAppsDir
	}
	return fallback
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
