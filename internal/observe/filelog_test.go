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
	data := filepath.Join(dir, "data")
	abs, err := SetupFileLog(data, slog.LevelInfo)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(data, RelLogFile)
	if abs != want {
		t.Fatalf("abs path = %q, want %q", abs, want)
	}
	if LogFileAbsPath() != want {
		t.Fatalf("LogFileAbsPath = %q", LogFileAbsPath())
	}
	slog.Info("hello from test")
	raw, err := os.ReadFile(want)
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
	data := filepath.Join(dir, "data")
	if _, err := SetupFileLog(data, slog.LevelInfo); err != nil {
		t.Fatal(err)
	}
	slog.Info("still written with broken stdout")
	raw, err := os.ReadFile(filepath.Join(data, RelLogFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "still written with broken stdout") {
		t.Fatalf("expected file write despite stdout failure: %s", raw)
	}
}
