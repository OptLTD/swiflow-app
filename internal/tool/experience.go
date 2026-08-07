package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/OptLTD/swiflow/internal/store"
)

type experienceTools struct {
	st store.Store
}

// RegisterExperience registers experience_write / experience_list / experience_use.
func RegisterExperience(r *Registry, st store.Store) {
	base := &experienceTools{st: st}
	r.Register(&experienceWriteTool{base: base})
	r.Register(&experienceListTool{base: base})
	r.Register(&experienceUseTool{base: base})
}

type experienceWriteTool struct{ base *experienceTools }

func (t *experienceWriteTool) Name() string { return "experience_write" }
func (t *experienceWriteTool) Description() string {
	return "Record a reusable handling rule (logic you would apply again in another task). " +
		"Write a pitfall, decision rule, or working recipe — not a task diary or changelog. " +
		"Save only when the lesson is general enough to reuse; skip one-off details. " +
		"You may write multiple distinct lessons from one task if each is independently reusable. " +
		"If this work reused past lessons, pass their ids in used_ids to raise their weight."
}
func (t *experienceWriteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"summary": map[string]any{
				"type": "string",
				"description": "One short sentence (≤200 chars) of reusable handling logic. " +
					"Good: \"OCR often swaps 皮重/毛重 on weigh tickets; prefer filename when values are physically impossible.\" " +
					"Bad: \"Built an Excel for today's 7 tickets\" (task report, not reusable).",
			},
			"outcome": map[string]any{
				"type":        "string",
				"enum":        []string{"success", "partial", "failure", "unknown"},
				"description": "Result of the task.",
			},
			"tags": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "1-3 lowercase English topic tags, e.g. [\"excel\", \"ocr\"].",
			},
			"used_ids": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional ids from experience_list that you actually applied this turn; bumps their weight.",
			},
		},
		"required": []string{"summary"},
	}
}

func (t *experienceWriteTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	summary, _ := args["summary"].(string)
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return "", fmt.Errorf("summary is required")
	}
	// Keep memory dense: reject task-report style dumps.
	runes := []rune(summary)
	if len(runes) > 240 {
		summary = string(runes[:237]) + "…"
	}
	outcome, _ := args["outcome"].(string)
	if outcome == "" {
		outcome = "unknown"
	}
	var tags []string
	if rawTags, ok := args["tags"].([]any); ok {
		for _, tag := range rawTags {
			if s, ok := tag.(string); ok && s != "" {
				tags = append(tags, strings.ToLower(strings.TrimSpace(s)))
				if len(tags) >= 3 {
					break
				}
			}
		}
	}
	usedIDs := parseStringIDs(args["used_ids"])

	rc, _ := RunContextFrom(ctx)
	agentKey := rc.Agent
	if agentKey == "" {
		agentKey = "default"
	}

	// Skip exact duplicate writes (model often double-fires experience_write).
	if recent, err := t.base.st.ListExperiences(ctx, agentKey, 8); err == nil {
		for _, ex := range recent {
			if ex.Summary == summary {
				bumped := bumpUsed(ctx, t.base.st, usedIDs)
				msg := fmt.Sprintf("experience already recorded (id %s); skipped duplicate", ex.ID)
				if bumped > 0 {
					msg += fmt.Sprintf("; bumped %d prior experience(s)", bumped)
				}
				return msg, nil
			}
			// Same session + near-identical long dump → also skip.
			if rc.SessionID != "" && ex.Sid == rc.SessionID && similarExperience(ex.Summary, summary) {
				bumped := bumpUsed(ctx, t.base.st, usedIDs)
				msg := fmt.Sprintf("experience already recorded (id %s); skipped duplicate", ex.ID)
				if bumped > 0 {
					msg += fmt.Sprintf("; bumped %d prior experience(s)", bumped)
				}
				return msg, nil
			}
		}
	}

	e := &store.Experience{
		Sid:     rc.SessionID,
		Agent:   agentKey,
		Summary: summary,
		Outcome: outcome,
		Tags:    tags,
		Weight:  1,
	}
	if err := t.base.st.CreateExperience(ctx, e); err != nil {
		return "", err
	}
	bumped := bumpUsed(ctx, t.base.st, usedIDs)
	msg := fmt.Sprintf("experience recorded (id %s, weight %d)", e.ID, e.Weight)
	if bumped > 0 {
		msg += fmt.Sprintf("; bumped %d prior experience(s)", bumped)
	}
	return msg, nil
}

// similarExperience is a cheap same-session dedupe for near-identical dumps.
func similarExperience(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	// Same prefix usually means the model pasted the same report twice.
	ar, br := []rune(a), []rune(b)
	n := 80
	if len(ar) < n || len(br) < n {
		return false
	}
	return string(ar[:n]) == string(br[:n])
}

func parseStringIDs(raw any) []string {
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	out := make([]string, 0, len(arr))
	seen := map[string]struct{}{}
	for _, v := range arr {
		s, _ := v.(string)
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
		if len(out) >= 8 {
			break
		}
	}
	return out
}

func bumpUsed(ctx context.Context, st store.Store, ids []string) int {
	n := 0
	for _, id := range ids {
		if _, err := st.BumpExperienceWeight(ctx, id, 1); err == nil {
			n++
		}
	}
	return n
}

type experienceListTool struct{ base *experienceTools }

func (t *experienceListTool) Name() string { return "experience_list" }
func (t *experienceListTool) Description() string {
	return "List this agent's experiences, highest weight first (weight rises when reused via experience_use / used_ids). " +
		"Check before complex tasks; prefer high-weight lessons."
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

type experienceUseTool struct{ base *experienceTools }

func (t *experienceUseTool) Name() string { return "experience_use" }
func (t *experienceUseTool) Description() string {
	return "Mark past experience(s) as useful in the current task (raises weight so they rank higher next time). " +
		"Call after you actually apply a lesson from experience_list."
}
func (t *experienceUseTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"ids": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "One or more experience ids to bump.",
			},
			"id": map[string]any{
				"type":        "string",
				"description": "Single experience id (alias for ids:[id]).",
			},
		},
	}
}

func (t *experienceUseTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	var raw []any
	if arr, ok := args["ids"].([]any); ok {
		raw = append(raw, arr...)
	}
	if id, _ := args["id"].(string); strings.TrimSpace(id) != "" {
		raw = append(raw, strings.TrimSpace(id))
	}
	ids := parseStringIDs(raw)
	if len(ids) == 0 {
		return "", fmt.Errorf("id or ids required")
	}
	type hit struct {
		ID     string `json:"id"`
		Weight int    `json:"weight"`
	}
	hits := make([]hit, 0, len(ids))
	for _, id := range ids {
		ex, err := t.base.st.BumpExperienceWeight(ctx, id, 1)
		if err != nil {
			continue
		}
		hits = append(hits, hit{ID: ex.ID, Weight: ex.Weight})
	}
	if len(hits) == 0 {
		return "", fmt.Errorf("no matching experiences")
	}
	b, _ := json.Marshal(map[string]any{"bumped": hits})
	return string(b), nil
}
