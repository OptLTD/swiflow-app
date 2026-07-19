package tool_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/OptLTD/swiflow/internal/tool"
	"github.com/OptLTD/swiflow/library/browser"
)

func TestE2EWebFetchZhihuBrowserFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	pool := browser.NewPool(true)
	defer pool.Close()
	opts := &tool.WebOptions{BrowserPool: pool, BrowserEnabled: true}
	reg := tool.NewRegistry()
	tool.RegisterWeb(reg, tool.WorkspaceRoots{Base: t.TempDir()}, opts)
	tl, ok := reg.Get("web_fetch")
	if !ok {
		t.Fatal("missing web_fetch")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := tl.Execute(ctx, map[string]any{
		"url":       "https://zhuanlan.zhihu.com/p/1898862235943703974",
		"max_chars": float64(2000),
	})
	t.Logf("err=%v\n%s", err, out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "browser") {
		t.Fatalf("expected browser fallback note, got: %s", out[:min(200, len(out))])
	}
	// Zhihu often serves captcha / login walls to datacenter IPs — not a product regression.
	if strings.Contains(out, "安全验证") || strings.Contains(out, "unhuman") || strings.Contains(out, "请您登录") {
		t.Skip("zhihu bot/login wall; skip live e2e")
	}
	if !strings.Contains(out, "孙燕姿") {
		t.Fatalf("expected article content, got: %s", out[:min(400, len(out))])
	}
}
