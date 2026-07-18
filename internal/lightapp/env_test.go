package lightapp

import (
	"strings"
	"testing"
)

func TestBuildEnvJS(t *testing.T) {
	js := buildEnvJS(map[string]string{"HELLO_WORLD": "ok", "EMPTY": ""})
	for _, want := range []string{
		"window.swiflow",
		"env: function(key)",
		`"HELLO_WORLD":"ok"`,
		"missing",
		"is empty",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("buildEnvJS missing %q\n%s", want, js)
		}
	}
}

func TestBuildEnvJSNil(t *testing.T) {
	js := buildEnvJS(nil)
	if !strings.Contains(js, "window.swiflow") {
		t.Fatalf("expected swiflow stub, got:\n%s", js)
	}
}

func TestInjectEnvIntoHTML(t *testing.T) {
	src := `<!DOCTYPE html><html><head><title>t</title></head><body><script>window.swiflow.env("X")</script></body></html>`
	envJS := buildEnvJS(map[string]string{"X": "1"})
	out := injectEnvIntoHTML(src, envJS)
	if !strings.Contains(out, envMarker) {
		t.Fatal("missing env marker")
	}
	appIdx := strings.Index(out, `window.swiflow.env("X")`)
	stubIdx := strings.Index(out, envMarker)
	if stubIdx < 0 || appIdx < 0 || stubIdx > appIdx {
		t.Fatalf("stub must precede app script: stub=%d app=%d\n%s", stubIdx, appIdx, out)
	}
	// Idempotent
	out2 := injectEnvIntoHTML(out, envJS)
	if strings.Count(out2, envMarker) != 1 {
		t.Fatalf("expected single inject, got %d", strings.Count(out2, envMarker))
	}
}
