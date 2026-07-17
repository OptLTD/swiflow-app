package tool

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebSearchNotConfigured(t *testing.T) {
	tool := &webSearchTool{}
	_, err := tool.Execute(context.Background(), map[string]any{"query": "go"})
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseDuckDuckGoHTML(t *testing.T) {
	html := `
<div class="result results_links">
  <div class="links_main">
    <a rel="nofollow" class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fpage">Example Title</a>
    <a class="result__snippet" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fpage">A short snippet here.</a>
  </div>
</div>
<div class="result results_links">
  <div class="links_main">
    <a class="result__a" href="https://golang.org">The Go Programming Language</a>
    <td class="result__snippet">Build simple, secure, scalable systems.</td>
  </div>
</div>`
	got := parseDuckDuckGoHTML(html, 5)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %+v", len(got), got)
	}
	if got[0].URL != "https://example.com/page" {
		t.Fatalf("url[0] = %q", got[0].URL)
	}
	if got[0].Title != "Example Title" {
		t.Fatalf("title[0] = %q", got[0].Title)
	}
	if !strings.Contains(got[0].Snippet, "short snippet") {
		t.Fatalf("snippet[0] = %q", got[0].Snippet)
	}
	if got[1].URL != "https://golang.org" {
		t.Fatalf("url[1] = %q", got[1].URL)
	}
}

func TestSearchBrave(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Subscription-Token") != "test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"web":{"results":[
			{"title":"Go","url":"https://go.dev","description":"The Go site"},
			{"title":"Spec","url":"https://go.dev/ref/spec","description":"Language spec"}
		]}}`))
	}))
	defer srv.Close()

	_, err := searchBrave(context.Background(), "", "go", 5)
	if err == nil || !strings.Contains(err.Error(), "api_key") {
		t.Fatalf("missing key err = %v", err)
	}

	prev := braveSearchEndpoint
	braveSearchEndpoint = srv.URL
	t.Cleanup(func() { braveSearchEndpoint = prev })

	got, err := searchBrave(context.Background(), "test-key", "go", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].URL != "https://go.dev" {
		t.Fatalf("got = %+v", got)
	}
}

func TestSearchSearXNG(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("q") != "swiflow" {
			t.Errorf("q = %q", r.URL.Query().Get("q"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[
			{"title":"Swiflow","url":"https://example.com/swiflow","content":"AI agent runtime"},
			{"title":"Docs","url":"https://example.com/docs","content":"Documentation"}
		]}`))
	}))
	defer srv.Close()

	got, err := searchSearXNG(context.Background(), srv.URL, "swiflow", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].Title != "Swiflow" || got[0].URL != "https://example.com/swiflow" {
		t.Fatalf("got[0] = %+v", got[0])
	}

	tool := &webSearchTool{opts: &WebOptions{SearchProvider: "searxng", SearchBaseURL: srv.URL}}
	out, err := tool.Execute(context.Background(), map[string]any{"query": "swiflow", "limit": float64(1)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "1. Swiflow") || !strings.Contains(out, "https://example.com/swiflow") {
		t.Fatalf("out = %q", out)
	}
	if strings.Contains(out, "2.") {
		t.Fatalf("limit not respected: %q", out)
	}
}

func TestUnwrapDuckDuckGoURL(t *testing.T) {
	in := "//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fa%3Fb%3D1"
	got := unwrapDuckDuckGoURL(in)
	if got != "https://example.com/a?b=1" {
		t.Fatalf("got %q", got)
	}
}

func TestRequireBrowserSearch(t *testing.T) {
	if err := requireBrowserSearch(nil, "bing"); err == nil {
		t.Fatal("want error when opts nil")
	}
	if err := requireBrowserSearch(&WebOptions{}, "bing"); err == nil {
		t.Fatal("want error when browser disabled")
	}
}

func TestBingSearchRequiresBrowserEnabled(t *testing.T) {
	reg := NewRegistry()
	RegisterWeb(reg, WorkspaceRoots{Base: t.TempDir()}, &WebOptions{SearchProvider: "bing"})
	tl, ok := reg.Get("web_search")
	if !ok {
		t.Fatal("missing web_search")
	}
	_, err := tl.Execute(t.Context(), map[string]any{"query": "test"})
	if err == nil || !strings.Contains(err.Error(), "browser") {
		t.Fatalf("want browser-required error, got %v", err)
	}
}
