package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/OptLTD/swiflow/library/document"
	"github.com/OptLTD/swiflow/library/document/providers"
	"github.com/OptLTD/swiflow/library/support"
)

const ToolDocumentExtract = "document_extract"

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
	svc     *document.Service
	allowed bool
}

func (t *documentExtractTool) Name() string { return ToolDocumentExtract }
func (t *documentExtractTool) Description() string {
	return "Extract structured JSON from an image, PDF, or text file in the workspace."
}

func (t *documentExtractTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path relative to the workspace.",
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
				"description": "Optional extraction instructions.",
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
	if t.svc == nil {
		return "", fmt.Errorf("document provider not configured")
	}
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
		return "", fmt.Errorf("fields, schema, or prompt required")
	}
	out, err := t.svc.Extract(ctx, req)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// RegisterDocument registers the structured document extraction tool.
func RegisterDocument(r *Registry, ws WorkspaceRoots, opt DocumentOptions) {
	provider := providers.NewOpenAICompatProvider(providers.OpenAICompatConfig{
		BaseURL: opt.BaseURL,
		APIKey:  opt.APIKey,
		Model:   opt.Model,
		Timeout: opt.Timeout,
	})
	svc := document.NewService(provider)
	r.Register(&documentExtractTool{ws: ws, svc: svc, allowed: opt.Enabled})
	if !opt.Enabled {
		r.SetEnabled(ToolDocumentExtract, false)
	}
}
