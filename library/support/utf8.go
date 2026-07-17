package support

import (
	"strings"
	"unicode/utf8"
)

// SanitizeUTF8 makes a string safe for Postgres text columns and JSON payloads:
// invalid UTF-8 sequences become U+FFFD, and NUL bytes (valid UTF-8 but rejected
// by Postgres) are stripped.
func SanitizeUTF8(s string) string {
	if s == "" {
		return s
	}
	if utf8.ValidString(s) && !strings.ContainsRune(s, 0) {
		return s
	}
	b := make([]rune, 0, len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == 0 {
			i += size
			continue
		}
		if r == utf8.RuneError && size == 1 {
			b = append(b, '\uFFFD')
			i++
			continue
		}
		b = append(b, r)
		i += size
	}
	return string(b)
}
