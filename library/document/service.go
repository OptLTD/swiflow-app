package document

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	pdf "github.com/ledongthuc/pdf"
)

const maxTextBytes = 512 * 1024

// Service routes files to the appropriate extractor and provider.
type Service struct {
	provider Provider
}

// NewService constructs a document extraction service.
func NewService(provider Provider) *Service {
	return &Service{provider: provider}
}

// Extract performs structured extraction for a local file.
func (s *Service) Extract(ctx context.Context, req Request) (*Result, error) {
	if s == nil || s.provider == nil {
		return nil, fmt.Errorf("document provider not configured")
	}
	if strings.TrimSpace(req.Path) == "" {
		return nil, fmt.Errorf("path required")
	}
	inputType := DetectInputType(req.Path, req.InputType)
	if inputType == "" {
		return nil, fmt.Errorf("unsupported document type")
	}

	providerReq := ProviderRequest{
		InputType:      inputType,
		Prompt:         strings.TrimSpace(req.Prompt),
		Schema:         req.Schema,
		Fields:         append([]string(nil), req.Fields...),
		IncludeRawText: req.IncludeRawText,
	}

	switch inputType {
	case "img":
		dataURL, err := imageDataURL(req.Path)
		if err != nil {
			return nil, err
		}
		providerReq.ImageDataURL = dataURL
	case "txt":
		text, err := readTextFile(req.Path)
		if err != nil {
			return nil, err
		}
		providerReq.Text = text
	case "pdf":
		text, err := readPDFText(req.Path)
		if err != nil {
			return nil, err
		}
		providerReq.Text = text
	default:
		return nil, fmt.Errorf("unsupported input_type %q", inputType)
	}

	out, err := s.provider.Extract(ctx, providerReq)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, fmt.Errorf("empty document result")
	}
	out.InputType = inputType
	if out.Fields == nil {
		out.Fields = map[string]any{}
	}
	if out.Confidence == nil {
		out.Confidence = map[string]float64{}
	}
	if out.Evidence == nil {
		out.Evidence = map[string]string{}
	}
	if out.Meta == nil {
		out.Meta = map[string]any{}
	}
	// For txt/pdf, source text is local; for images, RawText comes from the model when present.
	if req.IncludeRawText {
		if out.RawText == "" && providerReq.Text != "" {
			out.RawText = providerReq.Text
		}
		if out.RawText == "" {
			if s, ok := out.Fields["text"].(string); ok && strings.TrimSpace(s) != "" {
				out.RawText = strings.TrimSpace(s)
			}
		}
	}
	return out, nil
}

func readTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(data) > maxTextBytes {
		data = data[:maxTextBytes]
	}
	return string(data), nil
}

func readPDFText(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	reader, err := r.GetPlainText()
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, io.LimitReader(reader, maxTextBytes)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func imageDataURL(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("empty image file")
	}
	ext := strings.ToLower(path[strings.LastIndex(path, ".")+1:])
	mime := "image/png"
	switch ext {
	case "jpg", "jpeg":
		mime = "image/jpeg"
	case "webp":
		mime = "image/webp"
	case "gif":
		mime = "image/gif"
	case "bmp":
		mime = "image/bmp"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}
