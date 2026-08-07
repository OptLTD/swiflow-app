// Web tools: fetch (SSRF-safe) and search (provider-backed). Spec §8.
package tool

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"

	"github.com/OptLTD/swiflow/internal/tenant"
	"github.com/OptLTD/swiflow/library/browser"
	"github.com/OptLTD/swiflow/library/httputil"
	"github.com/OptLTD/swiflow/library/support"
	"github.com/OptLTD/swiflow/library/workspace"
)

type webFetchTool struct {
	ws   WorkspaceRoots
	opts *WebOptions
}

func (t *webFetchTool) Name() string { return "web_fetch" }
func (t *webFetchTool) Description() string {
	return "Fetch a URL and return its text content. Uses HTTP first; on anti-bot blocks (403/401/429/…) falls back to the headless browser when enabled. Binary responses (PDF/image/zip) are saved under @/downloads/ — then use content_extract for OCR or structured fields."
}
func (t *webFetchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url":       map[string]any{"type": "string"},
			"max_chars": map[string]any{"type": "integer", "default": 20000},
		},
		"required": []string{"url"},
	}
}

func (t *webFetchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	rawURL, _ := args["url"].(string)
	if err := support.CheckURL(rawURL); err != nil {
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
	setBrowserFetchHeaders(req)
	resp, err := httputil.Do(req, 15*time.Second)
	if err != nil {
		if text, berr := t.fetchViaBrowser(ctx, rawURL, maxChars); berr == nil {
			return "[fetched via browser after HTTP error: " + err.Error() + "]\n" + text, nil
		}
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		if fetchBlockedStatus(resp.StatusCode) {
			if text, berr := t.fetchViaBrowser(ctx, rawURL, maxChars); berr == nil {
				return fmt.Sprintf("[fetched via browser after http %d]\n%s", resp.StatusCode, text), nil
			} else if berr != nil && t.canBrowser() {
				return "", fmt.Errorf("http %d for %s; browser fallback: %w", resp.StatusCode, rawURL, berr)
			}
		}
		return "", fmt.Errorf("http %d for %s", resp.StatusCode, rawURL)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return "", err
	}
	ct := resp.Header.Get("Content-Type")
	if looksBinary(body) {
		return t.saveBinaryDownload(ctx, rawURL, ct, body)
	}
	text := stripHTML(string(body))
	if len(text) > maxChars {
		text = text[:maxChars] + "\n...[truncated]"
	}
	return text, nil
}

func (t *webFetchTool) canBrowser() bool {
	return t != nil && t.opts != nil && t.opts.BrowserEnabled && t.opts.BrowserPool != nil
}

func fetchBlockedStatus(code int) bool {
	switch code {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests,
		http.StatusServiceUnavailable, http.StatusBadGateway:
		return true
	default:
		return false
	}
}

func (t *webFetchTool) fetchViaBrowser(ctx context.Context, rawURL string, maxChars int) (string, error) {
	if !t.canBrowser() {
		return "", fmt.Errorf("browser fallback unavailable (enable tools.browser_enabled)")
	}
	return t.opts.BrowserPool.WithPageTenant(ctx, tenant.ID(ctx), 45*time.Second, func(page *rod.Page) (string, error) {
		if _, err := browser.Open(page, rawURL); err != nil {
			return "", err
		}
		info, err := page.Info()
		if err != nil {
			return "", err
		}
		body, err := browser.PageText(page, maxChars)
		if err != nil {
			return "", err
		}
		if block, ok := browser.DetectBotBlock(info.Title, info.URL, body); ok {
			return "", block
		}
		return fmt.Sprintf("title: %s\nurl: %s\n\n%s", info.Title, info.URL, body), nil
	})
}

func (t *webFetchTool) saveBinaryDownload(ctx context.Context, rawURL, contentType string, body []byte) (string, error) {
	base := WorkspaceBase(ctx, t.ws.Base)
	if strings.TrimSpace(base) == "" {
		return "", fmt.Errorf("url returned binary content (content-type=%q, %d bytes) but workspace is not configured", contentType, len(body))
	}
	name := downloadFileName(rawURL, contentType, body)
	rel := filepath.ToSlash(filepath.Join("downloads", name))
	full, err := support.SandboxPath(base, rel)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	// Avoid clobbering an existing download with the same name.
	if _, err := os.Stat(full); err == nil {
		ext := filepath.Ext(name)
		stem := strings.TrimSuffix(name, ext)
		name = stem + "-" + support.NewID()[:8] + ext
		rel = filepath.ToSlash(filepath.Join("downloads", name))
		full, err = support.SandboxPath(base, rel)
		if err != nil {
			return "", err
		}
	}
	if err := os.WriteFile(full, body, 0o644); err != nil {
		return "", err
	}
	at := "@/" + rel
	kind := binaryKind(body)
	msg := fmt.Sprintf("saved binary to %s (%d bytes, content-type=%q, kind=%s).\n", at, len(body), contentType, kind)
	msg += "web_fetch only returns text for HTML/plain responses. "
	switch kind {
	case "pdf", "image":
		msg += "Call content_extract with path=" + at + " to OCR or extract fields."
	default:
		msg += "For PDF/images/docs call content_extract on " + at + "; otherwise use fs_* / other tools on that path."
	}
	return msg, nil
}

func downloadFileName(rawURL, contentType string, body []byte) string {
	var base string
	if u, err := url.Parse(rawURL); err == nil {
		base = filepath.Base(u.Path)
	}
	base = strings.TrimSpace(base)
	if base == "" || base == "/" || base == "." {
		base = "download"
	}
	if name, err := workspace.SafeUploadName(base); err == nil {
		base = name
	} else {
		base = "download"
	}
	ext := strings.ToLower(filepath.Ext(base))
	want := extForBinary(contentType, body)
	if ext == "" && want != "" {
		base += want
	} else if ext == "" {
		base += ".bin"
	} else if want != "" && ext != want {
		// URL lied (e.g. .html that is actually PDF) — prefer magic.
		base = strings.TrimSuffix(base, ext) + want
	}
	return base
}

func extForBinary(contentType string, body []byte) string {
	switch binaryKind(body) {
	case "pdf":
		return ".pdf"
	case "image":
		if bytes.HasPrefix(body, []byte{0x89, 'P', 'N', 'G'}) {
			return ".png"
		}
		if bytes.HasPrefix(body, []byte{0xff, 0xd8, 0xff}) {
			return ".jpg"
		}
		return ".img"
	case "zip":
		return ".zip"
	}
	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "pdf"):
		return ".pdf"
	case strings.Contains(ct, "png"):
		return ".png"
	case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
		return ".jpg"
	case strings.Contains(ct, "zip"):
		return ".zip"
	}
	return ""
}

