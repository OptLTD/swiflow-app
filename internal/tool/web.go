// Web tools: fetch (SSSF-safe) and search (stub in Phase 1). Spec §8.
package tool

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/OptLTD/swiflow/internal/secure"
)

type webFetchTool struct {
	defaultMax int
}

func (t *webFetchTool) Name() string        { return "web_fetch" }
func (t *webFetchTool) Description() string { return "Fetch a URL and return its text content." }
func (t *webFetchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url":      map[string]any{"type": "string"},
			"max_chars": map[string]any{"type": "integer", "default": 20000},
		},
		"required": []string{"url"},
	}
}
func (t *webFetchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	rawURL, _ := args["url"].(string)
	if err := secure.CheckURL(rawURL); err != nil {
		return "", err
	}
	maxChars := 20000
	if mc, ok := args["max_chars"].(float64); ok && mc > 0 {
		maxChars = int(mc)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Swiflow/1.0")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return "", err
	}
	text := stripHTML(string(body))
	if len(text) > maxChars {
		text = text[:maxChars] + "\n...[truncated]"
	}
	return text, nil
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)
var wsRe = regexp.MustCompile(`\s+`)

func stripHTML(s string) string {
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = htmlTagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return wsRe.ReplaceAllString(s, " ")
}

type webSearchTool struct{}

func (t *webSearchTool) Name() string        { return "web_search" }
func (t *webSearchTool) Description() string { return "Search the web and return results." }
func (t *webSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
			"limit": map[string]any{"type": "integer", "default": 5},
		},
		"required": []string{"query"},
	}
}
func (t *webSearchTool) Execute(_ context.Context, _ map[string]any) (string, error) {
	return "", fmt.Errorf("web search is not configured in this build")
}

// RegisterWeb registers the web tools.
func RegisterWeb(r *Registry) {
	r.Register(&webFetchTool{})
	r.Register(&webSearchTool{})
}
