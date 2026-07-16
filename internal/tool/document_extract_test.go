package tool

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OptLTD/swiflow/library/document"
)

type stubProvider struct{}

func (stubProvider) Extract(_ context.Context, req document.ProviderRequest) (*document.Result, error) {
	return &document.Result{
		DocType: "note",
		Fields:  map[string]any{"task": "demo"},
		Meta:    map[string]any{"input": req.InputType},
	}, nil
}

func TestDocumentToolDisabled(t *testing.T) {
	tl := &documentExtractTool{allowed: false}
	_, err := tl.Execute(context.Background(), map[string]any{"path": "a.txt", "prompt": "x"})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("err=%v", err)
	}
}

func TestDocumentToolValidatesExtractionRequest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("demo"), 0o644); err != nil {
		t.Fatal(err)
	}
	tl := &documentExtractTool{
		ws:      WorkspaceRoots{Base: dir},
		svc:     document.NewService(stubProvider{}),
		allowed: true,
	}
	_, err := tl.Execute(context.Background(), map[string]any{"path": "a.txt"})
	if err == nil || !strings.Contains(err.Error(), "fields, schema, or prompt required") {
		t.Fatalf("err=%v", err)
	}
}
