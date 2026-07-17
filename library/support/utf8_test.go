package support

import "testing"

func TestSanitizeUTF8(t *testing.T) {
	if got := SanitizeUTF8("hello"); got != "hello" {
		t.Fatalf("valid string changed: %q", got)
	}
	invalid := string([]byte{0xe9, 0x87, 0x0a})
	got := SanitizeUTF8("a" + invalid + "b")
	if got == "a"+invalid+"b" {
		t.Fatal("invalid bytes should be replaced")
	}
	withNUL := "ok" + string([]byte{0}) + "yes"
	if got := SanitizeUTF8(withNUL); got != "okyes" {
		t.Fatalf("NUL not stripped: %q", got)
	}
}
