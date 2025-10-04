package builtin

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"swiflow/config"
	"swiflow/model"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

type ImageOCRTool struct {
	client model.LLMClient
	prompt string
}

var ocrMutex sync.RWMutex
var ocrCache = make(map[string]string)

func (a *ImageOCRTool) Prompt() string {
	// Build a human-readable usage prompt for the image-ocr builtin tool
	var b strings.Builder
	b.WriteString("### **image-ocr**\n")
	b.WriteString("- 描述：识别图片中的文字并返回清理后的文本\n")
	b.WriteString("- 入参：image path\n")
	b.WriteString("- 示例：\n")
	b.WriteString("```xml\n")
	b.WriteString("<use-builtin-tool>\n")
	b.WriteString("<desc>识别图片文字</desc>\n")
	b.WriteString("<tool>image-ocr</tool>\n")
	b.WriteString("<args>/path/to/image.png</args>\n")
	b.WriteString("</use-builtin-tool>\n")
	b.WriteString("```\n\n")
	return b.String()
}

func (a *ImageOCRTool) Handle(args string) (string, error) {
	// Treat args as image path string; no JSON parsing
	img := strings.TrimSpace(args)
	if img == "" {
		return "", fmt.Errorf("no image path provided")
	}
	if a.client == nil {
		return "", fmt.Errorf("no LLM client provided")
	}
	if !filepath.IsAbs(img) {
		img = filepath.Join(config.CurrentHome(), img)
	}

	// Build OCR instruction as system prompt
	var payload, prompt = "", ""
	if prompt = strings.TrimSpace(a.prompt); prompt == "" {
		prompt = `
			You are an OCR post-processor.
			Clean up and normalize the text.
			Return plain text only without extra commentary.
		`
	}

	// Read image bytes and compute hash for caching
	var buf []byte
	if data, err := os.ReadFile(img); err != nil {
		return "", fmt.Errorf("failed to read image: %v", err)
	} else {
		buf = data
	}
	sum := sha256.Sum256(buf)
	hash := hex.EncodeToString(sum[:])

	// Fast path: cache hit across instances
	ocrMutex.RLock()
	cached := ocrCache[hash]
	if cached != "" {
		ocrMutex.RUnlock()
		return cached, nil
	}
	ocrMutex.RUnlock()

	// Encode image bytes to base64
	payload = base64.StdEncoding.EncodeToString(buf)
	// Encode image bytes to base64 and format as a data URL
	userMsg := model.Message{
		Role: "user", MultiContent: []openai.ChatMessagePart{
			{
				Type: "image_url",
				ImageURL: &openai.ChatMessageImageURL{
					URL: payload, Detail: openai.ImageURLDetailAuto,
				},
			},
			{
				Type: "text",
				Text: prompt,
			},
		},
	}

	msgs := []model.Message{userMsg}
	choices, err := a.client.Respond("image-ocr", msgs)
	if err == nil && len(choices) > 0 {
		resp := choices[0].Message.Content
		resp = strings.TrimSpace(resp)
		if resp != "" {
			ocrMutex.Lock()
			ocrCache[hash] = resp
			ocrMutex.Unlock()
		}
		return resp, nil
	}
	return "", err
}
