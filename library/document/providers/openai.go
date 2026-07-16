package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/OptLTD/swiflow/library/httputil"
	"github.com/OptLTD/swiflow/library/document"
)

// OpenAICompatConfig configures an OpenAI-compatible multimodal endpoint.
type OpenAICompatConfig struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

// OpenAICompatProvider calls /chat/completions on an OpenAI-compatible API.
type OpenAICompatProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewOpenAICompatProvider constructs a provider for structured extraction.
func NewOpenAICompatProvider(cfg OpenAICompatConfig) *OpenAICompatProvider {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "gpt-4o-mini"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &OpenAICompatProvider{
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		model:   model,
		client:  httputil.Client(timeout),
	}
}

// Extract performs structured extraction and expects a JSON object response.
func (p *OpenAICompatProvider) Extract(ctx context.Context, req document.ProviderRequest) (*document.Result, error) {
	if strings.TrimSpace(p.apiKey) == "" {
		return nil, fmt.Errorf("document api key not configured")
	}
	body := map[string]any{
		"model": p.model,
		"messages": []map[string]any{
			{
				"role":    "system",
				"content": systemPrompt(),
			},
			{
				"role":    "user",
				"content": userContent(req),
			},
		},
		"response_format": map[string]any{
			"type": "json_object",
		},
	}
	raw, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Guard against a server that sends headers then stalls the body.
	respBody := httputil.NewIdleReadCloser(resp.Body, ocrIdleTimeout)
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(respBody)
		return nil, fmt.Errorf("document provider http %d: %s", resp.StatusCode, string(b))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(respBody).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("document provider returned no choices")
	}
	content := strings.TrimSpace(stripCodeFence(out.Choices[0].Message.Content))
	if content == "" {
		return nil, fmt.Errorf("document provider returned empty content")
	}
	return parseExtractJSON(content)
}

func parseExtractJSON(content string) (*document.Result, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("document provider returned invalid json: %w", err)
	}
	var result document.Result
	_ = json.Unmarshal([]byte(content), &result)
	if result.Fields == nil {
		result.Fields = map[string]any{}
	}

	// Promote common alternate shapes models return when no schema was given.
	if len(result.Fields) == 0 {
		if nested, ok := raw["fields"].(map[string]any); ok && len(nested) > 0 {
			result.Fields = nested
		}
	}
	for _, key := range []string{"text", "raw_text", "content", "ocr", "transcription", "全文"} {
		if result.RawText != "" {
			break
		}
		if s, ok := asNonEmptyString(raw[key]); ok {
			result.RawText = s
		}
	}
	if result.RawText == "" {
		if s, ok := asNonEmptyString(result.Fields["text"]); ok {
			result.RawText = s
		} else if s, ok := asNonEmptyString(result.Fields["全文"]); ok {
			result.RawText = s
		}
	}
	if len(result.Fields) == 0 && result.RawText != "" {
		result.Fields["text"] = result.RawText
	}
	if result.DocType == "" {
		if s, ok := asNonEmptyString(raw["doc_type"]); ok {
			result.DocType = s
		} else if s, ok := asNonEmptyString(raw["type"]); ok {
			result.DocType = s
		}
	}
	return &result, nil
}

func asNonEmptyString(v any) (string, bool) {
	s, ok := v.(string)
	s = strings.TrimSpace(s)
	return s, ok && s != ""
}

func systemPrompt() string {
	return strings.Join([]string{
		"You extract structured data from documents and images (OCR).",
		"Return one JSON object only.",
		"Use this shape:",
		`{"doc_type":"...","fields":{...},"confidence":{},"evidence":{},"meta":{}}`,
		"Always fill fields with every readable labeled value from the document (use the printed labels as keys when possible, e.g. 车牌号, 毛重, 净重).",
		"Also put a full plain-text transcription of all visible text into fields[\"text\"].",
		"confidence maps field names to numbers between 0 and 1.",
		"evidence maps field names to short source snippets.",
		"Never return an empty fields object when text is visible — at minimum include fields.text.",
		"If a specific requested field is missing, omit it or set it to null.",
	}, "\n")
}

func userContent(req document.ProviderRequest) any {
	instruction := buildInstruction(req)
	if req.InputType == "img" {
		return []map[string]any{
			{"type": "text", "text": instruction},
			{"type": "image_url", "image_url": map[string]any{"url": req.ImageDataURL}},
		}
	}
	return instruction + "\n\nDocument content:\n" + req.Text
}

func buildInstruction(req document.ProviderRequest) string {
	var b strings.Builder
	b.WriteString("Extract structured fields from this document.\n")
	if req.Prompt != "" {
		b.WriteString("Task: ")
		b.WriteString(req.Prompt)
		b.WriteString("\n")
	} else {
		b.WriteString("Task: Read all visible text and fill fields with every labeled value; include fields.text as the full transcription.\n")
	}
	if len(req.Fields) > 0 {
		b.WriteString("Preferred fields: ")
		b.WriteString(strings.Join(req.Fields, ", "))
		b.WriteString("\n")
	}
	if len(req.Schema) > 0 {
		if raw, err := json.Marshal(req.Schema); err == nil {
			b.WriteString("Target schema JSON: ")
			b.Write(raw)
			b.WriteString("\n")
		}
	}
	b.WriteString("Infer a concise doc_type.\n")
	return b.String()
}

func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	return strings.TrimSpace(s)
}
