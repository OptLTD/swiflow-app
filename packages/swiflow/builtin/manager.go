package builtin

import (
	"fmt"
	"log"
	"strings"
	"swiflow/entity"
	"swiflow/model"
	"swiflow/support"
	"time"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
)

type BuiltinTool interface {
	Prompt() string
	Handle(string) (string, error)
}

type BuiltinManager struct {
	tools []*entity.ToolEntity
	// cache merged IntentRequest per session
	cacheReq  map[string]*IntentRequest
	cacheTime map[string]time.Time
}

var manager *BuiltinManager

func GetManager() *BuiltinManager {
	if manager == nil {
		manager = &BuiltinManager{}
		manager.tools = make([]*entity.ToolEntity, 0)
		manager.cacheTime = make(map[string]time.Time)
		manager.cacheReq = make(map[string]*IntentRequest)
	}
	return manager
}

func (a *BuiltinManager) Init(tools []*entity.ToolEntity) *BuiltinManager {
	src := support.Or(tools, []*entity.ToolEntity{})
	findBy := func(name string) (int, *entity.ToolEntity) {
		for i, t := range src {
			if strings.EqualFold(t.UUID, name) {
				return i, t
			}
		}
		return -1, nil
	}

	// Ensure builtins at head in required order
	used := make(map[int]bool)
	head := make([]*entity.ToolEntity, 0, 4)
	ensure := func(uuidType string, desc string) {
		if idx, t := findBy(uuidType); t != nil {
			head = append(head, t)
			used[idx] = true
			return
		}
		head = append(head, &entity.ToolEntity{
			UUID: uuidType, Type: uuidType,
			Name: uuidType, Desc: desc,
		})
	}
	ensure("command", "command tool")
	ensure("python3", "python3 tool")
	ensure("chat2llm", "chat2llm tool")
	ensure("image-ocr", "image-ocr tool")
	ensure("get-intent", "image-ocr tool")

	// Append remaining tools preserving order
	tail := make([]*entity.ToolEntity, 0, len(src))
	for i, t := range src {
		if used[i] {
			continue
		}
		tail = append(tail, t)
	}

	a.tools = append(head, tail...)
	return a
}

func (a *BuiltinManager) GetList() []*entity.ToolEntity {
	var hidden = []string{"get-intent"}
	return slice.Filter(a.tools, func(idx int, t *entity.ToolEntity) bool {
		return !slice.Contain(hidden, t.UUID)
	})
}

func (a *BuiltinManager) AllTools() []*entity.ToolEntity {
	return a.tools
}

func (a *BuiltinManager) Query(name string) (BuiltinTool, error) {
	if strings.EqualFold(name, "command") {
		return &CommandTool{}, nil
	}
	if strings.EqualFold(name, "python3") {
		return &Python3Tool{}, nil
	}
	if strings.EqualFold(name, "get-intent") {
		client, prompt := a.getLLMClient("get-intent")
		return &GetIntentTool{client: client, prompt: prompt}, nil
	}
	if strings.EqualFold(name, "chat2llm") {
		client, prompt := a.getLLMClient("chat2llm")
		return &Chat2LLMTool{client: client, prompt: prompt}, nil
	}

	if strings.EqualFold(name, "image-ocr") {
		client, prompt := a.getLLMClient("image-ocr")
		return &ImageOCRTool{client: client, prompt: prompt}, nil
	}
	for _, tool := range a.tools {
		if tool.UUID != name {
			continue
		}
		switch tool.Type {
		case "cmd-alias":
			cmdAlias := &CmdAliasTool{
				UUID: tool.UUID, Name: tool.Name,
				Desc: tool.Desc, Args: "",
			}
			return cmdAlias, nil
		case "py3-alias":
			py3Alias := &Py3AliasTool{
				UUID: tool.UUID, Name: tool.Name,
				Desc: tool.Desc, Code: tool.Text,
			}
			return py3Alias, nil
		}
	}
	return nil, fmt.Errorf("no tool found")
}

func (a *BuiltinManager) GetPrompt(checked []string) string {
	var b strings.Builder
	buildin := []string{
		"builtin:command", "builtin:python3",
		"builtin:chat2llm", "builtin:image-ocr",
	}
	// checked builtin:*, includes all
	if slice.Contain(checked, "builtin:*") {
		checked = append(checked, buildin...)
	}
	if slice.Contain(checked, "builtin:command") {
		b.WriteString((&CommandTool{}).Prompt())
	}
	if slice.Contain(checked, "builtin:python3") {
		b.WriteString((&Python3Tool{}).Prompt())
	}
	if slice.Contain(checked, "builtin:chat2llm") {
		b.WriteString((&Chat2LLMTool{}).Prompt())
	}
	if slice.Contain(checked, "builtin:image-ocr") {
		b.WriteString((&ImageOCRTool{}).Prompt())
	}
	// Alias prompts derived from tool entities
	allChecked := slice.Contain(checked, "builtin:*")
	for _, tool := range a.tools {
		if slice.Contain(buildin, tool.Type) {
			continue
		}
		selfChecked := slice.Contain(checked, tool.UUID)
		if allChecked || selfChecked {
			switch tool.Type {
			case "cmd-alias":
				cmdAlias := &CmdAliasTool{
					UUID: tool.UUID, Name: tool.Name,
					Desc: tool.Desc, Args: "",
				}
				b.WriteString(cmdAlias.Prompt())
			case "py3-alias":
				py3Alias := &Py3AliasTool{
					UUID: tool.UUID, Name: tool.Name,
					Desc: tool.Desc, Code: tool.Text,
				}
				b.WriteString(py3Alias.Prompt())
			}
		}
	}
	pompt := strings.TrimSpace(b.String())
	return support.Or(pompt, "empty list")
}

