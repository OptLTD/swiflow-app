// Skill tools: use and search. Spec §8.
package tool

import (
	"context"
	"fmt"
	"strings"

	"mira/internal/skill"
)

type skillCatalog interface {
	Discover(context.Context) []skill.Skill
}

type useSkillTool struct{ cat skillCatalog }

func (t *useSkillTool) Name() string        { return "skill_use" }
func (t *useSkillTool) Description() string { return "Load and apply a skill's instructions by slug." }
func (t *useSkillTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"slug":  map[string]any{"type": "string"},
			"input": map[string]any{"type": "string"},
		},
		"required": []string{"slug"},
	}
}
func (t *useSkillTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	slug, _ := args["slug"].(string)
	input, _ := args["input"].(string)
	for _, s := range t.cat.Discover(ctx) {
		if s.Slug == slug {
			if s.Body == "" {
				return "", fmt.Errorf("skill %s has no body", slug)
			}
			out := s.Body
			if input != "" {
				out += "\n\n## Input\n\n" + input
			}
			return out, nil
		}
	}
	return "", fmt.Errorf("skill not found: %s", slug)
}

type skillSearchTool struct{ cat skillCatalog }

func (t *skillSearchTool) Name() string        { return "skill_search" }
func (t *skillSearchTool) Description() string { return "Search available skills by keyword." }
func (t *skillSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
		},
		"required": []string{"query"},
	}
}
func (t *skillSearchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	q := strings.ToLower(fmt.Sprintf("%v", args["query"]))
	var b strings.Builder
	for _, s := range t.cat.Discover(ctx) {
		hay := strings.ToLower(s.Name + " " + s.Description + " " + s.Slug)
		if strings.Contains(hay, q) {
			fmt.Fprintf(&b, "- %s: %s — %s\n", s.Slug, s.Name, s.Description)
		}
	}
	if b.Len() == 0 {
		return "no skills matched", nil
	}
	return b.String(), nil
}

// RegisterSkill registers the skill tools.
func RegisterSkill(r *Registry, cat skillCatalog) {
	r.Register(&useSkillTool{cat: cat})
	r.Register(&skillSearchTool{cat: cat})
}
