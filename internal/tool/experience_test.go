package tool_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/store/testutil"
	"github.com/OptLTD/swiflow/internal/tool"
)

func makeExperienceRegistry(t *testing.T) (*tool.Registry, store.Store) {
	t.Helper()
	st := testutil.OpenStore(t)
	reg := tool.NewRegistry()
	tool.RegisterExperience(reg, st)
	return reg, st
}

func toolCtx(sessionID, agentKey string) context.Context {
	return tool.WithRunContext(context.Background(), tool.RunContext{
		SessionID: sessionID,
		Agent:  agentKey,
	})
}

func TestExperienceWriteAndList(t *testing.T) {
	reg, _ := makeExperienceRegistry(t)
	ctx := toolCtx("sess-1", "default")

	result, err := reg.Execute(ctx, "experience_write", map[string]any{
		"summary": "Successfully parsed freight Excel with merged headers.",
		"outcome": "success",
		"tags":    []any{"excel", "freight"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result, "experience recorded") {
		t.Fatalf("unexpected result: %s", result)
	}

	listResult, err := reg.Execute(ctx, "experience_list", map[string]any{"limit": float64(10)})
	if err != nil {
		t.Fatal(err)
	}
	var list []store.Experience
	if err := json.Unmarshal([]byte(listResult), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 experience, got %d", len(list))
	}
	if list[0].Summary != "Successfully parsed freight Excel with merged headers." {
		t.Fatalf("unexpected summary: %s", list[0].Summary)
	}
	if len(list[0].Tags) != 2 || list[0].Tags[0] != "excel" {
		t.Fatalf("unexpected tags: %v", list[0].Tags)
	}
}

func TestExperienceWriteDefaultOutcome(t *testing.T) {
	reg, st := makeExperienceRegistry(t)
	ctx := toolCtx("sess-2", "default")

	if _, err := reg.Execute(ctx, "experience_write", map[string]any{
		"summary": "Tried web search but results were unhelpful.",
	}); err != nil {
		t.Fatal(err)
	}

	list, err := st.ListExperiences(context.Background(), "default", 10)
	if err != nil {
		t.Fatal(err)
	}
	if list[0].Outcome != "unknown" {
		t.Fatalf("expected outcome unknown, got %s", list[0].Outcome)
	}
}

func TestExperienceWriteRequiresSummary(t *testing.T) {
	reg, _ := makeExperienceRegistry(t)
	ctx := toolCtx("sess-3", "default")

	_, err := reg.Execute(ctx, "experience_write", map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "summary") {
		t.Fatalf("expected summary required error, got %v", err)
	}
}

func TestExperienceListEmpty(t *testing.T) {
	reg, _ := makeExperienceRegistry(t)
	ctx := toolCtx("sess-4", "default")

	result, err := reg.Execute(ctx, "experience_list", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if result != "[]" {
		t.Fatalf("expected empty list, got %s", result)
	}
}

func TestExperienceListLimitCapped(t *testing.T) {
	reg, _ := makeExperienceRegistry(t)
	ctx := toolCtx("sess-5", "default")

	for i := 0; i < 5; i++ {
		if _, err := reg.Execute(ctx, "experience_write", map[string]any{
			"summary": fmt.Sprintf("reusable tip number %d about freight parsing", i),
		}); err != nil {
			t.Fatal(err)
		}
	}

	result, err := reg.Execute(ctx, "experience_list", map[string]any{"limit": float64(3)})
	if err != nil {
		t.Fatal(err)
	}
	var list []store.Experience
	if err := json.Unmarshal([]byte(result), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3, got %d", len(list))
	}
}

func TestExperienceUseBumpsWeight(t *testing.T) {
	reg, st := makeExperienceRegistry(t)
	ctx := toolCtx("sess-use", "default")

	if _, err := reg.Execute(ctx, "experience_write", map[string]any{
		"summary": "Prefer filename over OCR when weigh-ticket numbers are impossible.",
		"tags":    []any{"ocr"},
	}); err != nil {
		t.Fatal(err)
	}
	list, err := st.ListExperiences(ctx, "default", 5)
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}
	id := list[0].ID
	if list[0].Weight != 1 {
		t.Fatalf("initial weight=%d", list[0].Weight)
	}

	res, err := reg.Execute(ctx, "experience_use", map[string]any{"id": id})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res, `"weight":2`) {
		t.Fatalf("expected weight 2 in %s", res)
	}

	list, err = st.ListExperiences(ctx, "default", 5)
	if err != nil {
		t.Fatal(err)
	}
	if list[0].Weight != 2 {
		t.Fatalf("weight after use=%d", list[0].Weight)
	}
}

func TestExperienceWriteUsedIDs(t *testing.T) {
	reg, st := makeExperienceRegistry(t)
	ctx := toolCtx("sess-used-ids", "default")
	if _, err := reg.Execute(ctx, "experience_write", map[string]any{
		"summary": "first lesson about excel merges",
	}); err != nil {
		t.Fatal(err)
	}
	first, _ := st.ListExperiences(ctx, "default", 1)
	id := first[0].ID

	if _, err := reg.Execute(ctx, "experience_write", map[string]any{
		"summary":  "second lesson about freight sheets",
		"used_ids": []any{id},
	}); err != nil {
		t.Fatal(err)
	}
	list, err := st.ListExperiences(ctx, "default", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2, got %d", len(list))
	}
	// Higher weight should rank first.
	if list[0].ID != id || list[0].Weight != 2 {
		t.Fatalf("expected reused experience first with weight 2, got %+v", list[0])
	}
}

func TestTodoPersistence(t *testing.T) {
	st := testutil.OpenStore(t)
	reg := tool.NewRegistry()
	tool.RegisterTodo(reg, st)

	ctx := toolCtx("sess-todo", "default")
	writeResult, err := reg.Execute(ctx, "todo_write", map[string]any{
		"items": []any{
			map[string]any{"text": "step one", "done": false},
			map[string]any{"text": "step two", "done": true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var written []map[string]any
	if err := json.Unmarshal([]byte(writeResult), &written); err != nil {
		t.Fatal(err)
	}
	if len(written) != 2 {
		t.Fatalf("expected 2 items, got %d", len(written))
	}

	readResult, err := reg.Execute(ctx, "todo_read", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	var read []map[string]any
	if err := json.Unmarshal([]byte(readResult), &read); err != nil {
		t.Fatal(err)
	}
	if len(read) != 2 || read[0]["text"] != "step one" {
		t.Fatalf("unexpected read result: %s", readResult)
	}
}

func TestTodoReadEmptySession(t *testing.T) {
	st := testutil.OpenStore(t)
	reg := tool.NewRegistry()
	tool.RegisterTodo(reg, st)

	ctx := toolCtx("sess-new", "default")
	result, err := reg.Execute(ctx, "todo_read", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if result != "[]" {
		t.Fatalf("expected [], got %s", result)
	}
}