// GetIntent recognizes intent from user content and optional uploads.
// 1) If uploads empty, use GetIntentTool
// 2) If uploads present, OCR images first (limit concurrency 4) then feed merged text
// 3) Cache merged IntentRequest for 5 minutes per session to handle split IM messages
// History messages are provided for better context.
func (a *BuiltinManager) GetIntent(req *IntentRequest, history []model.Message) (*IntentResult, error) {
	requestStart := time.Now()

	var intentTool *GetIntentTool
	if tool, err := a.Query("get-intent"); err != nil {
		return nil, fmt.Errorf("get intent tool error: %w", err)
	} else if tool, ok := tool.(*GetIntentTool); !ok {
		return nil, fmt.Errorf("get intent tool type error")
	} else {
		intentTool = tool
	}

	content := strings.TrimSpace(req.Content)
	if len(req.Uploads) == 0 {
		intentStart := time.Now()
		res, err := intentTool.GetIntent(content, history)
		intentMs := time.Since(intentStart).Milliseconds()
		totalMs := time.Since(requestStart).Milliseconds()
		log.Printf("[Intent] timing: ocr=%dms intent=%dms total=%dms", 0, intentMs, totalMs)
		return res, err
	}

	// OCR images first
	var imageOcrTool *ImageOCRTool
	if ocr, err := a.Query("image-ocr"); err != nil {
		return nil, fmt.Errorf("get image ocr tool error: %w", err)
	} else if ocrTool, ok := ocr.(*ImageOCRTool); !ok {
		return nil, fmt.Errorf("image ocr tool type error")
	} else {
		imageOcrTool = ocrTool
	}

	type result struct {
		idx  int
		text string
		err  error
	}
	started, ocrStart := 0, time.Now()
	sem := make(chan struct{}, 4)
	done := make(chan result, len(req.Uploads))
	results := make([]string, len(req.Uploads))
	for idx, p := range req.Uploads {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		started++
		sem <- struct{}{}
		i := idx
		go func(path string, id int) {
			defer func() { <-sem }()
			txt, e := imageOcrTool.Handle(path)
			done <- result{idx: id, text: txt, err: e}
		}(p, i)
	}
	// collect
	for i := 0; i < started; i++ {
		r := <-done
		if r.err == nil && strings.TrimSpace(r.text) != "" {
			results[r.idx] = strings.TrimSpace(r.text)
		}
	}
	ocrMs := time.Since(ocrStart).Milliseconds()

	// if no input present, almost upload image
	if content == "" && len(req.Uploads) == 1 {
		log.Printf("[Intent] timing: ocr=%dms", ocrMs)
		return &IntentResult{
			Intent: "image-ocr", Message: results[0],
		}, nil
	}

	// merge content and ocr
	var b strings.Builder
	b.WriteString(content)
	if started > 0 {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("**UPLOAD FILES**")
		for idx, t := range results {
			if t == "" {
				continue
			}
			b.WriteString(fmt.Sprintf(
				"- File Name **%s**\n```",
				req.Uploads[idx],
			))
			b.WriteString("**OCR Result**")
			b.WriteString(fmt.Sprintf(
				"\n```text\n%s\n```\n\n", t,
			))
		}
	}
	merged := b.String()
	intentStart := time.Now()
	res, err := intentTool.GetIntent(merged, history)
	intentMs := time.Since(intentStart).Milliseconds()
	totalMs := time.Since(requestStart).Milliseconds()
	log.Printf("[Intent] timing: ocr=%dms intent=%dms total=%dms", ocrMs, intentMs, totalMs)
	return res, err
}

func (a *BuiltinManager) GenerateCode(client model.LLMClient, desc string) (string, error) {
	if strings.TrimSpace(desc) == "" {
		return "", fmt.Errorf("desc is empty")
	}

	prompt := "你要根据用户的诉求，生成对应的Python3代码。"
	prompt += "不要输出其他额外内容，只输出Python3代码。"
	prompt += "Python代码中的相关数据要从参数中读取，建议使用typer库。"
	choices, err := client.Respond(
		"gen-code", []model.Message{
			{Role: "system", Content: prompt},
			{Role: "user", Content: desc},
		},
	)
	if err == nil && len(choices) > 0 {
		data := choices[0].Message.Content
		text := strings.TrimSpace(data)
		text = strings.TrimPrefix(text, "```python")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		return strings.TrimSpace(text), nil
	}
	return "", err
}

func (a *BuiltinManager) findTool(name string) *entity.ToolEntity {
	for _, t := range a.tools {
		if strings.EqualFold(t.UUID, name) {
			return t
		}
	}
	return nil
}

func (a *BuiltinManager) getLLMClient(name string) (model.LLMClient, string) {
	cfg := model.LLMConfig{}
	tool := a.findTool(name)
	if tool != nil && tool.Data != nil {
		_ = maputil.MapTo(tool.Data, &cfg)
	}
	// Build client only when provider is set.
	var client model.LLMClient
	if strings.TrimSpace(cfg.Provider) != "" {
		client = model.GetClient(&cfg)
	}
	return client, tool.Text
}
