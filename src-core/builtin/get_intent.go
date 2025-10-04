package builtin

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"swiflow/model"
)

type IntentRequest struct {
	Session string   `json:"session"`
	Content string   `json:"content"`
	Uploads []string `json:"uploads"`
}

type IntentResult struct {
	Intent string `json:"intent"`
	TaskID string `json:"taskid"`
	Worker string `json:"worker"`

	Emoji   string `json:"emoji"`
	Message string `json:"message"`
}

type GetIntentTool struct {
    client model.LLMClient
    prompt string
}

// cleanJSONText removes BOM/zero-width chars and common artifacts that can break JSON decoding.
func cleanJSONText(s string) string {
    s = strings.TrimSpace(s)
    // Remove UTF-8 BOM and zero-width no-break space
    s = strings.ReplaceAll(s, "\ufeff", "")
    // Handle BOM shown as Latin-1 sequence
    s = strings.ReplaceAll(s, "ï»¿", "")
    // Remove stray backslash + BOM sequence if present
    s = strings.ReplaceAll(s, "\\\ufeff", "")
    s = strings.ReplaceAll(s, "\\ï»¿", "")
    return s
}

// extractJSONBlob returns the substring from the first '{' to the last '}' if present.
func extractJSONBlob(s string) string {
    start := strings.Index(s, "{")
    end := strings.LastIndex(s, "}")
    if start >= 0 && end > start {
        return s[start : end+1]
    }
    return s
}

func (a *GetIntentTool) GetIntent(input, events string) (*IntentResult, error) {
    a.prompt += a.Format() + events
    resp, err := a.Handle(input)
    if err != nil {
        return nil, err
    }
    resp = cleanJSONText(resp)
    var data = []byte(resp)
    var result IntentResult
    if err := json.Unmarshal(data, &result); err != nil {
        // Fallback: try extracting a JSON object blob and decode again
        fallback := extractJSONBlob(resp)
        if fallback != resp {
            if err2 := json.Unmarshal([]byte(cleanJSONText(fallback)), &result); err2 == nil {
                return &result, nil
            }
        }
        log.Println("[Intent] decode", resp, err)
        return nil, fmt.Errorf("decode intent error: %w", err)
    }
    return &result, nil
}

func (a *GetIntentTool) Format() string {
	var builder strings.Builder

	builder.WriteString("\n\n---------------\n\n")
	builder.WriteString("# Output Field Explain\n")
	builder.WriteString("- intent: one of options('talk'、'task')\n")
	builder.WriteString("  if task seleted you should pick a worker to resolve the task\n")
	builder.WriteString("  if continue an existing task you should set taskid\n")
	builder.WriteString("  any choice you selected, 'message' will be sent to user\n")
	builder.WriteString("  as a fun reactions, you can use emoji to express yourself\n")
	builder.WriteString("# Output Format Required\n")
	builder.WriteString("```json")
	builder.WriteString(`
    {
        "intent": "talk/task",
        "taskid": "taskid-xx",
        "worker": "workerid",
        "emoji": "SMILE",
        "message": "Hi, Nice To Meet You",
    }`)
	builder.WriteString("\n```")
	return builder.String()
}

func (a *GetIntentTool) Prompt() string {
	return ""
}

func (a *GetIntentTool) Handle(args string) (string, error) {
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
	choices, err := a.client.Respond("get-intent", msgs)
	if err == nil && len(choices) > 0 {
		resp := choices[0].Message.Content
		resp = strings.TrimSpace(resp)
		if strings.HasPrefix(resp, "```json") {
			resp = strings.TrimPrefix(resp, "```json")
		} else if strings.HasPrefix(resp, "```") {
			resp = strings.TrimPrefix(resp, "```")
		}
		resp = strings.TrimSuffix(resp, "```")
		return strings.TrimSpace(resp), nil
	}
	return "", err
}
