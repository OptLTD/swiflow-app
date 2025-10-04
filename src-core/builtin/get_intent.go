package builtin

import (
	"encoding/xml"
	"fmt"
	"log"
	"strings"
	"swiflow/model"
	"swiflow/support"
)

type IntentRequest struct {
	Session string   `json:"session"`
	Content string   `json:"content"`
	Uploads []string `json:"uploads"`
}

type IntentResult struct {
	XMLName xml.Name `json:"-" xml:"intent"`

	Intent string `json:"intent" xml:"type"`
	TaskID string `json:"taskid" xml:"taskid"`
	Worker string `json:"worker" xml:"worker"`

	Emoji   string `json:"emoji" xml:"emoji"`
	Message string `json:"message" xml:"message"`
}

type GetIntentTool struct {
	client model.LLMClient
	prompt string
}

func (a *GetIntentTool) Prompt() string {
	return ""
}

func (a *GetIntentTool) Handle(args string) (string, error) {
	return "", nil
}

func (a *GetIntentTool) GetIntent(input string, history []model.Message) (*IntentResult, error) {
	resp, err := a.HandleWithHistory(input, history)
	if err != nil {
		return nil, err
	}
	var data = []byte(strings.TrimSpace(resp))
	var result IntentResult
	if err := xml.Unmarshal(data, &result); err != nil {
		fallback := extractXMLBlob(resp)
		if fallback == resp {
			result.Message = resp
			result.Intent = "talk"
			return &result, nil
		}
		data = []byte(strings.TrimSpace(fallback))
		if err2 := xml.Unmarshal(data, &result); err2 == nil {
			return &result, nil
		}
		log.Println("[Intent] decode error", resp, err)
		return nil, fmt.Errorf("decode error: %w", err)
	}
	return &result, nil
}

func (a *GetIntentTool) Format() string {
	var builder strings.Builder

	builder.WriteString("\n\n---------------\n\n")
	builder.WriteString("# Output Field Explain\n")
	builder.WriteString("- intent: one of options(talk/task/wait)\n")
	builder.WriteString("  if task seleted you should pick a worker to resolve the task\n")
	builder.WriteString("  if continue an existing task you should set taskid\n")
	builder.WriteString("  upload files list always after '**UPLOAD FILES**' in user input\n")
	builder.WriteString("  if file、image、upload were mentioned, but no file attached set intent to 'wait'\n")
	builder.WriteString("  diff between talk and task:\n")
	builder.WriteString("    talk: just a simple conversation, no clear intent\n")
	builder.WriteString("    task: very confident to need a worker resolve\n")
	builder.WriteString("    wait: need resolve, but not now, still waiting for user input\n")
	builder.WriteString("  any choice you selected, 'message' will be sent to user\n")
	builder.WriteString("  as a fun reactions, you can use emoji to express yourself\n")
	builder.WriteString("# Output Format Required\n")
	builder.WriteString("```xml")
	builder.WriteString(support.TrimIndent(`
		<intent>
			<type>talk/task/wait</type>
			<taskid>taskid-xx</taskid>
			<worker>workerid</worker>
			<emoji>SMILE</emoji>
			<message>Hi, Nice To Meet You</message>
		</intent>`,
	))
	builder.WriteString("\n```")
	return builder.String()
}

func (a *GetIntentTool) HandleWithHistory(args string, history []model.Message) (string, error) {
	input := strings.TrimSpace(args)
	prompt := a.prompt + a.Format()
	if input == "" || a.prompt == "" {
		return "", fmt.Errorf("no input text provided")
	}
	if a.client == nil {
		return "", fmt.Errorf("no LLM client provided")
	}

	// Build message list:
	// system prompt -> history -> user input
	msgs := make([]model.Message, 0)
	msgs = append(msgs, model.Message{
		Role: "system", Content: prompt,
	})
	msgs = append(msgs, history...)
	msgs = append(msgs, model.Message{
		Role: "user", Content: input,
	})

	choices, err := a.client.Respond("get-intent", msgs)
	if err == nil && len(choices) > 0 {
		resp := choices[0].Message.Content
		return strings.TrimSpace(resp), nil
	}
	return "", err
}

// extractXMLBlob tries to find the first <intent ...>...</intent> block
func extractXMLBlob(s string) string {
	start := strings.Index(s, "<intent")
	if start == -1 {
		return s
	}
	end := strings.LastIndex(s, "</intent>")
	if end == -1 {
		return s
	}
	end += len("</intent>")
	return s[start:end]
}
