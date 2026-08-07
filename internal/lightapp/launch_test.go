package lightapp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLaunchStaticMissingEntry(t *testing.T) {
	base := t.TempDir()
	m := NewManager(base)
	id := "empty-app"
	if err := m.EnsureDir(id); err != nil {
		t.Fatal(err)
	}
	_, _, err := m.Launch(context.Background(), id, LaunchConfig{
		Runtime:    RuntimeStatic,
		EntryPoint: "index.html",
		BaseDir:    base,
	})
	if err == nil {
		t.Fatal("expected error for missing index.html")
	}
}

func TestLaunchStaticOK(t *testing.T) {
	base := t.TempDir()
	m := NewManager(base)
	id := "ok-app"
	if err := m.EnsureDir(id); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(base, id, "index.html")
	if err := os.WriteFile(path, []byte("<html><body>ok</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	url, port, err := m.Launch(context.Background(), id, LaunchConfig{
		Runtime:    RuntimeStatic,
		EntryPoint: "index.html",
		BaseDir:    base,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Stop(id)
	if port < 30000 || port > 30100 {
		t.Fatalf("unexpected port %d", port)
	}
	if url == "" {
		t.Fatal("empty url")
	}
}
