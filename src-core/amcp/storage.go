package amcp

import (
	"errors"
	"swiflow/entity"
	"swiflow/storage"

	"gorm.io/gorm"
)

type McpStorage struct {
	store storage.MyStore
}

func NewMcpStorage(store storage.MyStore) *McpStorage {
	return &McpStorage{store: store}
}

func (m *McpStorage) ParseServers(servers map[string]any) []*McpServer {
	result := []*McpServer{}
	for uuid, val := range servers {
		server := &McpServer{UUID: uuid}
		if server.FromMap(val) == nil {
			result = append(result, server)
		}
	}
	return result
}

func (m *McpStorage) LoadServers() []*McpServer {
	cfg := entity.CfgEntity{
		Type: entity.KEY_MCP_SERVER,
		Name: entity.KEY_MCP_SERVER,
	}
	result := []*McpServer{}
	if err := m.store.FindCfg(&cfg); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			m.store.SaveCfg(&cfg)
		}
		return result
	} else if len(cfg.Data) == 0 {
		return result
	}
	var servers = map[string]any{}
	if data, ok := cfg.Data["servers"]; ok {
		servers, _ = data.(map[string]any)
	}
	var config = map[string]any{}
	if data, ok := cfg.Data["config"]; ok {
		config, _ = data.(map[string]any)
	}
	result = m.ParseServers(servers)
	for _, server := range result {
		if cfg, ok := config[server.UUID]; ok {
			server.FromCfg(cfg)
		}
	}
	return result
}

func (m *McpStorage) DeleteServer(mcp *McpServer) error {
	cfg := entity.CfgEntity{
		Type: entity.KEY_MCP_SERVER,
		Name: entity.KEY_MCP_SERVER,
	}
	_ = m.store.FindCfg(&cfg)
	if cfg.Data == nil {
		return nil
	}
	var servers = map[string]any{}
	if data, ok := cfg.Data["servers"]; ok {
		servers, _ = data.(map[string]any)
	}
	delete(servers, mcp.UUID)
	cfg.Data["servers"] = servers
	return m.store.SaveCfg(&cfg)
}

func (m *McpStorage) UpsertServer(mcp *McpServer) error {
	cfg := entity.CfgEntity{
		Type: entity.KEY_MCP_SERVER,
		Name: entity.KEY_MCP_SERVER,
	}
	_ = m.store.FindCfg(&cfg)
	if len(cfg.Data) == 0 {
		cfg.Data = map[string]any{}
	}
	var servers = map[string]any{}
	if data, ok := cfg.Data["servers"]; ok {
		servers, _ = data.(map[string]any)
	}
	servers[mcp.UUID] = mcp.ToMap()
	cfg.Data["servers"] = servers
	return m.store.SaveCfg(&cfg)
}

func (m *McpStorage) UpsertConfig(mcp *McpServer) error {
	cfg := entity.CfgEntity{
		Type: entity.KEY_MCP_SERVER,
		Name: entity.KEY_MCP_SERVER,
	}
	_ = m.store.FindCfg(&cfg)
	if len(cfg.Data) == 0 {
		cfg.Data = map[string]any{}
	}
	var config = map[string]any{}
	if data, ok := cfg.Data["config"]; ok {
		config, _ = data.(map[string]any)
	}
	status := mcp.Status.ToMap()
	config[mcp.UUID] = status
	cfg.Data["config"] = config
	return m.store.SaveCfg(&cfg)
}
