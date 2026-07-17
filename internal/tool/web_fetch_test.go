package tool

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebFetchSaveBinaryDownload(t *testing.T) {
	dir := t.TempDir()
	tl := &webFetchTool{ws: WorkspaceRoots{Base: dir}}
	out, err := tl.saveBinaryDownload(
		"https://example.com/reports/The-Four-Clusters.pdf",
		"application/pdf",
		[]byte("%PDF-1.4 fake pdf content"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "@/downloads/The-Four-Clusters.pdf") {
		t.Fatalf("want @/downloads path, got: %s", out)
	}
	if !strings.Contains(out, "document_extract") {
		t.Fatalf("want document_extract hint, got: %s", out)
	}
	data, err := os.ReadFile(filepath.Join(dir, "downloads", "The-Four-Clusters.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "%PDF") {
		t.Fatalf("file content corrupted: %q", data)
	}
}

func TestWebFetchSaveBinaryUniqueName(t *testing.T) {
	dir := t.TempDir()
	tl := &webFetchTool{ws: WorkspaceRoots{Base: dir}}
	body := []byte("%PDF-1.4 x")
	if _, err := tl.saveBinaryDownload("https://ex.com/a.pdf", "application/pdf", body); err != nil {
		t.Fatal(err)
	}
	out, err := tl.saveBinaryDownload("https://ex.com/a.pdf", "application/pdf", body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "@/downloads/a.pdf\n") || strings.HasSuffix(strings.Split(out, "\n")[0], "@/downloads/a.pdf).") {
		// First line embeds the path; unique name must differ from a.pdf alone.
	}
	entries, err := os.ReadDir(filepath.Join(dir, "downloads"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 files, got %d", len(entries))
	}
}

func TestDownloadFileNamePrefersMagic(t *testing.T) {
	name := downloadFileName("https://ex.com/a.html", "application/octet-stream", []byte("%PDF-1.4"))
	if !strings.HasSuffix(name, ".pdf") {
		t.Fatalf("got %q", name)
	}
}

func TestLooksBinary(t *testing.T) {
	if !looksBinary([]byte("%PDF-1.4")) {
		t.Fatal("pdf")
	}
	if looksBinary([]byte("<html>hello</html>")) {
		t.Fatal("html should not be binary")
	}
	if !looksBinary([]byte("a\x00b")) {
		t.Fatal("nul")
	}
}

func TestSetBrowserFetchHeaders(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	setBrowserFetchHeaders(req)
	if !strings.Contains(req.Header.Get("User-Agent"), "Chrome/") {
		t.Fatalf("UA=%q", req.Header.Get("User-Agent"))
	}
	if req.Header.Get("Accept-Language") == "" {
		t.Fatal("missing Accept-Language")
	}
	if req.Header.Get("Accept-Encoding") != "" {
		t.Fatal("Accept-Encoding must stay empty for Go auto-gzip")
	}
}

func TestFetchBlockedStatus(t *testing.T) {
	if !fetchBlockedStatus(403) || !fetchBlockedStatus(429) {
		t.Fatal("expected 403/429 blocked")
	}
	if fetchBlockedStatus(404) || fetchBlockedStatus(200) {
		t.Fatal("404/200 should not trigger browser fallback")
	}
}

func TestWebFetchBrowserFallbackUnavailable(t *testing.T) {
	tl := &webFetchTool{ws: WorkspaceRoots{Base: t.TempDir()}, opts: &WebOptions{}}
	_, err := tl.fetchViaBrowser(t.Context(), "https://example.com/", 1000)
	if err == nil || !strings.Contains(err.Error(), "browser") {
		t.Fatalf("want browser unavailable error, got %v", err)
	}
}
