package amcp

import (
	"testing"

	"swiflow/entity"
	"swiflow/storage"
)

func newTestMcpStorage(cfgData map[string]any) *McpStorage {
	mock := storage.NewMockStore()
	cfg := &entity.CfgEntity{
		Type: entity.KEY_MCP_SERVER,
		Name: entity.KEY_MCP_SERVER,
		Data: cfgData,
	}
	mock.SetCfgs([]*entity.CfgEntity{cfg})
	return &McpStorage{store: mock}
}

func TestMcpStorage_LoadServer(t *testing.T) {
	serversMap := map[string]any{
		"test1": map[string]any{
			"type":    "test-type",
			"command": "echo",
			"url":     "http://localhost",
			"args":    []string{"a", "b"},
		},
	}
	configData := map[string]any{"servers": serversMap}
	storage := newTestMcpStorage(configData)
	servers := storage.LoadServers()
	if len(servers) != 1 {
		t.Fatalf("期望1个server，实际%v", len(servers))
	}
	if servers[0].Name != "test1" || servers[0].Cmd != "echo" {
		t.Errorf("server内容不符: %+v", servers[0])
	}
}

func TestMcpStorage_CreateServer(t *testing.T) {
	storage := newTestMcpStorage(map[string]any{"servers": map[string]any{}})
	mcp := &McpServer{
		Name: "test2",
		Type: "type2",
		Cmd:  "ls",
		Url:  "http://test2",
		Args: []string{"-l"},
	}
	err := storage.UpsertServer(mcp)
	if err != nil {
		t.Fatalf("CreateServer 错误: %v", err)
	}
	servers := storage.LoadServers()
	found := false
	for _, s := range servers {
		if s.Name == "test2" && s.Cmd == "ls" {
			found = true
		}
	}
	if !found {
		t.Logf("[all servers] %v", servers[0])
		t.Errorf("CreateServer 未找到新建server")
	}
}

func TestMcpStorage_DeleteServer(t *testing.T) {
	serversMap := map[string]any{
		"test3": map[string]any{"type": "t3", "command": "pwd"},
	}
	storage := newTestMcpStorage(map[string]any{"servers": serversMap})
	mcp := &McpServer{Name: "test3"}
	err := storage.DeleteServer(mcp)
	if err != nil {
		t.Fatalf("DeleteServer 错误: %v", err)
	}
	servers := storage.LoadServers()
	for _, s := range servers {
		if s.Name == "test3" {
			t.Errorf("DeleteServer 未删除server")
		}
	}
}

func TestMcpStorage_UpdateServer(t *testing.T) {
	serversMap := map[string]any{
		"test4": map[string]any{"type": "t4", "command": "old"},
	}
	storage := newTestMcpStorage(map[string]any{"servers": serversMap})
	mcp := &McpServer{Name: "test4", Cmd: "new-cmd", Type: "t4"}
	err := storage.UpsertServer(mcp)
	if err != nil {
		t.Fatalf("UpdateServer 错误: %v", err)
	}
	servers := storage.LoadServers()
	found := false
	for _, s := range servers {
		if s.Name == "test4" && s.Cmd == "new-cmd" {
			found = true
		}
	}
	if !found {
		t.Errorf("UpdateServer 未更新server")
	}
}
