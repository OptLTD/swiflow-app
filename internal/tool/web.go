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

	"github.com/OptLTD/swiflow/library/httputil"
	"github.com/OptLTD/swiflow/library/support"
	"github.com/OptLTD/swiflow/library/workspace"
)

type webFetchTool struct {
	ws WorkspaceRoots
}

func (t *webFetchTool) Name() string { return "web_fetch" }
func (t *webFetchTool) Description() string {
	return "Fetch a URL and return its text content. Binary responses (PDF/image/zip) are saved under @/downloads/ — then use document_extract on that path for PDF/image OCR."
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
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Swiflow/1.0)")
	resp, err := httputil.Do(req, 15*time.Second)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http %d for %s", resp.StatusCode, rawURL)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return "", err
	}
	ct := resp.Header.Get("Content-Type")
	if looksBinary(body) {
		return t.saveBinaryDownload(rawURL, ct, body)
	}
	text := stripHTML(string(body))
	if len(text) > maxChars {
		text = text[:maxChars] + "\n...[truncated]"
	}
	return text, nil
}

func (t *webFetchTool) saveBinaryDownload(rawURL, contentType string, body []byte) (string, error) {
	if strings.TrimSpace(t.ws.Base) == "" {
		return "", fmt.Errorf("url returned binary content (content-type=%q, %d bytes) but workspace is not configured", contentType, len(body))
	}
	name := downloadFileName(rawURL, contentType, body)
	rel := filepath.ToSlash(filepath.Join("downloads", name))
	full, err := support.SandboxPath(t.ws.Base, rel)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	// Avoid clobbering an existing download with the same name.
	if _, err := os.Stat(full); err == nil {
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		name = base + "-" + support.NewID()[:8] + ext
		rel = filepath.ToSlash(filepath.Join("downloads", name))
		full, err = support.SandboxPath(t.ws.Base, rel)
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
		msg += "Call document_extract with path=" + at + " to extract text/OCR."
	default:
		msg += "For PDF/images call document_extract on " + at + "; otherwise use fs_* / other tools on that path."
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

// RegisterWeb registers the web tools. opts may be shared and updated at runtime.
func RegisterWeb(r *Registry, ws WorkspaceRoots, opts *WebOptions) {
	if opts == nil {
		opts = &WebOptions{}
	}
	r.Register(&webFetchTool{ws: ws})
	r.Register(&webSearchTool{opts: opts})
}
