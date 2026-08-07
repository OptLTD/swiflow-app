package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

func TestWorkspaceUpload(t *testing.T) {
	e := newAPIEnv(t)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("path", "."); err != nil {
		t.Fatal(err)
	}
	part, err := w.CreateFormFile("files", "hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("hello world")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, e.server.URL+"/api/workspace/act?act=upload", &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, data)
	}
	if !strings.Contains(string(data), `"hello.txt"`) {
		t.Fatalf("unexpected body: %s", data)
	}

	var payload struct {
		Uploaded []struct {
			Path string `json:"path"`
		} `json:"uploaded"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Uploaded) != 1 || payload.Uploaded[0].Path == "" {
		t.Fatalf("uploaded path missing: %s", data)
	}
	rel := payload.Uploaded[0].Path

	readResp, readData := e.do("GET", "/api/workspace/read?path="+rel, nil)
	if readResp.StatusCode != http.StatusOK {
		t.Fatalf("read status %d: %s", readResp.StatusCode, readData)
	}
	if !strings.Contains(string(readData), "hello world") {
		t.Fatalf("read body: %s", readData)
	}

	dlResp, dlData := e.do("POST", "/api/workspace/act", map[string]string{"act": "download", "path": rel})
	if dlResp.StatusCode != http.StatusOK {
		t.Fatalf("download status %d: %s", dlResp.StatusCode, dlData)
	}
	if !strings.Contains(string(dlData), `"encoding":"base64"`) {
		t.Fatalf("download body: %s", dlData)
	}
	if !strings.Contains(string(dlData), "aGVsbG8gd29ybGQ=") { // "hello world"
		t.Fatalf("download missing base64 payload: %s", dlData)
	}
}

func TestWorkspaceUploadIgnoresClientPath(t *testing.T) {
	// Client "path" is ignored: uploads always land in the immutable uploads/ inbox.
	e := newAPIEnv(t)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("path", "../"); err != nil {
		t.Fatal(err)
	}
	part, err := w.CreateFormFile("files", "evil.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("nope")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, e.server.URL+"/api/workspace/act?act=upload", &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.StatusCode, data)
	}
	if !strings.Contains(string(data), "uploads/") {
		t.Fatalf("expected uploads/ path, got: %s", data)
	}
}
