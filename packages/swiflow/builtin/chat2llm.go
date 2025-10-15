package builtin

import (
	"fmt"
	"strings"
	"swiflow/model"
)

type Chat2LLMTool struct {
	client model.LLMClient
	prompt string
}

func (a *Chat2LLMTool) Prompt() string {
	// Build a human-readable usage prompt for the chat2llm builtin tool
	var b strings.Builder
	b.WriteString("### **chat2llm**\n")
	b.WriteString("- 描述：将输入文本交给LLM处理并返回回答\n")
	b.WriteString("- 入参：text\n")
	b.WriteString("- 示例：\n")
	b.WriteString("```xml\n")
	b.WriteString("<use-builtin-tool>\n")
	b.WriteString("<desc>翻译为英文</desc>\n")
	b.WriteString("<tool>chat2llm</tool>\n")
	b.WriteString("<args>\n")
	b.WriteString("请将下面文本翻译为英文：\n")
	b.WriteString("你好，世界\n")
	b.WriteString("</args>\n")
	b.WriteString("</use-builtin-tool>\n")
	b.WriteString("```\n\n")
	return b.String()
}

func (a *Chat2LLMTool) Handle(args string) (string, error) {
	input := strings.TrimSpace(args)
	prompt := strings.TrimSpace(a.prompt)
	if input == "" {
		return "", fmt.Errorf("no input text provided")
	}
	if a.client == nil {
		return "", fmt.Errorf("no LLM client provided")
	}

	msgs := make([]model.Message, 0, 2)
	if prompt != "" {
		msgs = append(msgs, model.Message{
			Role: "system", Content: prompt,
		})
	}
	msgs = append(msgs, model.Message{
		Role: "user", Content: input,
	})
	// Use a fixed group name for builtin tool calls.
	choices, err := a.client.Respond("chat2llm", msgs)
	if err == nil && len(choices) > 0 {
		resp := choices[0].Message.Content
		return strings.TrimSpace(resp), nil
	}
	return "", err
}
