package amcp

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"swiflow/ability"
	"swiflow/config"
	"swiflow/entity"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/structs"

	// "github.com/mark3labs/mcp-go/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var CONNECT_TIMEOUT = 30
var EXECUTE_TIMEOUT = 60

type McpTool = mcp.Tool

type McpStatus struct {
	// collect error message
	ErrMsg error `json:"error,omitempty"`
	// if this mcp server is active
	Active bool `json:"active,omitempty"`
	Enable bool `json:"enable,omitempty"`

	// then check tools enable, zero means all
	Checked []string `json:"checked,omitempty"`
	// query all mcp tool infomation
	McpTools []*McpTool `json:"tools,omitempty"`
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
		log.Println("[AGENT] #debug#", bot.Tools)
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

func (s *McpServer) getNpxUvx() (string, error) {
	if s.Cmd != "npx" && s.Cmd != "uvx" {
		return s.Cmd, nil
	}

	if _, err := exec.LookPath(s.Cmd); err == nil {
		checkCmd := exec.Command(s.Cmd, "--version")
		if checkCmd.Run() == nil {
			return s.Cmd, nil
		}
	}

	var localBinPath string // Try ~/.local/bin
	if homeDir, err := os.UserHomeDir(); err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	} else {
		localBinPath = filepath.Join(homeDir, ".local", "bin", s.Cmd)
	}

	if _, err := os.Stat(localBinPath); err != nil {
		return "", fmt.Errorf("%s not found in PATH or ~/.local/bin", s.Cmd)
	}

	// Check if the local binary works
	if checkCmd := exec.Command(localBinPath, "--version"); checkCmd.Run() != nil {
		return "", fmt.Errorf("%s found in ~/.local/bin but version check failed", s.Cmd)
	}

	return localBinPath, nil
}

func (s *McpServer) GetCmd() (*exec.Cmd, error) {
	cmdPath, err := s.getNpxUvx()
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
