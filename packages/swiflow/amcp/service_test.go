package amcp

import (
	"encoding/json"
	"os"
	"testing"
)

// 复制自 client.go，便于测试
func LoadMcpServerConfigs(path string) ([]McpServer, error) {
	type rawServer struct {
		Command  string            `json:"command"`
		Args     []string          `json:"args"`
		Env      map[string]string `json:"env"`
		Protocol string            `json:"protocol"`
	}
	type rawConfig struct {
		McpServers map[string]rawServer `json:"mcpServers"`
	}
	var cfg rawConfig
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	var result []McpServer
	for name, s := range cfg.McpServers {
		result = append(result, McpServer{
			Name: name,
			Cmd:  s.Command,
			Args: s.Args,
			Env:  s.Env,
			Type: s.Protocol,
		})
	}
	return result, nil
}

func TestMcpService_ListTools(t *testing.T) {
	// 加载 servers.json
	servers, err := LoadMcpServerConfigs("./servers.json")
	if err != nil {
		t.Fatalf("加载 servers.json 失败: %v", err)
	}
	if len(servers) == 0 {
		t.Fatalf("未加载到任何 server 配置")
	}

	// 输出加载到的servers详细信息
	for i, s := range servers {
		t.Logf("[server %d] Name: %s, Cmd: %s, Args: %v, Env: %v, Protocol: %s", i, s.Name, s.Cmd, s.Args, s.Env, s.Type)
	}

	// 构造 []*McpServer
	service := &McpService{
		servers: map[string]*McpServer{},
	}
	for i := range servers {
		mcp := servers[i]
		service.servers[mcp.Name] = &mcp
	}

	tools := service.ListTools()
	if len(tools) == 0 {
		t.Errorf("ListTools 未获取到任何工具")
	}
	// 输出tools详细信息
	for i, tool := range tools {
		t.Logf("[tool %d] Name: %s, Detail: %+v", i, tool.Name, tool)
	}
}
