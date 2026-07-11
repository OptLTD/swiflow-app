// Package tool defines the Tool interface, a Registry, and built-in tools.
// Spec §6.5, §8.
package tool

import (
	"context"
	"fmt"
	"log/slog"

	"mira/internal/llm"
)

// Tool is a capability an agent may invoke.
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any // JSON Schema
	Execute(ctx context.Context, args map[string]any) (string, error)
}

// Registry holds tool instances and their enable state.
type Registry struct {
	tools    map[string]Tool
	disabled map[string]bool
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}, disabled: map[string]bool{}}
}

// Register adds a tool.
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Names returns all registered tool names.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.tools))
	for n := range r.tools {
		out = append(out, n)
	}
	return out
}

// SetEnabled enables/disables a tool at runtime (mirrors persisted policy).
func (r *Registry) SetEnabled(name string, enabled bool) {
	if !enabled {
		r.disabled[name] = true
	} else {
		delete(r.disabled, name)
	}
}

// IsEnabled reports whether a tool is enabled.
func (r *Registry) IsEnabled(name string) bool {
	return !r.disabled[name]
}

// Definitions returns tool defs for all enabled tools, for advertising to the LLM.
func (r *Registry) Definitions() []llm.ToolDef {
	out := make([]llm.ToolDef, 0, len(r.tools))
	for _, t := range r.tools {
		if r.disabled[t.Name()] {
			continue
		}
		out = append(out, llm.ToolDef{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}
	return out
}

// All returns all registered tools with their enable state (for the admin API).
func (r *Registry) All() []Info {
	out := make([]Info, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, Info{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
			Enabled:     !r.disabled[t.Name()],
		})
	}
	return out
}

// Info describes a tool for API responses.
type Info struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
	Enabled     bool           `json:"enabled"`
}

// Execute runs a tool by name with panic recovery. A panic becomes a tool-error
// string, never a run crash. A disabled or unknown tool returns an error.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (result string, err error) {
	t, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	if r.disabled[name] {
		return "", fmt.Errorf("tool disabled: %s", name)
	}
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("tool panic", "tool", name, "panic", rec)
			result = fmt.Sprintf("error: panic: %v", rec)
			err = nil
		}
	}()
	return t.Execute(ctx, args)
}
