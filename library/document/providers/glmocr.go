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

// GLMOCRProvider calls Zhipu's layout_parsing API (glm-ocr), not chat/completions.
type GLMOCRProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// ocrIdleTimeout bounds a stalled response body (headers received, no data).
const ocrIdleTimeout = 60 * time.Second

// IsGLMOCRModel reports whether the configured model should use layout_parsing.
func IsGLMOCRModel(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	return m == "glm-ocr" || strings.Contains(m, "glm-ocr")
}

// NewGLMOCRProvider constructs a glm-ocr layout_parsing client.
func NewGLMOCRProvider(cfg OpenAICompatConfig) *GLMOCRProvider {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://open.bigmodel.cn/api/paas/v4"
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "glm-ocr"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	return &GLMOCRProvider{
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		model:   model,
		client:  httputil.Client(timeout),
	}
}

func (p *GLMOCRProvider) Extract(ctx context.Context, req document.ProviderRequest) (*document.Result, error) {
	if strings.TrimSpace(p.apiKey) == "" {
		return nil, fmt.Errorf("document api key not configured")
	}
	file := strings.TrimSpace(req.ImageDataURL)
	if file == "" {
		return nil, fmt.Errorf("glm-ocr requires image/pdf file input (got empty file payload)")
	}
	body := map[string]any{
		"model": p.model,
		"file":  file,
	}
	raw, _ := json.Marshal(body)
	url := p.baseURL + "/layout_parsing"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
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
	b, _ := io.ReadAll(respBody)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("document provider http %d: %s", resp.StatusCode, string(b))
	}
	var out struct {
		ID         string `json:"id"`
		MDResults  string `json:"md_results"`
		LayoutDetails [][]struct {
			Label   string `json:"label"`
			Content string `json:"content"`
		} `json:"layout_details"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("glm-ocr response decode: %w", err)
	}
	md := strings.TrimSpace(out.MDResults)
	if md == "" {
		md = collectLayoutText(out.LayoutDetails)
	}
	if md == "" {
		return nil, fmt.Errorf("glm-ocr returned empty text")
	}
	result := &document.Result{
		DocType: "document",
		Fields:  map[string]any{"text": md},
		RawText: md,
		Meta: map[string]any{
			"provider": "glm-ocr",
			"task_id":  out.ID,
		},
	}
	if req.Prompt != "" {
		result.Meta["prompt"] = req.Prompt
	}
	return result, nil
}

func collectLayoutText(pages [][]struct {
	Label   string `json:"label"`
	Content string `json:"content"`
}) string {
	var b strings.Builder
	for _, page := range pages {
		for _, el := range page {
			if el.Label != "text" && el.Label != "table" && el.Label != "formula" {
				continue
			}
			c := strings.TrimSpace(el.Content)
			if c == "" {
				continue
			}
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(c)
		}
	}
	return b.String()
}

// NewDocumentProvider picks glm-ocr layout_parsing or OpenAI-compatible chat/completions.
func NewDocumentProvider(cfg OpenAICompatConfig) document.Provider {
	if IsGLMOCRModel(cfg.Model) {
		return NewGLMOCRProvider(cfg)
	}
	return NewOpenAICompatProvider(cfg)
}
