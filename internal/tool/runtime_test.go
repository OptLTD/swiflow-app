package tool_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/OptLTD/swiflow/internal/tool"
)

func TestExecRun(t *testing.T) {
	dir := t.TempDir()
	reg := tool.NewRegistry()
	tool.RegisterExec(reg, tool.WorkspaceRoots{Base: dir}, true)

	tl, ok := reg.Get("exec")
	if !ok {
		t.Fatal("exec not registered")
	}
	out, err := tl.Execute(context.Background(), map[string]any{"command": "echo hello"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello\n" && out != "hello" {
		t.Fatalf("got %q", out)
	}
}

func TestExecPythonInline(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		if _, err2 := exec.LookPath("python"); err2 != nil {
			t.Skip("python not installed")
		}
	}
	reg := tool.NewRegistry()
	tool.RegisterExec(reg, tool.WorkspaceRoots{Base: t.TempDir()}, true)
	tl, ok := reg.Get("exec")
	if !ok {
		t.Fatal("exec not registered")
	}
	out, err := tl.Execute(context.Background(), map[string]any{"command": "python3 -c 'print(42)'"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "42\n" && out != "42" {
		t.Fatalf("got %q", out)
	}
}

func TestExecPythonFile(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		if _, err2 := exec.LookPath("python"); err2 != nil {
			t.Skip("python not installed")
		}
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "main.py")
	if err := os.WriteFile(script, []byte("print('file')"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := tool.NewRegistry()
	tool.RegisterExec(reg, tool.WorkspaceRoots{Base: dir}, true)
	tl, _ := reg.Get("exec")
	out, err := tl.Execute(context.Background(), map[string]any{"command": "python3 main.py"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "file\n" && out != "file" {
		t.Fatalf("got %q", out)
	}
}

func TestExecDisabled(t *testing.T) {
	reg := tool.NewRegistry()
	tool.RegisterExec(reg, tool.WorkspaceRoots{Base: t.TempDir()}, false)
	if _, ok := reg.Get("exec"); !ok {
		t.Fatal("exec should be registered")
	}
	if reg.IsEnabled("exec") {
		t.Fatal("exec should be disabled when exec_enabled is false")
	}
	if len(reg.Definitions()) != 0 {
		t.Fatal("disabled exec should not be advertised to the LLM")
	}
}

func TestParseTimeoutClamped(t *testing.T) {
	reg := tool.NewRegistry()
	tool.RegisterExec(reg, tool.WorkspaceRoots{Base: t.TempDir()}, true)
	tl, _ := reg.Get("exec")
	_, err := tl.Execute(context.Background(), map[string]any{
		"command": "sleep 0",
		"timeout": float64(99999),
	})
	if err != nil {
		t.Fatal(err)
	}
}
