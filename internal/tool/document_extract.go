package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/library/document"
	"github.com/OptLTD/swiflow/library/document/providers"
	"github.com/OptLTD/swiflow/library/support"
)

const ToolDocumentExtract = "document_extract"

// Provider names used when DocumentOptions leave credentials empty.
const (
	documentVisionProvider = "vision"
	documentTextProvider   = "default"
)

// IsDocumentTool reports whether name is the document extraction tool.
func IsDocumentTool(name string) bool {
	return name == ToolDocumentExtract
}

// DocumentOptions configures the document extraction tool.
type DocumentOptions struct {
	Enabled   bool
	BaseURL   string
	APIKey    string
	Model     string
	Timeout   time.Duration
	Workspace string
}

type documentExtractTool struct {
	ws      WorkspaceRoots
	st      store.Store
	opt     DocumentOptions
	allowed bool
}

func (t *documentExtractTool) Name() string { return ToolDocumentExtract }
func (t *documentExtractTool) Description() string {
	return "OCR / extract text or structured JSON from one image, PDF, or text file in the workspace. " +
		"For ≥3 attachments or batch table/Excel extraction, the main agent must use delegate_task instead of calling this repeatedly."
}

func (t *documentExtractTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path relative to the workspace (also accepts @/…).",
			},
			"input_type": map[string]any{
				"type":        "string",
				"enum":        []string{"auto", "img", "pdf", "txt"},
				"description": "Optional input type override. Defaults to auto.",
			},
			"fields": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional flat field names to extract.",
			},
			"schema": map[string]any{
				"type":        "object",
				"description": "Optional target schema object for structured extraction.",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "Optional extraction instructions (e.g. extract all visible text).",
			},
			"include_raw_text": map[string]any{
				"type":        "boolean",
				"description": "Include normalized source text in the result when available.",
			},
		},
		"required": []string{"path"},
	}
}

func (t *documentExtractTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if !t.allowed {
		return "", fmt.Errorf("document extraction is disabled")
	}
	baseURL, apiKey, model, err := t.resolveCreds(ctx)
	if err != nil {
		return "", err
	}
	slog.Info("document_extract.model", "model", model, "base_url", baseURL, "api", map[bool]string{true: "layout_parsing", false: "chat/completions"}[providers.IsGLMOCRModel(model)])
	svc := document.NewService(providers.NewDocumentProvider(providers.OpenAICompatConfig{
		BaseURL: baseURL, APIKey: apiKey, Model: model, Timeout: t.opt.Timeout,
	}))

	path, _ := args["path"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path required")
	}
	full, err := support.SandboxPath(t.ws.Base, path)
	if err != nil {
		return "", err
	}
	req := document.Request{
		Path:   full,
		Schema: map[string]any{},
	}
	if inputType, ok := args["input_type"].(string); ok {
		req.InputType = strings.TrimSpace(inputType)
	}
	if prompt, ok := args["prompt"].(string); ok {
		req.Prompt = strings.TrimSpace(prompt)
	}
	if req.InputType == "" {
		req.InputType = "auto"
	}
	if raw, ok := args["include_raw_text"].(bool); ok {
		req.IncludeRawText = raw
	}
	if fields, ok := args["fields"].([]any); ok {
		for _, f := range fields {
			if s, ok := f.(string); ok && strings.TrimSpace(s) != "" {
				req.Fields = append(req.Fields, strings.TrimSpace(s))
			}
		}
	}
	if schema, ok := args["schema"].(map[string]any); ok && len(schema) > 0 {
		req.Schema = schema
	}
	if len(req.Fields) == 0 && len(req.Schema) == 0 && req.Prompt == "" {
		// Default OCR: fill fields with labeled values + full transcription.
		req.Prompt = "Extract every visible labeled field from this document into fields (use Chinese labels as keys when printed). Also set fields.text to the complete plain-text transcription of all visible text."
		req.IncludeRawText = true
	}
	out, err := svc.Extract(ctx, req)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (t *documentExtractTool) resolveCreds(ctx context.Context) (baseURL, apiKey, model string, err error) {
	baseURL = strings.TrimSpace(t.opt.BaseURL)
	apiKey = strings.TrimSpace(t.opt.APIKey)
	model = strings.TrimSpace(t.opt.Model)
	if t.st != nil {
		for _, name := range []string{documentVisionProvider, documentTextProvider} {
			b, k, m, e := t.st.ProviderCreds(ctx, name)
			if e != nil || strings.TrimSpace(k) == "" {
				continue
			}
			if baseURL == "" {
				baseURL = b
			}
			if apiKey == "" {
				apiKey = k
			}
			if model == "" {
				model = m
			}
			break
		}
	}
	if apiKey == "" {
		return "", "", "", fmt.Errorf("document provider not configured (set vision provider in Settings, or tools.document_api_key)")
	}
	return baseURL, apiKey, model, nil
}

// RegisterDocument registers the structured document extraction tool.
// When opt credentials are empty, Execute resolves the vision (then default) provider from st.
func RegisterDocument(r *Registry, ws WorkspaceRoots, st store.Store, opt DocumentOptions) {
	r.Register(&documentExtractTool{ws: ws, st: st, opt: opt, allowed: opt.Enabled})
	if !opt.Enabled {
		r.SetEnabled(ToolDocumentExtract, false)
	}
}
