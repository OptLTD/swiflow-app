package skill_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mira/internal/skill"
)

func TestCreateSkill(t *testing.T) {
	dir := t.TempDir()
	cat := skill.NewCatalog("", dir)

	content := `---
slug: my-workflow
name: My Workflow
description: Demo skill
---

# Steps

1. Do thing
`
	if err := cat.CreateSkill("my-workflow", content); err != nil {
		t.Fatal(err)
	}
	got, err := cat.ReadSkillMD("my-workflow")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "# Steps") {
		t.Fatalf("read back: %q", got)
	}
	if err := cat.CreateSkill("my-workflow", content); err == nil {
		t.Fatal("expected duplicate create to fail")
	}
}

func TestPatchSkill(t *testing.T) {
	dir := t.TempDir()
	cat := skill.NewCatalog("", dir)
	content := `---
slug: patch-me
name: Patch Me
description: test
---

alpha
`
	if err := cat.CreateSkill("patch-me", content); err != nil {
		t.Fatal(err)
	}
	if err := cat.PatchSkill("patch-me", "alpha", "beta", false); err != nil {
		t.Fatal(err)
	}
	got, _ := cat.ReadSkillMD("patch-me")
	if !strings.Contains(got, "beta") || strings.Contains(got, "alpha") {
		t.Fatalf("patch failed: %q", got)
	}
}

func TestPatchOverridesBuiltin(t *testing.T) {
	dir := t.TempDir()
	cat := skill.NewCatalog("", dir)
	if err := cat.PatchSkill("example", "Example Skill", "Patched Example", false); err != nil {
		t.Fatal(err)
	}
	userFile := filepath.Join(dir, "example", "SKILL.md")
	data, err := os.ReadFile(userFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Patched Example") {
		t.Fatalf("user override not written: %s", data)
	}
}

func TestPatchAmbiguousOldString(t *testing.T) {
	dir := t.TempDir()
	cat := skill.NewCatalog("", dir)
	content := `---
slug: dup
name: Dup
description: test
---

foo bar foo
`
	_ = cat.CreateSkill("dup", content)
	err := cat.PatchSkill("dup", "foo", "baz", false)
	if err == nil || !strings.Contains(err.Error(), "matches 2 times") {
		t.Fatalf("expected ambiguous error, got %v", err)
	}
}
