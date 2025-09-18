package initial

import (
	"embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"swiflow/support"
)

//go:embed bots
//go:embed tools
//go:embed script
//go:embed html/*
var fs embed.FS

var basic = []string{
	"basic-tools",
	"file-system",
	"use-command",
}

// leader 拥有交互能力
// 还有支配其他 bot 能力
var leader = []string{
	"basic-tools",
	"file-system",
	"use-subagent",
}

// worker 拥有实战能力
var worker = []string{
	"basic-tools",
	"file-system",
	"use-command",
	"use-mcp-tools",
}

// debug 拥有mcp能力
var debug = []string{
	"use-mcp-tools",
}

type object = map[string]any
type profile struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Desc string `json:"desc"`
	Mcps object `json:"mcps"`
}

func GetBots() []*profile {
	var bots []*profile
	list, _ := fs.ReadDir("bots")
	for _, item := range list {
		if !strings.HasSuffix(item.Name(), ".md") {
			continue
		}

		path := "bots/" + item.Name()
		data, err := fs.ReadFile(path)
		if err != nil {
			continue
		}

		var curr = &profile{Desc: string(data)}
		curr.UUID = strings.TrimSuffix(item.Name(), ".md")
		curr.Name = support.Capitalize(curr.UUID)

		// parse mcps
		mcps := "bots/" + curr.UUID + ".json"
		if data, err := fs.ReadFile(mcps); err == nil {
			var value = map[string]any{}
			json.Unmarshal(data, &value)
			for key, val := range value {
				switch key {
				case "mcp", "servers", "mcpServers":
					curr.Mcps, _ = val.(map[string]any)
				}
			}
		}
		bots = append(bots, curr)
	}
	return bots
}

func GeEmbedFS() *embed.FS {
	return &fs
}

func GetScript(name string) ([]byte, error) {
	list, _ := fs.ReadDir("script")
	for _, item := range list {
		if item.Name() != name {
			continue
		}
		path := "script/" + item.Name()
		return fs.ReadFile(path)
	}
	return nil, fmt.Errorf("%s not found", name)
}

// get tools info
func GetTools() map[string]string {
	var result = map[string]string{}

	list, _ := fs.ReadDir("tools")
	for _, item := range list {
		name := toolName(item.Name())
		if slices.Contains(basic, name) {
			continue
		}
		path := "tools/" + item.Name()
		data, _ := fs.ReadFile(path)
		result[name] = string(data)
	}
	return result
}

// use tool prompt
func UsePrompt(kind string) string {
	ability := []string{}
	switch kind {
	case "leader":
		ability = append(ability, leader...)
	case "worker":
		ability = append(ability, worker...)
	case "debug":
		ability = append(ability, debug...)
	default:
		ability = append(ability, basic...)
	}

	var tools strings.Builder
	list, _ := fs.ReadDir("tools")
	for _, item := range list {
		name := toolName(item.Name())
		if !slices.Contains(ability, name) {
			continue
		}

		dir := "tools/" + item.Name()
		data, _ := fs.ReadFile(dir)
		tools.WriteString(string(data))
	}

	path := fmt.Sprintf("tools/0.%s-prompt.md", kind)
	if data, _ := fs.ReadFile(path); len(data) != 0 {
		return strings.ReplaceAll(string(data), "${{BASE_TOOLS}}", tools.String())
	}

	return ""
}

func toolName(name string) string {
	var parts = strings.Split(name, ".")
	switch len(parts) {
	case 3:
		name = parts[1]
	case 1, 2:
		name = parts[0]
	default:
		return ""
	}
	return name
}
