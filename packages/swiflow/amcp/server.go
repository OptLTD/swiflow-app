package amcp

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"swiflow/ability"
	"swiflow/config"
	"swiflow/entity"
	"swiflow/support"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/structs"
	"github.com/google/jsonschema-go/jsonschema"
)

var CONNECT_TIMEOUT = 30
var EXECUTE_TIMEOUT = 60

type McpTool struct {
	Name string `json:"name"`
	// See [specification/2025-06-18/basic/index#general-fields] for notes on _meta
	// usage.
	Meta map[string]any `json:"_meta,omitempty"`
	// Optional additional tool information.
	//
	// A human-readable description of the tool.
	//
	// This can be used by clients to improve the LLM's understanding of available
	// tools. It can be thought of like a "hint" to the model.
	Description string `json:"description,omitempty"`
	// A JSON Schema object defining the expected parameters for the tool.
	InputSchema *jsonschema.Schema `json:"inputSchema"`
	// Intended for programmatic or logical use, but used as a display name in past
	// specs or fallback (if title isn't present).
	// An optional JSON Schema object defining the structure of the tool's output
	// returned in the structuredContent field of a CallToolResult.
	OutputSchema *jsonschema.Schema `json:"outputSchema,omitempty"`
	// Intended for UI and end-user contexts — optimized to be human-readable and
	// easily understood, even by those unfamiliar with domain-specific terminology.
	// If not provided, Annotations.Title should be used for display if present,
	// otherwise Name.
	Title string `json:"title,omitempty"`
}
type Resource struct {
	// See [specification/2025-06-18/basic/index#general-fields] for notes on _meta
	// usage.
	Meta map[string]any `json:"_meta,omitempty"`
	// Optional annotations for the client.
	// Annotations *Annotations `json:"annotations,omitempty"`
	// A description of what this resource represents.
	//
	// This can be used by clients to improve the LLM's understanding of available
	// resources. It can be thought of like a "hint" to the model.
	Description string `json:"description,omitempty"`
	// The MIME type of this resource, if known.
	MIMEType string `json:"mimeType,omitempty"`
	// Intended for programmatic or logical use, but used as a display name in past
	// specs or fallback (if title isn't present).
	Name string `json:"name"`
	// The size of the raw resource content, in bytes (i.e., before base64 encoding
	// or any tokenization), if known.
	//
	// This can be used by Hosts to display file sizes and estimate context window
	// usage.
	Size int64 `json:"size,omitempty"`
	// Intended for UI and end-user contexts — optimized to be human-readable and
	// easily understood, even by those unfamiliar with domain-specific terminology.
	//
	// If not provided, the name should be used for display (except for Tool, where
	// Annotations.Title should be given precedence over using name, if
	// present).
	Title string `json:"title,omitempty"`
	// The URI of this resource.
	URI string `json:"uri"`
}

type McpStatus struct {
	// collect error message
	ErrMsg error `json:"error,omitempty"`
	// if this mcp server is active
	Active bool `json:"active,omitempty"`
	Enable bool `json:"enable,omitempty"`

	// then check tools enable, zero means all
	Checked []string `json:"checked,omitempty"`
	// query all mcp tool infomation
	McpTools  []*McpTool  `json:"tools,omitempty"`
	Resources []*Resource `json:"resources,omitempty"`
}

func (s *McpStatus) ToMap() map[string]any {
	data, _ := structs.ToMap(s)
	if data != nil {
		delete(data, "name")
		delete(data, "tools")
	}
	return data
}

type McpServer struct {
	UUID string   `json:"uuid,omitempty"`
	Name string   `json:"name,omitempty"`
	Type string   `json:"type,omitempty"`
	Cmd  string   `json:"command,omitempty"`
	Url  string   `json:"url,omitempty"`
	Args []string `json:"args,omitempty"`

	Env map[string]string `json:"env,omitempty"`

	Status McpStatus `json:"status,omitempty"`
}

func (s *McpServer) FromCfg(v any) error {
	data, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("wrong format")
	}
	for key, val := range data {
		switch key {
		case "active":
			s.Status.Active, _ = val.(bool)
		case "enable":
			s.Status.Enable, _ = val.(bool)
		case "checked":
			if val, ok := val.([]any); ok {
				s.Status.Checked = []string{}
				for _, v := range val {
					s.Status.Checked = append(
						s.Status.Checked, v.(string),
					)
				}
			}
		}
	}
	// 停用的状态也置为不活跃
	if !s.Status.Enable {
		s.Status.Active = false
	}
	return nil
}

