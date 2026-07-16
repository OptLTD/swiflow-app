package document

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeProvider struct {
	last ProviderRequest
	out  *Result
}

func (f *fakeProvider) Extract(_ context.Context, req ProviderRequest) (*Result, error) {
	f.last = req
	return f.out, nil
}

func TestDetectInputType(t *testing.T) {
	cases := map[string]string{
		"invoice.png": "img",
		"scan.pdf":    "pdf",
		"notes.txt":   "txt",
	}
	for path, want := range cases {
		if got := DetectInputType(path, "auto"); got != want {
			t.Fatalf("%s => %s, want %s", path, got, want)
		}
	}
}

func TestServiceExtractText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.txt")
	if err := os.WriteFile(path, []byte("hello swiflow"), 0o644); err != nil {
		t.Fatal(err)
	}
	fp := &fakeProvider{out: &Result{DocType: "note", Fields: map[string]any{"title": "demo"}}}
	svc := NewService(fp)
	out, err := svc.Extract(context.Background(), Request{
		Path:           path,
		InputType:      "auto",
		Fields:         []string{"title"},
		IncludeRawText: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.InputType != "txt" {
		t.Fatalf("input_type=%s", out.InputType)
	}
	if !strings.Contains(out.RawText, "hello swiflow") {
		t.Fatalf("raw_text=%q", out.RawText)
	}
	if fp.last.Text != "hello swiflow" {
		t.Fatalf("provider text=%q", fp.last.Text)
	}
}

func TestServiceExtractImage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.png")
	if err := os.WriteFile(path, []byte{0x89, 0x50, 0x4e, 0x47}, 0o644); err != nil {
		t.Fatal(err)
	}
	fp := &fakeProvider{out: &Result{DocType: "receipt", Fields: map[string]any{}}}
	svc := NewService(fp)
	out, err := svc.Extract(context.Background(), Request{
		Path:      path,
		InputType: "auto",
		Prompt:    "extract total",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.InputType != "img" {
		t.Fatalf("input_type=%s", out.InputType)
	}
	if !strings.HasPrefix(fp.last.ImageDataURL, "data:image/png;base64,") {
		t.Fatalf("image data url=%q", fp.last.ImageDataURL)
	}
}
