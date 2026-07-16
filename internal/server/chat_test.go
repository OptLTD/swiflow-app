package server_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestChatSupportsSSE(t *testing.T) {
	e := newAPIEnv(t)
	body := []byte(`{"message":"hi","agent":"default"}`)
	req, err := http.NewRequest(http.MethodPost, e.server.URL+"/api/sessions/sse-test/chat", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusInternalServerError && bytes.Contains(data, []byte("streaming unsupported")) {
		t.Fatalf("chat handler lost http.Flusher through middleware: %s", data)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type = %q body = %s", ct, data)
	}
}
