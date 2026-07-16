package providers

import "testing"

func TestParseExtractJSONPromotesText(t *testing.T) {
	out, err := parseExtractJSON(`{"doc_type":"采购计量单","fields":{},"text":"车牌号 宁AP3937\n净重 32.24"}`)
	if err != nil {
		t.Fatal(err)
	}
	if out.DocType != "采购计量单" {
		t.Fatalf("doc_type=%q", out.DocType)
	}
	if out.RawText == "" {
		t.Fatal("expected RawText from top-level text")
	}
	if out.Fields["text"] == nil {
		t.Fatalf("fields=%v", out.Fields)
	}
}

func TestParseExtractJSONKeepsFields(t *testing.T) {
	out, err := parseExtractJSON(`{"doc_type":"slip","fields":{"车牌号":"宁AP3937","净重":"32.24"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if out.Fields["车牌号"] != "宁AP3937" {
		t.Fatalf("fields=%v", out.Fields)
	}
}
