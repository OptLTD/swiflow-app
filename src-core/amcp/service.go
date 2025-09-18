package amcp

import (
	"fmt"
	"swiflow/entity"
	"swiflow/storage"
	"sync"
)

type McpService struct {
	storage *McpStorage
	mockdb  *storage.MockStore
	servers map[string]*McpServer
	mu      sync.RWMutex
}

var service *McpService

func NewMcpService(store storage.MyStore) *McpService {
	return &McpService{
		storage: NewMcpStorage(store),
		servers: map[string]*McpServer{},
	}
}
func GetMcpService(store storage.MyStore) *McpService {
	if service == nil {
		service = NewMcpService(store)
	}
	return service
}

func (m *McpService) GetMcpClient(server *McpServer) *McpClient {
	return NewMcpClient(server)
}

func (m *McpService) ServerClose(server *McpServer) error {
	client := NewMcpClient(server)
	if client == nil {
		return nil
	}
	return client.Close()
}

func (m *McpService) ServerStatus(server *McpServer) error {
	client := NewMcpClient(server)
	if client == nil {
		server.Status.Active = false
		err := server.Status.ErrMsg
		return fmt.Errorf("error: %v", err)
	}
	server.Status.Checked = []string{}
	server.Status.McpTools = []*McpTool{}
	toolResult, err := client.ListTools()
	if toolResult == nil || err != nil {
		return fmt.Errorf("error: %v", err)
	}
	for _, tool := range toolResult {
		server.Status.McpTools = append(server.Status.McpTools, tool)
		server.Status.Checked = append(server.Status.Checked, tool.Name)
	}
	server.Status.Active = true
	return nil
}

func (m *McpService) ParseServer(data map[string]any) *McpServer {
	servers := m.storage.ParseServers(data)
	if len(servers) == 0 {
		return nil
	}
	return servers[0]
}

func (m *McpService) ListServers() []*McpServer {
	servers := m.storage.LoadServers()
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, mcp := range servers {
		origin := m.servers[mcp.UUID]
		if origin != nil {
			mcp.Status = origin.Status
		}
		m.servers[mcp.UUID] = mcp
	}
	return servers
}

func (m *McpService) ListTools() []*McpTool {
	var allTools []*McpTool
	for _, server := range m.servers {
		if !server.Status.Active {
			continue
		}

		if client := NewMcpClient(server); client == nil {
			continue
		} else if tools, err := client.ListTools(); err != nil {
			continue
		} else if len(tools) > 0 {
			allTools = tools
		}
	}
	return allTools
}

func (m *McpService) GetMockStore() *storage.MockStore {
	if m.mockdb == nil {
		m.mockdb = storage.NewMockStore()
	}
	return m.mockdb
}

func (m *McpService) LoadDebugMsgs(server *McpServer) []*entity.MsgEntity {
	if m.GetMockStore() == nil {
		return nil
	}
	task := &storage.TaskEntity{Name: server.Name}
	task.UUID = "#debug#" + server.UUID
	messages, _ := m.mockdb.LoadMsg(task)
	return messages
}

func (m *McpService) ClearDebugMsgs(server *McpServer) error {
	if m.GetMockStore() == nil {
		return nil
	}
	task := &storage.TaskEntity{Name: server.Name}
	task.UUID = "#debug#" + server.UUID
	return m.mockdb.ClearMsg(task)
}

func (m *McpService) QueryServer(mcp *McpServer, args ...int) error {
	if len(m.servers) == 0 || len(args) > 0 {
		m.ListServers()
	}
	if m.servers[mcp.UUID] != nil {
		mcp = m.servers[mcp.UUID]
		return nil
	}
	return fmt.Errorf("mcp server not found")
}

func (m *McpService) UpsertServer(server *McpServer) error {
	return m.storage.UpsertServer(server)
}

func (m *McpService) RemoveServer(server *McpServer) error {
	return m.storage.DeleteServer(server)
}

func (m *McpService) EnableServer(server *McpServer) error {
	server.Status.Enable = true
	return m.storage.UpsertConfig(server)
}

func (m *McpService) DisableServer(server *McpServer) error {
	server.Status.Enable = false
	server.Status.Active = false
	return m.storage.UpsertConfig(server)
}

// LoadMcpServer: 合并mcps配置到服务，并返回所有tools key（uuid:name）
func (m *McpService) LoadMcpServer(mcps map[string]any) {
	if len(mcps) == 0 {
		return
	}
	exist := map[string]*McpServer{}
	for _, s := range m.ListServers() {
		exist[s.UUID] = s
	}
	for uuid, val := range mcps {
		mcp, ok := val.(map[string]any)
		if !ok {
			continue
		}
		server := exist[uuid]
		if server == nil {
			server = &McpServer{UUID: uuid}
			_ = server.FromMap(mcp)
			m.UpsertServer(server)
			exist[uuid] = server
		}
		if len(server.Status.McpTools) == 0 {
			m.ServerStatus(server)
			m.EnableServer(server)
		}
	}
}
