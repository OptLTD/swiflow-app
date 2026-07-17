package tool_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/OptLTD/swiflow/internal/tool"
	"github.com/OptLTD/swiflow/library/browser"
	"github.com/OptLTD/swiflow/library/httputil"
)

func TestE2EBingBrowserSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	pool := browser.NewPool(true)
	defer pool.Close()
	reg := tool.NewRegistry()
	tool.RegisterWeb(reg, tool.WorkspaceRoots{Base: t.TempDir()}, &tool.WebOptions{
		SearchProvider: "bing",
		BrowserPool:    pool,
		BrowserEnabled: true,
	})
	tr, ok := reg.Get("web_search")
	if !ok {
		t.Fatal("web_search not registered")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	start := time.Now()
	out, err := tr.Execute(ctx, map[string]any{"query": "孙燕姿 演唱会 2025", "limit": 5})
	t.Logf("bing in %s: err=%v\n%s", time.Since(start).Round(time.Millisecond), err, out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "孙燕姿") {
		t.Fatalf("expected 孙燕姿 in results, got: %s", out)
	}
	if strings.Contains(out, "192.168.") {
		t.Fatalf("poisoned router SERP leaked through: %s", out)
	}
}

func TestE2EGoogleBrowserSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	t.Logf("ProxyServer=%q LocalProxy=%q", httputil.ProxyServer(), httputil.LocalProxyServer())
	pool := browser.NewPool(true)
	defer pool.Close()
	reg := tool.NewRegistry()
	tool.RegisterWeb(reg, tool.WorkspaceRoots{Base: t.TempDir()}, &tool.WebOptions{
		SearchProvider: "google",
		BrowserPool:    pool,
		BrowserEnabled: true,
	})
	tr, ok := reg.Get("web_search")
	if !ok {
		t.Fatal("web_search not registered")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	start := time.Now()
	out, err := tr.Execute(ctx, map[string]any{"query": "golang tutorial", "num_results": 3})
	t.Logf("google in %s: err=%v\n%s", time.Since(start).Round(time.Millisecond), err, out)
	if err != nil {
		t.Fatal(err)
	}
	if out == "No results found." || !strings.Contains(out, "http") {
		t.Fatalf("expected urls, got: %s", out)
	}
}
