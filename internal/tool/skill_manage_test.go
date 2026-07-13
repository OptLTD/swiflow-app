package tool_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OptLTD/swiflow/internal/skill"
	"github.com/OptLTD/swiflow/internal/tool"
)

type noopSkillStore struct{}

func (noopSkillStore) DisabledSkills(context.Context) ([]string, error) { return nil, nil }

func TestSkillManageCreate(t *testing.T) {
	dir := t.TempDir()
	cat := skill.NewCatalog("", dir)
	reg := tool.NewRegistry()
	tool.RegisterSkill(reg, cat, noopSkillStore{})

	tl, ok := reg.Get("skill_manage")
	if !ok {
		t.Fatal("skill_manage not registered")
	}
	content := `---
slug: agent-saved
name: Agent Saved
description: saved by agent
---

Body here.
`
	out, err := tl.Execute(context.Background(), map[string]any{
		"action":  "create",
		"slug":    "agent-saved",
		"content": content,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Fatal("expected confirmation")
	}
	data, err := os.ReadFile(filepath.Join(dir, "agent-saved", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Body here.") {
		t.Fatalf("file: %s", data)
	}
}

func TestSkillManagePatch(t *testing.T) {
	dir := t.TempDir()
	cat := skill.NewCatalog("", dir)
	_ = cat.CreateSkill("fix-me", `---
slug: fix-me
name: Fix
description: x
---

old text
`)
	reg := tool.NewRegistry()
	tool.RegisterSkill(reg, cat, noopSkillStore{})
	tl, _ := reg.Get("skill_manage")

	_, err := tl.Execute(context.Background(), map[string]any{
		"action":     "patch",
		"slug":       "fix-me",
		"old_string": "old text",
		"new_string": "new text",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := cat.ReadSkillMD("fix-me")
	if !strings.Contains(got, "new text") {
		t.Fatalf("got %q", got)
	}
}
