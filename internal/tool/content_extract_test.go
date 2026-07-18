package tool

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentToolDisabled(t *testing.T) {
	tl := &contentExtractTool{allowed: false}
	_, err := tl.Execute(context.Background(), map[string]any{"path": "a.txt", "prompt": "x"})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("err=%v", err)
	}
}

func TestDocumentToolRequiresProvider(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("demo"), 0o644); err != nil {
		t.Fatal(err)
	}
	tl := &contentExtractTool{
		ws:      WorkspaceRoots{Base: dir},
		allowed: true,
	}
	_, err := tl.Execute(context.Background(), map[string]any{"path": "a.txt", "prompt": "extract"})
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("err=%v", err)
	}
}

func TestDocumentToolRequiresPath(t *testing.T) {
	tl := &contentExtractTool{
		allowed: true,
		opt:     DocumentOptions{APIKey: "sk-test"},
	}
	_, err := tl.Execute(context.Background(), map[string]any{"prompt": "x"})
	if err == nil || !strings.Contains(err.Error(), "path required") {
		t.Fatalf("err=%v", err)
	}
}
