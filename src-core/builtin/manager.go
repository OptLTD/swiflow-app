package builtin

import (
	"fmt"
	"strings"
	"swiflow/entity"
	"swiflow/model"
	"swiflow/storage"

	"github.com/duke-git/lancet/v2/maputil"
)

type BuiltinTool interface {
	Prompt() string
	Handle(string) (string, error)
}

type BuiltinManager struct {
	tools []BuiltinTool

	config []*entity.CfgEntity
}

var manager *BuiltinManager

func GetManager() *BuiltinManager {
	if manager == nil {
		manager = &BuiltinManager{}
		manager.tools = make([]BuiltinTool, 0)
	}
	return manager
}

func (a *BuiltinManager) Init(store storage.MyStore) *BuiltinManager {
	// Reset tools to avoid duplicates when re-initializing.
	// Intent: make InitTools idempotent across repeated calls.
	a.tools = make([]BuiltinTool, 0)
	query := []string{"type", entity.KEY_CFG_DATA}
	if pools, err := store.LoadCfg(query); err == nil {
		a.config = pools
	}
	a.Append(&CommandTool{})
	a.Append(&Python3Tool{})
	if client := a.buildClient("image-ocr"); client != nil {
		a.Append(&ImageOCRTool{client: client})
	}
	if client := a.buildClient("chat2llm"); client != nil {
		a.Append(&Chat2LLMTool{client: client})
	}
	// Load stored tools and wrap as alias tools
	if list, err := store.LoadTool(); err == nil {
		for _, t := range list {
			base := strings.TrimSpace(t.Code)
			if base == "" {
				base = strings.TrimSpace(t.Name)
			}
			if base == "" {
				continue
			}
			a.Append(&CmdAliasTool{UUID: t.UUID, Name: base})
		}
	}
	return a
}

func (a *BuiltinManager) GetPrompt(tools []string) string {
	var builder strings.Builder
	for _, tool := range a.tools {
		builder.WriteString(tool.Prompt())
	}
	return strings.TrimSpace(builder.String())
}

func (a *BuiltinManager) Append(tools ...BuiltinTool) {
	a.tools = append(a.tools, tools...)
}

func (a *BuiltinManager) Query(name string) (BuiltinTool, error) {
	for _, tool := range a.tools {
		switch tool := tool.(type) {
		case *Chat2LLMTool:
			if name == "chat2llm" {
				return tool, nil
			}
		case *ImageOCRTool:
			if name == "image_ocr" {
				return tool, nil
			}
		case *CommandTool:
			if name == "command" {
				return tool, nil
			}
		case *Python3Tool:
			if name == "python3" {
				return tool, nil
			}
		case *CmdAliasTool:
			if name == tool.UUID {
				return tool, nil
			}
		}
	}
	return nil, fmt.Errorf("no tool found")
}

// buildClient constructs an LLM client from storage cfg-data; falls back to env vars.
func (a *BuiltinManager) buildClient(name string) model.LLMClient {
	var cfg model.LLMConfig
	var find *entity.CfgEntity
	for _, item := range a.config {
		if item.Name == name {
			find = item
			break
		}
	}
	if find == nil {
		return nil
	}
	if m := find.Data; len(m) != 0 {
		_ = maputil.MapTo(m, &cfg)
	}
	if cfg.Provider == "" || cfg.ApiKey == "" {
		return nil
	}
	return model.GetClient(&cfg)
}
