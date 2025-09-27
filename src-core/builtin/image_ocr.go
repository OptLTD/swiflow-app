package builtin

import (
	"fmt"
	"strings"
	"swiflow/config"
	"swiflow/model"
)

type ImageOCRTool struct {
	client model.LLMClient
	prompt string
}

func (a *ImageOCRTool) Prompt() string {
	// Build a human-readable usage prompt for the image_ocr builtin tool
	var b strings.Builder
	b.WriteString("### **image_ocr**\n")
	b.WriteString("- 描述：识别图片中的文字并返回清理后的文本\n")
	b.WriteString("- 入参：image path\n")
	b.WriteString("- 示例：\n")
	b.WriteString("```xml\n")
	b.WriteString("<use-builtin-tool>\n")
	b.WriteString("<desc>识别图片文字</desc>\n")
	b.WriteString("<tool>image_ocr</tool>\n")
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

	// Determine working directory
	home := config.CurrentHome()
	if strings.TrimSpace(home) == "" {
		home = config.GetWorkHome()
	}

	// image -> base64
	base64, prompt := "", ""
	if prompt = strings.TrimSpace(a.prompt); prompt == "" {
		prompt = `
			You are an OCR post-processor. 
			Clean up and normalize the text. 
			Return plain text only without extra commentary.
		`
	}
	msgs := []model.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: base64},
	}
	choices, err := a.client.Respond("image_ocr", msgs)
	if err == nil && len(choices) > 0 {
		resp := choices[0].Message.Content
		return strings.TrimSpace(resp), nil
	}
	return "", err
}
