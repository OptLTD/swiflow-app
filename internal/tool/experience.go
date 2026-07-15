package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/OptLTD/swiflow/internal/store"
)

type experienceTools struct {
	st store.Store
}

// RegisterExperience registers experience_write and experience_list tools.
func RegisterExperience(r *Registry, st store.Store) {
	base := &experienceTools{st: st}
	r.Register(&experienceWriteTool{base: base})
	r.Register(&experienceListTool{base: base})
}

type experienceWriteTool struct{ base *experienceTools }

func (t *experienceWriteTool) Name() string { return "experience_write" }
func (t *experienceWriteTool) Description() string {
	return "Record a learning experience after completing or failing a significant task. Builds persistent memory across sessions."
}
func (t *experienceWriteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"summary": map[string]any{
				"type":        "string",
				"description": "One or two sentences: what was done, learned, or why it failed.",
			},
			"outcome": map[string]any{
				"type":        "string",
				"enum":        []string{"success", "partial", "failure", "unknown"},
				"description": "Result of the task.",
			},
			"tags": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "1-3 topic tags, e.g. [\"excel\", \"data-analysis\"].",
			},
		},
		"required": []string{"summary"},
	}
}

func (t *experienceWriteTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	summary, _ := args["summary"].(string)
	if summary == "" {
		return "", fmt.Errorf("summary is required")
	}
	outcome, _ := args["outcome"].(string)
	if outcome == "" {
		outcome = "unknown"
	}
	var tags []string
	if rawTags, ok := args["tags"].([]any); ok {
		for _, tag := range rawTags {
			if s, ok := tag.(string); ok && s != "" {
				tags = append(tags, s)
			}
		}
	}

	rc, _ := RunContextFrom(ctx)
	agentKey := rc.Agent
	if agentKey == "" {
		agentKey = "default"
	}
	e := &store.Experience{
		Sid: rc.SessionID,
		Agent:  agentKey,
		Summary:   summary,
		Outcome:   outcome,
		Tags:      tags,
	}
	if err := t.base.st.CreateExperience(ctx, e); err != nil {
		return "", err
	}
	return fmt.Sprintf("experience recorded (id %s)", e.ID), nil
}

type experienceListTool struct{ base *experienceTools }

func (t *experienceListTool) Name() string { return "experience_list" }
func (t *experienceListTool) Description() string {
	return "List recent experiences recorded by this agent. Check before complex tasks to avoid repeating past mistakes."
}
func (t *experienceListTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"limit": map[string]any{
				"type":        "integer",
				"description": "Max results to return (1-50, default 10).",
			},
		},
	}
}

func (t *experienceListTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	limit := 10
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}
	if limit > 50 {
		limit = 50
	}

	rc, _ := RunContextFrom(ctx)
	agentKey := rc.Agent
	if agentKey == "" {
		agentKey = "default"
	}

	list, err := t.base.st.ListExperiences(ctx, agentKey, limit)
	if err != nil {
		return "", err
	}
	if list == nil {
		list = []store.Experience{}
	}
	b, _ := json.Marshal(list)
	return string(b), nil
}
