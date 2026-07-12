package mcpclient

import "testing"

func TestToolName(t *testing.T) {
	got := ToolName("my-server", "read_file")
	want := "mcp_my_server_read_file"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSanitize(t *testing.T) {
	if sanitize("") != "x" {
		t.Fatal("empty should become x")
	}
}
