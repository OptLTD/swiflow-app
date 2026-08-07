package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/OptLTD/swiflow/internal/store/testutil"
)

func TestAppendCharterPrinciple(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	ctx := context.Background()
	ag, err := st.GetAgentByKey(ctx, "default")
	if err != nil {
		t.Fatal(err)
	}
	appendCharterPrinciple(ctx, st, "sess", ag, "以后都用 gbk 读 csv", "correction")
	ag2, err := st.GetAgentByKey(ctx, "default")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ag2.Charter, "以后都用 gbk") {
		t.Fatalf("charter not updated: %q", ag2.Charter)
	}
	// idempotent
	appendCharterPrinciple(ctx, st, "sess", ag2, "以后都用 gbk 读 csv", "correction")
	ag3, _ := st.GetAgentByKey(ctx, "default")
	if strings.Count(ag3.Charter, "以后都用 gbk") != 1 {
		t.Fatalf("expected single append, got %q", ag3.Charter)
	}
}
