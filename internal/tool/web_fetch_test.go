package tool

import (
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
