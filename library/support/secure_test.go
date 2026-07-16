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

func TestValidateHTTPURL(t *testing.T) {
	if err := support.ValidateHTTPURL("https://api.openai.com/v1"); err != nil {
		t.Fatal(err)
	}
	if err := support.ValidateHTTPURL("ftp://example.com"); err == nil {
		t.Fatal("expected scheme error")
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
