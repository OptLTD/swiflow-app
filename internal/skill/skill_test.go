package skill

import (
	"context"
	"strings"
	"testing"
)

func TestDiscoverEmbeddedBuiltin(t *testing.T) {
	cat := NewCatalog("", "")
	skills := cat.Discover(context.Background())
	var hit *Skill
	for i := range skills {
		if skills[i].Slug == "window-context" {
			hit = &skills[i]
			break
		}
	}
	if hit == nil {
		t.Fatal("embedded window-context skill not found")
	}
	if hit.Source != "init" {
		t.Fatalf("source = %q, want init", hit.Source)
	}
	if hit.Body == "" {
		t.Fatal("skill body is empty")
	}
	if !strings.HasPrefix(hit.Path, "embed:init-skills/") {
		t.Fatalf("path = %q, want embed:init-skills/ prefix", hit.Path)
	}
}