func binaryKind(body []byte) string {
	switch {
	case bytes.HasPrefix(body, []byte("%PDF")):
		return "pdf"
	case bytes.HasPrefix(body, []byte{0x89, 'P', 'N', 'G'}),
		bytes.HasPrefix(body, []byte{0xff, 0xd8, 0xff}):
		return "image"
	case bytes.HasPrefix(body, []byte("PK")):
		return "zip"
	default:
		return "binary"
	}
}

func looksBinary(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	if bytes.HasPrefix(b, []byte("%PDF")) ||
		bytes.HasPrefix(b, []byte{0x89, 'P', 'N', 'G'}) ||
		bytes.HasPrefix(b, []byte{0xff, 0xd8, 0xff}) ||
		bytes.HasPrefix(b, []byte("PK")) {
		return true
	}
	n := len(b)
	if n > 8000 {
		n = 8000
	}
	for _, c := range b[:n] {
		if c == 0 {
			return true
		}
	}
	return false
}

// browserFetchUA is a current desktop Chrome UA. The old "compatible; Swiflow/1.0"
// string is treated as a bot by many CDNs and news sites.
const browserFetchUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

func setBrowserFetchHeaders(req *http.Request) {
	if req == nil {
		return
	}
	req.Header.Set("User-Agent", browserFetchUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	// Do not set Accept-Encoding: Go's Transport handles gzip transparently only
	// when this header is left unset by the caller.
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Cache-Control", "max-age=0")
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

// RegisterWeb registers the web tools. opts may be shared and updated at runtime
// (BrowserPool / BrowserEnabled are often set after RegisterWeb returns).
func RegisterWeb(r *Registry, ws WorkspaceRoots, opts *WebOptions) {
	if opts == nil {
		opts = &WebOptions{}
	}
	r.Register(&webFetchTool{ws: ws, opts: opts})
	r.Register(&webSearchTool{opts: opts})
}
