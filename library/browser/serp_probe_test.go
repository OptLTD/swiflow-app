package browser_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"

	"github.com/OptLTD/swiflow/library/browser"
)

func TestProbeBingSERP(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	// Bing must use direct Chrome — local Clash often breaks cn.bing.com.
	pool := browser.NewPool(true)
	defer pool.Close()

	rawURL := "https://cn.bing.com/search?q=golang"
	start := time.Now()
	out, err := pool.WithPage(context.Background(), 60*time.Second, func(page *rod.Page) (string, error) {
		note, err := browser.Open(page, rawURL)
		if err != nil {
			return "", fmt.Errorf("open: %w", err)
		}
		info, err := page.Info()
		if err != nil {
			return "", err
		}
		t.Logf("after open (%s): title=%q url=%q note=%q", time.Since(start).Round(time.Millisecond), info.Title, info.URL, note)

		res, err := page.Timeout(10 * time.Second).Eval(`(limit) => {
  const out = [];
  const nodes = document.querySelectorAll('#b_results > li.b_algo, ol#b_results li.b_algo');
  for (const el of nodes) {
    if (out.length >= limit) break;
    const a = el.querySelector('h2 a');
    if (!a || !a.href) continue;
    const title = (a.innerText || a.textContent || '').trim();
    if (!title) continue;
    out.push({ title, url: a.href });
  }
  return { count: nodes.length, items: out };
}`, 5)
		if err != nil {
			return "", fmt.Errorf("extract: %w", err)
		}
		return res.Value.String(), nil
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("bing probe failed after %s: %v", elapsed.Round(time.Millisecond), err)
	}
	t.Logf("bing ok in %s: %s", elapsed.Round(time.Millisecond), truncate(out, 800))
	if !strings.Contains(out, "http") {
		t.Fatalf("no http urls in extract: %s", out)
	}
}

func TestProbeGoogleSERP(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	proxy := browser.ResolveBrowserProxy()
	if proxy == "" {
		t.Skip("no reachable proxy for Google")
	}
	t.Logf("using proxy %s", proxy)
	pool := browser.NewPoolWithProxy(true, proxy)
	defer pool.Close()

	rawURL := "https://www.google.com/search?q=golang"
	start := time.Now()
	out, err := pool.WithPage(context.Background(), 60*time.Second, func(page *rod.Page) (string, error) {
		note, err := browser.Open(page, rawURL)
		if err != nil {
			return "", fmt.Errorf("open: %w", err)
		}
		info, err := page.Info()
		if err != nil {
			return "", err
		}
		t.Logf("after open (%s): title=%q url=%q note=%q", time.Since(start).Round(time.Millisecond), info.Title, info.URL, note)
		body, _ := browser.PageText(page, 1500)
		t.Logf("body preview: %s", truncate(body, 300))

		res, err := page.Timeout(10 * time.Second).Eval(`(limit) => {
  const out = [];
  const seen = new Set();
  const nodes = document.querySelectorAll('#search a h3, #rso a h3, div.g a h3');
  for (const h3 of nodes) {
    if (out.length >= limit) break;
    const a = h3.closest('a');
    if (!a || !a.href) continue;
    const href = a.href;
    if (seen.has(href)) continue;
    if (href.includes('google.') && (href.includes('/search') || href.includes('accounts.google'))) continue;
    const title = (h3.innerText || h3.textContent || '').trim();
    if (!title) continue;
    seen.add(href);
    out.push({ title, url: href });
  }
  return { count: nodes.length, items: out, href: location.href };
}`, 5)
		if err != nil {
			return "", fmt.Errorf("extract: %w; title=%q url=%q", err, info.Title, info.URL)
		}
		return res.Value.String(), nil
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("google probe failed after %s: %v", elapsed.Round(time.Millisecond), err)
	}
	t.Logf("google result in %s: %s", elapsed.Round(time.Millisecond), truncate(out, 800))
	// Bot-block → DuckDuckGo fallback is acceptable; zero google h3s is then expected.
	if strings.Contains(out, "duckduckgo.com") || strings.Contains(strings.ToLower(out), "duckduckgo") {
		t.Log("landed on DuckDuckGo fallback (Google bot wall)")
		return
	}
	if !strings.Contains(out, "http") {
		t.Fatalf("no parseable google results: %s", out)
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
