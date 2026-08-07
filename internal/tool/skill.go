// Skill tools: use and search. Spec §8.
package tool

import (
	"context"
	"fmt"
	"strings"

	"github.com/OptLTD/swiflow/internal/skill"
)

type skillStore interface {
	DisabledSkills(context.Context) ([]string, error)
}

type skillTools struct {
	cat *skill.Catalog
	st  skillStore
}

func (t *skillTools) catalog(ctx context.Context) *skill.Catalog {
	dir := SkillsBase(ctx, "")
	if dir != "" {
		return t.cat.ForUserDir(dir)
	}
	return t.cat
}

func (t *skillTools) enabledSkills(ctx context.Context) []skill.Skill {
	disabled := map[string]bool{}
	if list, err := t.st.DisabledSkills(ctx); err == nil {
		for _, slug := range list {
			disabled[slug] = true
		}
	}
	var out []skill.Skill
	for _, s := range t.catalog(ctx).Discover(ctx) {
		if !disabled[s.Slug] {
			out = append(out, s)
		}
	}
	return out
}

type useSkillTool struct{ base *skillTools }

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
	for _, s := range t.base.enabledSkills(ctx) {
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

type skillSearchTool struct{ base *skillTools }

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
	for _, s := range t.base.enabledSkills(ctx) {
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

// RegisterSkill registers skill discovery, use, search, manage, and draft tools.
func RegisterSkill(r *Registry, cat *skill.Catalog, st skillStore) {
	base := &skillTools{cat: cat, st: st}
	r.Register(&useSkillTool{base: base})
	r.Register(&skillSearchTool{base: base})
	r.Register(&skillManageTool{base: base})
	r.Register(&skillDraftTool{base: base})
}
