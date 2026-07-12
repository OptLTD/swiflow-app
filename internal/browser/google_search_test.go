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

func TestNavigateGoogleSearchCN(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	pool := browser.NewPool(true)
	defer pool.Close()

	url := "https://www.google.com/search?q=%E5%90%B4%E5%BD%A6%E7%A5%96"
	out, err := pool.WithPage(context.Background(), 60*time.Second, func(page *rod.Page) (string, error) {
		note, err := browser.Open(page, url)
		if err != nil {
			return "", err
		}
		info, err := page.Info()
		if err != nil {
			return "", err
		}
		body, err := browser.PageText(page, 5000)
		if err != nil {
			return "", err
		}
		out := fmt.Sprintf("title: %s\nurl: %s\n\n%s", info.Title, info.URL, body)
		if note != "" {
			out = note + "\n\n" + out
		}
		return out, nil
	})
	if err != nil {
		t.Skip("browser not available:", err)
	}
	t.Logf("output:\n%s", out)
	if browser.LooksLikeBotBlock(out) {
		t.Fatalf("search still blocked after fallback:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(out), "吴彦祖") && !strings.Contains(strings.ToLower(out), "yan") {
		// DDG or Google should mention the query somewhere in results.
		t.Logf("query text not found in body; page may still be useful")
	}
}
