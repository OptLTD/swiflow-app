package secure_test

import (
	"strings"
	"testing"

	"github.com/OptLTD/swiflow/internal/secure"
)

func TestSandboxPathRejectsEscape(t *testing.T) {
	ws := t.TempDir()
	_, err := secure.SandboxPath(ws, "../etc/passwd")
	if err == nil {
		t.Fatal("expected escape rejection")
	}
}

func TestSandboxPathAllowsInside(t *testing.T) {
	ws := t.TempDir()
	p, err := secure.SandboxPath(ws, "notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p, "notes.txt") {
		t.Fatalf("got %s", p)
	}
}

func TestValidateHTTPURL(t *testing.T) {
	if err := secure.ValidateHTTPURL("https://api.openai.com/v1"); err != nil {
		t.Fatal(err)
	}
	if err := secure.ValidateHTTPURL("ftp://example.com"); err == nil {
		t.Fatal("expected scheme error")
	}
}

func TestEncryptRoundTrip(t *testing.T) {
	key := secure.DeriveKey("test-passphrase-16chars")
	ct, err := secure.Encrypt(key, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	pt, err := secure.Decrypt(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if string(pt) != "secret" {
		t.Fatalf("got %q", pt)
	}
}
