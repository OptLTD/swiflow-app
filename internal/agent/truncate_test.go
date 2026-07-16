package agent

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateToolResultUTF8Safe(t *testing.T) {
	long := strings.Repeat("中", 5000)
	out := truncateToolResult(long)
	if !utf8.ValidString(out) {
		t.Fatal("truncated tool result must be valid UTF-8")
	}
}
