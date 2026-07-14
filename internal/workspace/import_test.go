package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OptLTD/swiflow/internal/workspace"
)

func TestImportFiles(t *testing.T) {
	ws := t.TempDir()
	src := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	uploaded, err := workspace.ImportFiles(ws, ".", []string{src})
	if err != nil {
		t.Fatal(err)
	}
	if len(uploaded) != 1 || uploaded[0].Name != "note.txt" {
		t.Fatalf("uploaded = %+v", uploaded)
	}

	got, err := os.ReadFile(filepath.Join(ws, "note.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("content = %q", got)
	}
}
