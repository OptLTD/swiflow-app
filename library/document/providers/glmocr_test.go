package providers

import "testing"

func TestIsGLMOCRModel(t *testing.T) {
	for _, m := range []string{"glm-ocr", "GLM-OCR", "glm-ocr-latest"} {
		if !IsGLMOCRModel(m) {
			t.Fatalf("want glm-ocr match for %q", m)
		}
	}
	if IsGLMOCRModel("glm-4v-flash") {
		t.Fatal("glm-4v should not use layout_parsing")
	}
}

func TestCollectLayoutText(t *testing.T) {
	pages := [][]struct {
		Label   string `json:"label"`
		Content string `json:"content"`
	}{
		{
			{Label: "text", Content: "车牌 宁AP3937"},
			{Label: "table", Content: "<table></table>"},
		},
	}
	got := collectLayoutText(pages)
	if got == "" || !contains(got, "宁AP3937") {
		t.Fatalf("got=%q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
