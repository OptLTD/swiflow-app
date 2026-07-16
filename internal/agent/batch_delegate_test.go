package agent

import (
	"strings"
	"testing"

	"github.com/OptLTD/swiflow/internal/llmclient"
)

func TestShouldForceBatchDelegate(t *testing.T) {
	msg := "[UPLOAD FILES START]\n@/a.png\n@/b.png\n@/c.png\n[UPLOAD FILES END]\nMake a table."
	paths, forced := shouldForceBatchDelegate(msg, false)
	if !forced {
		t.Fatalf("expected forced delegate for %d attachments", len(paths))
	}
	if len(paths) != 3 || paths[0] != "@/a.png" {
		t.Fatalf("paths=%v", paths)
	}

	// Below threshold: not forced.
	if _, forced := shouldForceBatchDelegate("[UPLOAD FILES START]\n@/a.png\n@/b.png\n[UPLOAD FILES END]", false); forced {
		t.Fatal("2 attachments should not force delegate")
	}

	// Children never force (they cannot delegate).
	if _, forced := shouldForceBatchDelegate(msg, true); forced {
		t.Fatal("child run must not force delegate")
	}
}

func TestBatchDelegateNudge(t *testing.T) {
	n := batchDelegateNudge([]string{"@/a.png", "@/b.png", "@/c.png"})
	if !strings.Contains(n, "delegate_task") || !strings.Contains(n, "@/a.png") {
		t.Fatalf("nudge=%q", n)
	}
	if !strings.Contains(n, "DISABLED on the MAIN agent") {
		t.Fatal("nudge should say document_extract is disabled on main")
	}
	if !strings.Contains(n, "3 @/ files") {
		t.Fatalf("nudge should mention count, got %q", n)
	}
}

func TestExtractAtPaths(t *testing.T) {
	got := extractAtPaths("Wrote results to @/out/report.xlsx and @/notes.md.")
	if len(got) != 2 || got[0] != "@/out/report.xlsx" || got[1] != "@/notes.md" {
		t.Fatalf("got=%v", got)
	}
	if len(extractAtPaths("no paths here")) != 0 {
		t.Fatal("expected no paths")
	}
}

func TestArtifactPath(t *testing.T) {
	w := llmclient.ToolCall{Name: "fs_write", Arguments: map[string]any{"path": "out/report.xlsx"}}
	if got := artifactPath(w); got != "@/out/report.xlsx" {
		t.Fatalf("write artifact=%q", got)
	}
	r := llmclient.ToolCall{Name: "fs_read", Arguments: map[string]any{"path": "in.txt"}}
	if got := artifactPath(r); got != "" {
		t.Fatalf("non-writer should have no artifact, got %q", got)
	}
}
