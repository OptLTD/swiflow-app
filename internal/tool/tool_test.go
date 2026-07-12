package tool

import (
	"testing"
)

func TestAllSortedByName(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&webSearchTool{})
	reg.Register(&webFetchTool{})
	reg.Register(&readFileTool{})

	all := reg.All()
	if len(all) != 3 {
		t.Fatalf("got %d tools", len(all))
	}
	want := []string{"fs_read", "web_fetch", "web_search"}
	for i, name := range want {
		if all[i].Name != name {
			t.Fatalf("all[%d].Name = %q, want %q", i, all[i].Name, name)
		}
	}
}
