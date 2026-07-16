package document

import "context"

// Request is a high-level structured extraction request.
type Request struct {
	Path           string
	InputType      string
	Prompt         string
	Fields         []string
	Schema         map[string]any
	IncludeRawText bool
}

// Result is the normalized structured extraction response.
type Result struct {
	InputType  string             `json:"input_type"`
	DocType    string             `json:"doc_type"`
	Fields     map[string]any     `json:"fields"`
	Confidence map[string]float64 `json:"confidence,omitempty"`
	Evidence   map[string]string  `json:"evidence,omitempty"`
	RawText    string             `json:"raw_text,omitempty"`
	Meta       map[string]any     `json:"meta,omitempty"`
}

// ProviderRequest is the low-level request sent to a multimodal provider.
type ProviderRequest struct {
	InputType      string
	Prompt         string
	Schema         map[string]any
	Fields         []string
	Text           string
	ImageDataURL   string
	IncludeRawText bool
}

// Provider extracts structured data from multimodal input.
type Provider interface {
	Extract(ctx context.Context, req ProviderRequest) (*Result, error)
}
