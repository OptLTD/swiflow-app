// Package mcpclient connects to MCP servers and surfaces their tools in the
// tool registry. Phase 2.
package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/tenant"
	"github.com/OptLTD/swiflow/internal/tool"
)

const toolPrefix = "mcp_"

var sanitizeRe = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

// Manager maintains MCP server sessions and registers bridged tools.
//
// Sessions are keyed by tid+"/"+serverID. Sync(ctx) only refreshes the tenant
// from context (does not tear down other tenants).
//
// Tool names stay mcp_{server}_{tool}. Ownership is tracked per name; if another
// tenant already owns the same name, registration is skipped (first-wins).
type Manager struct {
	st  store.Store
	reg *tool.Registry

	mu        sync.Mutex
	sessions  map[string]*conn // keyed by tid+"/"+serverID
	toolOwner map[string]string // tool name -> tid
}

type conn struct {
	tid        string
	serverID   string
	serverName string
	session    *sdkmcp.ClientSession
	toolNames  []string
}

func sessionKey(tid, serverID string) string {
	if tid == "" {
		tid = tenant.DefaultID
	}
	return tid + "/" + serverID
}

// NewManager creates an MCP manager.
func NewManager(st store.Store, reg *tool.Registry) *Manager {
	return &Manager{
		st:        st,
		reg:       reg,
		sessions:  map[string]*conn{},
		toolOwner: map[string]string{},
	}
}

// Sync connects enabled MCP servers for tenant.ID(ctx) and registers their tools.
// Other tenants' sessions remain untouched.
func (m *Manager) Sync(ctx context.Context) error {
	tid := tenant.ID(ctx)
	servers, err := m.st.ListMCPServers(ctx)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.unregisterTenantLocked(tid)

	for _, srv := range servers {
		if !srv.Enabled {
			continue
		}
		c, err := m.connectServer(ctx, tid, srv)
		if err != nil {
			slog.Error("mcp connect failed", "tenant", tid, "server", srv.Name, "error", err)
			continue
		}
		m.sessions[sessionKey(tid, srv.ID)] = c
	}
	return nil
}

// UnregisterTenant tears down MCP sessions and tools owned by tid.
func (m *Manager) UnregisterTenant(tid string) {
	if tid == "" {
		tid = tenant.DefaultID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unregisterTenantLocked(tid)
}

// OwnsTool reports whether name is an MCP tool owned by tid.
// Non-MCP tool names return false (caller should treat them as globally shared builtins).
func (m *Manager) OwnsTool(name, tid string) bool {
	if m == nil || !strings.HasPrefix(name, toolPrefix) {
		return false
	}
	if tid == "" {
		tid = tenant.DefaultID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	owner, ok := m.toolOwner[name]
	return ok && owner == tid
}

// IsMCPTool reports whether name is an MCP-bridged tool currently registered.
func (m *Manager) IsMCPTool(name string) bool {
	if m == nil || !strings.HasPrefix(name, toolPrefix) {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.toolOwner[name]
	return ok
}

func (m *Manager) unregisterTenantLocked(tid string) {
	prefix := tid + "/"
	for key, c := range m.sessions {
		if c.tid != tid && !strings.HasPrefix(key, prefix) {
			continue
		}
		m.unregisterConnLocked(c)
		_ = c.session.Close()
		delete(m.sessions, key)
	}
}

// Close shuts down all MCP sessions.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, c := range m.sessions {
		m.unregisterConnLocked(c)
		_ = c.session.Close()
		delete(m.sessions, id)
	}
	m.toolOwner = map[string]string{}
}

func (m *Manager) connectServer(ctx context.Context, tid string, srv store.MCPServer) (*conn, error) {
	transport, err := newTransport(srv)
	if err != nil {
		return nil, err
	}
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "swiflow", Version: "0.2.0"}, nil)
	connectCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	session, err := client.Connect(connectCtx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	c := &conn{tid: tid, serverID: srv.ID, serverName: srv.Name, session: session}
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("list tools: %w", err)
	}
	for _, t := range tools.Tools {
		fullName := ToolName(srv.Name, t.Name)
		if owner, ok := m.toolOwner[fullName]; ok && owner != tid {
			slog.Warn("mcp tool name conflict; skipping",
				"tool", fullName, "owner_tid", owner, "skipped_tid", tid, "server", srv.Name)
			continue
		}
		schema := inputSchemaMap(t.InputSchema)
		bt := &bridgeTool{
			fullName: fullName,
			mcpName:  t.Name,
			desc:     toolDescription(srv.Name, t),
			schema:   schema,
			session:  session,
		}
		m.reg.Register(bt)
		m.toolOwner[fullName] = tid
		c.toolNames = append(c.toolNames, fullName)
	}
	slog.Info("mcp server connected", "tenant", tid, "server", srv.Name, "tools", len(c.toolNames))
	return c, nil
}

