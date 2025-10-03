package builtin

import (
	"fmt"
	"strings"
	"swiflow/entity"
	"swiflow/model"
	"swiflow/support"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
)

type BuiltinTool interface {
	Prompt() string
	Handle(string) (string, error)
}

type BuiltinManager struct {
	tools []*entity.ToolEntity
}

var manager *BuiltinManager

func GetManager() *BuiltinManager {
	if manager == nil {
		manager = &BuiltinManager{}
		manager.tools = make([]*entity.ToolEntity, 0)
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

	if strings.EqualFold(name, "image_ocr") {
		client, prompt := a.getLLMClient("image-ocr")
		return &ImageOCRTool{client: client, prompt: prompt}, nil
	}
	return nil, fmt.Errorf("no tool found")
}

func (a *BuiltinManager) GetPrompt(checked []string) string {
	var b strings.Builder
	buildin := []string{
		"command", "python3",
		"chat2llm", "image-ocr",
	}
	// checked builtin:*, includes all
	if slice.Contain(checked, "builtin:*") {
		checked = append(checked, buildin...)
	}
	if slice.Contain(checked, "command") {
		b.WriteString((&CommandTool{}).Prompt())
	}
	if slice.Contain(checked, "python3") {
		b.WriteString((&Python3Tool{}).Prompt())
	}
	if slice.Contain(checked, "chat2llm") {
		b.WriteString((&Chat2LLMTool{}).Prompt())
	}
	if slice.Contain(checked, "image-ocr") {
		b.WriteString((&ImageOCRTool{}).Prompt())
	}
	// Alias prompts derived from tool entities
	allChecked := slice.Contain(checked, "builtin:*")
	for _, ent := range a.tools {
		if slice.Contain(buildin, ent.Type) {
			continue
		}
		selfChecked := slice.Contain(checked, ent.UUID)
		if allChecked || selfChecked {
			alias := &CmdAliasTool{
				UUID: ent.UUID, Name: ent.Name,
				Desc: ent.Desc, Args: "",
			}
			b.WriteString(alias.Prompt())
		}
	}
	pompt := strings.TrimSpace(b.String())
	return support.Or(pompt, "empty list")
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
	return client, tool.Desc
}
