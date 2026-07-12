package skill

import (
	"context"
	"strings"
	"testing"
)

func TestDiscoverEmbeddedBuiltin(t *testing.T) {
	cat := NewCatalog("", "")
	skills := cat.Discover(context.Background())
	var example *Skill
	for i := range skills {
		if skills[i].Slug == "example" {
			example = &skills[i]
			break
		}
	}
	if example == nil {
		t.Fatal("embedded example skill not found")
	}
	if example.Source != "init" {
		t.Fatalf("source = %q, want init", example.Source)
	}
	if example.Body == "" {
		t.Fatal("example skill body is empty")
	}
	if !strings.HasPrefix(example.Path, "embed:init-skills/") {
		t.Fatalf("path = %q, want embed:init-skills/ prefix", example.Path)
	}
}
