package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OptLTD/swiflow/library/workspace"
)

func TestAllocUploadRel(t *testing.T) {
	rel, err := workspace.AllocUploadRel("note.txt")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(rel, "/")
	if len(parts) != 3 || parts[0] != workspace.UploadsRoot {
		t.Fatalf("rel=%q", rel)
	}
	if len(parts[1]) != 8 { // YYYYMMDD
		t.Fatalf("day=%q", parts[1])
	}
	if !strings.HasSuffix(parts[2], "_note.txt") {
		t.Fatalf("file=%q", parts[2])
	}
}

func TestImportFiles(t *testing.T) {
	ws := t.TempDir()
	src := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	uploaded, err := workspace.ImportFiles(ws, []string{src})
	if err != nil {
		t.Fatal(err)
	}
	if len(uploaded) != 1 || uploaded[0].Name != "note.txt" {
		t.Fatalf("uploaded = %+v", uploaded)
	}
	if !strings.HasPrefix(uploaded[0].Path, workspace.UploadsRoot+"/") {
		t.Fatalf("path=%q want under uploads/", uploaded[0].Path)
	}

	got, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(uploaded[0].Path)))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("content = %q", got)
	}

	// Second import of same name must not overwrite the first.
	uploaded2, err := workspace.ImportFiles(ws, []string{src})
	if err != nil {
		t.Fatal(err)
	}
	if uploaded2[0].Path == uploaded[0].Path {
		t.Fatal("expected unique upload paths")
	}
	if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(uploaded[0].Path))); err != nil {
		t.Fatalf("first upload missing after second: %v", err)
	}
}