func (s *McpServer) FromMap(v any) error {
	data, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("wrong format")
	}
	for key, val := range data {
		switch key {
		case "name", "title":
			s.Name, _ = val.(string)
		case "cmd", "command":
			s.Cmd, _ = val.(string)
		case "type", "protocol":
			s.Type, _ = val.(string)
		case "url":
			s.Url, _ = val.(string)
		case "env":
			if val, ok := val.(map[string]any); ok {
				s.Env = map[string]string{}
				for k, v := range val {
					s.Env[k], _ = v.(string)
				}
			}
		case "args":
			if val, ok := val.([]any); ok {
				s.Args = []string{}
				for _, v := range val {
					s.Args = append(s.Args, v.(string))
				}
			}
		}
	}
	if s.Name == "" {
		s.Name = s.UUID
	}
	return nil
}

func (s *McpServer) GetEnv() []string {
	if len(s.Env) == 0 {
		return nil
	}
	result := make([]string, 0)
	for key, val := range s.Env {
		if val == "$SWIFLOW_HOME" {
			val = config.GetWorkHome()
		}
		if val == "$CURRENT_HOME" {
			val = config.CurrentHome()
		}
		env := fmt.Sprintf("%s=%v", key, val)
		result = append(result, env)
	}
	return result
}

func (s *McpServer) AllEnv() []string {
	if len(s.GetEnv()) == 0 {
		return nil
	}
	set, env := s.GetEnv(), os.Environ()
	return append(env, set...)
}

func (s *McpServer) Inputs() []map[string]any {
	if len(s.Env) == 0 {
		return nil
	}
	return nil
}

func (s *McpServer) ToMap() map[string]any {
	data, _ := structs.ToMap(s)
	if data != nil {
		delete(data, "status")
	}
	return data
}

func (s *McpServer) Checked(bot *entity.BotEntity) []*McpTool {
	if bot.Type == "debug" && bot.UUID != s.UUID {
		return nil
	}
	if bot.Type == "debug" && bot.UUID == s.UUID {
		log.Println("[MCP] #debug#", bot.Tools)
		return s.Status.McpTools
	}

	// 检查是否有全选标记 "server:*"
	allToolsKey := fmt.Sprintf("%s:*", s.UUID)
	if slice.Contain(bot.Tools, allToolsKey) {
		return s.Status.McpTools
	}

	// 检查单个工具
	var result = make([]*McpTool, 0)
	for _, tool := range s.Status.McpTools {
		toolKey := fmt.Sprintf("%s:%s", s.UUID, tool.Name)
		if !slice.Contain(bot.Tools, toolKey) {
			continue
		}
		result = append(result, tool)
	}
	return result
}

func (s *McpServer) Preload() error {
	if s.Cmd == "" {
		return nil // No command to preload
	}

	var pkg = &PackageInfo{}
	err := pkg.ParseCommand(s.Cmd, s.Args)
	if err != nil || pkg.Name == "" {
		log.Printf("[MCP] Command parsing failed (not uvx/npx?): %v", err)
		return nil
	}

	log.Printf("[MCP] Install %s for %s", pkg.Name, s.Name)
	if installed, err := pkg.IsInstalled(); err != nil {
		log.Printf("[MCP] Check package %s is failed: %v", pkg.Name, err)
		return err
	} else if installed {
		return nil
	}

	if err := pkg.Install(); err != nil {
		log.Printf("[MCP] Install package %s: %v", s.Name, err)
		return err
	}
	return nil
}

func (s *McpServer) GetHeaders() map[string]string {
	headers := map[string]string{}
	token := ""
	for key, val := range s.Env {
		switch strings.ToUpper(key) {
		case "API_TOKEN", "API_KEY",
			"TOKEN", "BEARER_TOKEN":
			token = val
			continue
		}
		if support.Bool(val) {
			headers["X-"+key] = val
		}
	}
	if token != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", token)
	}
	return headers
}

func (s *McpServer) GetCmd() (*exec.Cmd, error) {
	cmdPath, err := config.GetMcpEnv(s.Cmd)
	if err != nil {
		return nil, fmt.Errorf("command preparation failed: %v", err)
	}
	dev := new(ability.DevCommandAbility)
	cmd := dev.Cmd(cmdPath, s.Args)
	if env := s.GetEnv(); env != nil {
		if cmd.Env != nil {
			cmd.Env = append(cmd.Env, env...)
		} else {
			cmd.Env = append(os.Environ(), env...)
		}
	}

	if dir := config.GetWorkHome(); dir != "" {
		cmd.Dir = dir
	}
	return cmd, nil
}
