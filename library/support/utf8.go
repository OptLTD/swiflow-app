package support

import "unicode/utf8"

// SanitizeUTF8 replaces invalid UTF-8 byte sequences so Postgres/text columns accept the string.
func SanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	b := make([]rune, 0, len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
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
