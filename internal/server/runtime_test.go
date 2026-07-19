package server_test

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"testing"
)

func TestGetRuntime(t *testing.T) {
	e := newAPIEnv(t)
	resp, body := e.do(http.MethodGet, "/api/runtime", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d %s", resp.StatusCode, body)
	}
	var out struct {
		Python3 struct {
			Found   bool   `json:"found"`
			Path    string `json:"path"`
			Version string `json:"version"`
		} `json:"python3"`
		Node struct {
			Found   bool   `json:"found"`
			Path    string `json:"path"`
			Version string `json:"version"`
		} `json:"node"`
		Uvx *struct {
			Found bool `json:"found"`
		} `json:"uvx"`
		Npx *struct {
			Found bool `json:"found"`
		} `json:"npx"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("python3"); err == nil || execLookPathOK("python") {
		if !out.Python3.Found || out.Python3.Path == "" {
			t.Fatalf("expected python3 found: %+v", out.Python3)
		}
	}
	if _, err := exec.LookPath("node"); err == nil {
		if !out.Node.Found || out.Node.Path == "" {
			t.Fatalf("expected node found: %+v", out.Node)
		}
	}
}

func TestInstallRuntimeValidation(t *testing.T) {
	e := newAPIEnv(t)
	resp, body := e.do(http.MethodPost, "/api/runtime/install", map[string]any{"name": "nope"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d %s", resp.StatusCode, body)
	}
}

func execLookPathOK(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
