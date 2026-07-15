package sqlite_test

import (
	"context"
	"testing"

	"github.com/OptLTD/swiflow/internal/store"
)

func TestCreateAndListExperiences(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()

	if err := st.CreateExperience(ctx, &store.Experience{
		Sid: "s1", Agent: "default",
		Summary: "Parsed an Excel file with merged cells.", Outcome: "success",
		Tags: []string{"excel", "parsing"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateExperience(ctx, &store.Experience{
		Sid: "s2", Agent: "default",
		Summary: "Browser navigation timed out.", Outcome: "failure",
		Tags: []string{"browser"},
	}); err != nil {
		t.Fatal(err)
	}
	// Different agent — must not appear in results.
	if err := st.CreateExperience(ctx, &store.Experience{
		Sid: "s3", Agent: "other",
		Summary: "Should not appear.", Outcome: "success",
	}); err != nil {
		t.Fatal(err)
	}

	list, err := st.ListExperiences(ctx, "default", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}
	// Results are newest-first.
	if list[0].Outcome != "failure" {
		t.Fatalf("expected failure first, got %s", list[0].Outcome)
	}
	if len(list[1].Tags) != 2 || list[1].Tags[0] != "excel" {
		t.Fatalf("unexpected tags: %v", list[1].Tags)
	}
}

func TestListExperiencesLimit(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if err := st.CreateExperience(ctx, &store.Experience{
			Sid: "s", Agent: "default",
			Summary: "item", Outcome: "unknown",
		}); err != nil {
			t.Fatal(err)
		}
	}

	list, err := st.ListExperiences(ctx, "default", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3, got %d", len(list))
	}
}

func TestDeleteExperience(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()

	e := &store.Experience{
		Sid: "s1", Agent: "default",
		Summary: "temp", Outcome: "unknown",
	}
	if err := st.CreateExperience(ctx, e); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteExperience(ctx, e.ID); err != nil {
		t.Fatal(err)
	}
	list, err := st.ListExperiences(ctx, "default", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 after delete, got %d", len(list))
	}
}

func TestSaveAndLoadTodos(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()

	const items = `[{"id":"1","text":"write tests","done":false}]`
	if err := st.SaveTodos(ctx, "sess-a", items); err != nil {
		t.Fatal(err)
	}
	got, err := st.LoadTodos(ctx, "sess-a")
	if err != nil {
		t.Fatal(err)
	}
	if got != items {
		t.Fatalf("got %q, want %q", got, items)
	}
}

func TestLoadTodosMissing(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()

	got, err := st.LoadTodos(ctx, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if got != "[]" {
		t.Fatalf("expected [] for missing session, got %q", got)
	}
}

func TestSaveTodosUpsert(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()

	if err := st.SaveTodos(ctx, "sess-b", `[{"id":"1","text":"old","done":false}]`); err != nil {
		t.Fatal(err)
	}
	const updated = `[{"id":"1","text":"new","done":true}]`
	if err := st.SaveTodos(ctx, "sess-b", updated); err != nil {
		t.Fatal(err)
	}
	got, err := st.LoadTodos(ctx, "sess-b")
	if err != nil {
		t.Fatal(err)
	}
	if got != updated {
		t.Fatalf("upsert failed: got %q", got)
	}
}
