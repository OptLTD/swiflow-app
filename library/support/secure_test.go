package support_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/OptLTD/swiflow/library/support"
)

func TestSandboxPathRejectsEscape(t *testing.T) {
	ws := t.TempDir()
	_, err := support.SandboxPath(ws, "../etc/passwd")
	if err == nil {
		t.Fatal("expected escape rejection")
	}
}

func TestSandboxPathAllowsInside(t *testing.T) {
	ws := t.TempDir()
	p, err := support.SandboxPath(ws, "notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p, "notes.txt") {
		t.Fatalf("got %s", p)
	}
}

func TestSandboxPathAtAlias(t *testing.T) {
	ws := t.TempDir()
	p, err := support.SandboxPath(ws, "@/notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p, "notes.txt") {
		t.Fatalf("got %s", p)
	}
	// Must not create a literal "@" directory segment.
	if strings.Contains(p, string([]byte{'@', filepath.Separator})) || strings.Contains(p, "@/") {
		t.Fatalf("literal @/ retained: %s", p)
	}
	p2, err := support.SandboxPath(ws, "@/docs/a.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p2, filepath.Join("docs", "a.md")) {
		t.Fatalf("got %s", p2)
	}
}

func TestCheckURL(t *testing.T) {
	if err := support.CheckURL("https://example.com"); err != nil {
		t.Fatal(err)
	}
	if err := support.CheckURL("http://127.0.0.1:30000"); err == nil {
		t.Fatal("expected private IP rejection")
	}
	if err := support.CheckURL("http://localhost/"); err == nil {
		t.Fatal("expected localhost rejection")
	}
}

func TestCheckURLAllowLoopback(t *testing.T) {
	for _, u := range []string{
		"http://127.0.0.1:30000",
		"http://localhost:30000",
		"http://[::1]:30000",
	} {
		if err := support.CheckURLAllowLoopback(u); err != nil {
			t.Fatalf("%s: %v", u, err)
		}
	}
	// Non-loopback private still blocked.
	if err := support.CheckURLAllowLoopback("http://192.168.1.1/"); err == nil {
		t.Fatal("expected private LAN rejection")
	}
	if err := support.CheckURLAllowLoopback("http://metadata.google.internal/"); err == nil {
		t.Fatal("expected metadata host rejection")
	}
}

func TestEncryptRoundTrip(t *testing.T) {
	key := support.DeriveKey("test-passphrase-16chars")
	ct, err := support.Encrypt(key, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	pt, err := support.Decrypt(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if string(pt) != "secret" {
		t.Fatalf("got %q", pt)
	}
}
