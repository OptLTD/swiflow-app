package tool

import (
	"context"
	"fmt"
	"strings"
)

type skillManageTool struct{ base *skillTools }

func (t *skillManageTool) Name() string { return "skill_manage" }
func (t *skillManageTool) Description() string {
	return "Create or patch user skills (SKILL.md). Use create for new skills; patch for small targeted edits (preferred over rewriting)."
}
func (t *skillManageTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"create", "patch"},
				"description": "create: new skill; patch: replace old_string with new_string in existing skill.",
			},
			"slug": map[string]any{
				"type":        "string",
				"description": "Skill identifier (lowercase, hyphens).",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Full SKILL.md for create (YAML front matter + body).",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "Exact text to replace (patch).",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "Replacement text (patch).",
			},
			"replace_all": map[string]any{
				"type":        "boolean",
				"description": "Replace every occurrence of old_string (patch). Default false.",
			},
		},
		"required": []string{"action", "slug"},
	}
}

func (t *skillManageTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	action := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", args["action"])))
	slug, _ := args["slug"].(string)
	cat := t.base.catalog(ctx)
	switch action {
	case "create":
		content, _ := args["content"].(string)
		if content == "" {
			return "", fmt.Errorf("content is required for create")
		}
		if err := cat.CreateSkill(slug, content); err != nil {
			return "", err
		}
		return fmt.Sprintf("skill %q created in user skills", slug), nil
	case "patch":
		oldStr, _ := args["old_string"].(string)
		newStr, _ := args["new_string"].(string)
		replaceAll, _ := args["replace_all"].(bool)
		if err := cat.PatchSkill(slug, oldStr, newStr, replaceAll); err != nil {
			return "", err
		}
		return fmt.Sprintf("skill %q patched", slug), nil
	default:
		return "", fmt.Errorf("unknown action %q (use create or patch)", action)
	}
}

type skillDraftTool struct{ base *skillTools }

func (t *skillDraftTool) Name() string { return "skill_draft" }
func (t *skillDraftTool) Description() string {
	return "Submit a skill draft for human confirmation (does not install until accepted in the UI)."
}
func (t *skillDraftTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"slug":    map[string]any{"type": "string"},
			"content": map[string]any{"type": "string", "description": "Full SKILL.md"},
			"note":    map[string]any{"type": "string", "description": "Why this skill / when to use"},
		},
		"required": []string{"slug", "content"},
	}
}

func (t *skillDraftTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	slug, _ := args["slug"].(string)
	content, _ := args["content"].(string)
	note, _ := args["note"].(string)
	d, err := t.base.catalog(ctx).SaveDraft(slug, content, note)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("draft %s saved for slug %q — awaiting human accept", d.ID, d.Slug), nil
}
