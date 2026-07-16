package observe

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupFileLog(t *testing.T) {
	dir := t.TempDir()
	ws := filepath.Join(dir, "workspace")
	rel, err := SetupFileLog(ws, slog.LevelInfo)
	if err != nil {
		t.Fatal(err)
	}
	if rel != RelLogFile {
		t.Fatalf("rel path = %q", rel)
	}
	slog.Info("hello from test")
	raw, err := os.ReadFile(filepath.Join(ws, RelLogFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "hello from test") {
		t.Fatalf("log missing entry: %s", raw)
	}
	if !strings.Contains(string(raw), "file logging enabled") {
		t.Fatalf("log missing setup line: %s", raw)
	}
}

func TestSetupFileLogIgnoresBrokenStdout(t *testing.T) {
	// Simulate a Windows GUI process with no usable console by temporarily
	// replacing stdout with a closed pipe writer.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	_ = w.Close()
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	dir := t.TempDir()
	ws := filepath.Join(dir, "workspace")
	if _, err := SetupFileLog(ws, slog.LevelInfo); err != nil {
		t.Fatal(err)
	}
	slog.Info("still written with broken stdout")
	raw, err := os.ReadFile(filepath.Join(ws, RelLogFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "still written with broken stdout") {
		t.Fatalf("expected file write despite stdout failure: %s", raw)
	}
}
