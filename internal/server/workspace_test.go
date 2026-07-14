package server_test

import (
	"bytes"
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

	req, err := http.NewRequest(http.MethodPost, e.server.URL+"/api/workspace/upload", &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+e.token)

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

	readResp, readData := e.do("GET", "/api/workspace/read?path=hello.txt", nil)
	if readResp.StatusCode != http.StatusOK {
		t.Fatalf("read status %d: %s", readResp.StatusCode, readData)
	}
	if !strings.Contains(string(readData), "hello world") {
		t.Fatalf("read body: %s", readData)
	}

	dlResp, dlData := e.do("POST", "/api/workspace/download", map[string]string{"path": "hello.txt"})
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

func TestWorkspaceUploadRejectsEscape(t *testing.T) {
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

	req, err := http.NewRequest(http.MethodPost, e.server.URL+"/api/workspace/upload", &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+e.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
