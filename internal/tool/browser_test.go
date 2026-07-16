package tool_test

import (
	"context"
	"testing"

	"github.com/OptLTD/swiflow/internal/tool"
	"github.com/OptLTD/swiflow/library/browser"
)

func TestBrowserToolDisabled(t *testing.T) {
	reg := tool.NewRegistry()
	pool := browser.NewPool(true)
	defer pool.Close()
	tool.RegisterBrowser(reg, tool.WorkspaceRoots{Base: t.TempDir()}, pool, tool.BrowserOptions{Enabled: false})

	tl, ok := reg.Get("browser")
	if !ok {
		t.Fatal("browser not registered")
	}
	if reg.IsEnabled("browser") {
		t.Fatal("browser should be disabled")
	}
	_, err := tl.Execute(context.Background(), map[string]any{"action": "content"})
	if err == nil {
		t.Fatal("expected error when disabled")
	}
}

func TestBrowserNavigateExample(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	reg := tool.NewRegistry()
	pool := browser.NewPool(true)
	defer pool.Close()
	tool.RegisterBrowser(reg, tool.WorkspaceRoots{Base: t.TempDir()}, pool, tool.BrowserOptions{Enabled: true, Headless: true})

	tl, _ := reg.Get("browser")
	out, err := tl.Execute(context.Background(), map[string]any{
		"action": "navigate",
		"url":    "https://example.com",
	})
	if err != nil {
		t.Skip("browser not available:", err)
	}
	if out == "" {
		t.Fatal("empty output")
	}
}