func (m *Manager) unregisterConnLocked(c *conn) {
	for _, name := range c.toolNames {
		if owner, ok := m.toolOwner[name]; ok && owner == c.tid {
			delete(m.toolOwner, name)
			m.reg.Unregister(name)
		}
	}
}

func newTransport(srv store.MCPServer) (sdkmcp.Transport, error) {
	switch srv.Type {
	case "stdio":
		if srv.Cmd == "" {
			return nil, fmt.Errorf("stdio type requires cmd")
		}
		cmd := exec.Command(srv.Cmd, srv.Args...)
		cmd.Env = os.Environ()
		for k, v := range srv.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
		return &sdkmcp.CommandTransport{Command: cmd}, nil
	case "sse":
		if srv.URL == "" {
			return nil, fmt.Errorf("sse type requires url")
		}
		return &sdkmcp.SSEClientTransport{
			Endpoint:   srv.URL,
			HTTPClient: &http.Client{Timeout: 0},
		}, nil
	case "streamable":
		if srv.URL == "" {
			return nil, fmt.Errorf("streamable type requires url")
		}
		return &sdkmcp.StreamableClientTransport{
			Endpoint:   srv.URL,
			HTTPClient: &http.Client{Timeout: 0},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported mcp type %q", srv.Type)
	}
}

// ToolName builds the registry name for an MCP tool.
func ToolName(serverName, mcpToolName string) string {
	return toolPrefix + sanitize(serverName) + "_" + sanitize(mcpToolName)
}

func sanitize(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "x"
	}
	return sanitizeRe.ReplaceAllString(s, "_")
}

func toolDescription(serverName string, t *sdkmcp.Tool) string {
	if t.Description != "" {
		return fmt.Sprintf("[%s] %s", serverName, t.Description)
	}
	return fmt.Sprintf("[%s] MCP tool %s", serverName, t.Name)
}

func inputSchemaMap(schema any) map[string]any {
	if schema == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	switch v := schema.(type) {
	case map[string]any:
		return v
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return map[string]any{"type": "object", "properties": map[string]any{}}
		}
		var out map[string]any
		if json.Unmarshal(b, &out) == nil {
			return out
		}
	}
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

type bridgeTool struct {
	fullName string
	mcpName  string
	desc     string
	schema   map[string]any
	session  *sdkmcp.ClientSession
}

func (t *bridgeTool) Name() string        { return t.fullName }
func (t *bridgeTool) Description() string { return t.desc }
func (t *bridgeTool) Parameters() map[string]any {
	return t.schema
}

func (t *bridgeTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	res, err := t.session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      t.mcpName,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}
	return formatResult(res), nil
}

func formatResult(res *sdkmcp.CallToolResult) string {
	if res == nil {
		return ""
	}
	if res.IsError {
		return "error: " + contentText(res.Content)
	}
	if res.StructuredContent != nil {
		b, err := json.MarshalIndent(res.StructuredContent, "", "  ")
		if err == nil {
			return string(b)
		}
	}
	text := contentText(res.Content)
	if text != "" {
		return text
	}
	if res.StructuredContent != nil {
		b, _ := json.Marshal(res.StructuredContent)
		return string(b)
	}
	return ""
}

func contentText(parts []sdkmcp.Content) string {
	var b strings.Builder
	for _, p := range parts {
		switch c := p.(type) {
		case *sdkmcp.TextContent:
			b.WriteString(c.Text)
		default:
			raw, err := json.Marshal(p)
			if err == nil {
				b.Write(raw)
			}
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
	}
	return strings.TrimSpace(b.String())
}
