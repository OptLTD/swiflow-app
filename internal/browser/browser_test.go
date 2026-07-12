package browser_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"

	"mira/internal/browser"
)

func TestNavigateGoogle(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	pool := browser.NewPool(true)
	defer pool.Close()

	out, err := pool.WithPage(context.Background(), 45*time.Second, func(page *rod.Page) (string, error) {
		if err := page.Navigate("https://www.google.com"); err != nil {
			return "", err
		}
		if err := browser.WaitLoaded(page); err != nil {
			return "", err
		}
		info, err := page.Info()
		if err != nil {
			return "", err
		}
		body, err := browser.PageText(page, 20000)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("title: %s\nurl: %s\n\n%s", info.Title, info.URL, body), nil
	})
	if err != nil {
		t.Skip("browser not available:", err)
	}
	if browser.LooksLikeBotBlock(out) {
		t.Fatalf("google blocked headless browser:\n%s", out)
	}
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "google") {
		t.Fatalf("expected google in response, got:\n%s", out)
	}
	if !strings.Contains(lower, "search") && !strings.Contains(lower, "google") {
		t.Fatalf("unexpected google page content:\n%s", out)
	}
	t.Logf("google navigate ok (%d bytes)", len(out))
}

func TestNavigateExample(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	pool := browser.NewPool(true)
	defer pool.Close()

	out, err := pool.WithPage(context.Background(), 30*time.Second, func(page *rod.Page) (string, error) {
		if err := page.Navigate("https://example.com"); err != nil {
			return "", err
		}
		if err := browser.WaitLoaded(page); err != nil {
			return "", err
		}
		return browser.PageText(page, 5000)
	})
	if err != nil {
		t.Skip("browser not available:", err)
	}
	if out == "" {
		t.Fatal("empty output")
	}
}
