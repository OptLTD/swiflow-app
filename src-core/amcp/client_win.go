//go:build windows
// +build windows

package amcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"swiflow/config"
	"swiflow/support"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

var clients = make(map[string]*McpClient)

func NewMcpClient(server *McpServer) *McpClient {
	if get, ok := clients[server.UUID]; ok {
		return get
	}
	c := &McpClient{server: server}
	if err := c.Initialize(); err != nil {
		server.Status.ErrMsg = err
		log.Println("[MCP] init fail:", err)
		return nil
	}
	// debug mode not cache
	if server.Type != "debug" {
		clients[server.UUID] = c
	}
	return c
}

type McpClient struct {
	server *McpServer
	client *client.Client
}

func (a *McpClient) Initialize() error {
	log.Println("[MCP] Start Init Mcp Server:", a.server.UUID)

	// Create client based on server type
	var mcpClient *client.Client
	switch a.server.Type {
	case "debug":
		cmdPath, err := config.GetMcpEnv(a.server.Cmd)
		mcpClient, err = client.NewStdioMCPClient(
			cmdPath, a.server.GetEnv(), a.server.Args...,
		)
		if err != nil {
			return fmt.Errorf("启动MCP客户端失败: %v", err)
		}
	default:
		opt := transport.WithCommandFunc(func(ctx context.Context, cmd string, env, args []string) (*exec.Cmd, error) {
			command := exec.CommandContext(ctx, cmd, args...)
			command.Env = append(os.Environ(), env...)
			command.Dir = config.CurrentHome()
			return command, nil
		})
		cmdPath, err := config.GetMcpEnv(a.server.Cmd)
		mcpClient, err = client.NewStdioMCPClientWithOptions(
			cmdPath, a.server.GetEnv(), a.server.Args, opt,
		)
		if err != nil {
			return fmt.Errorf("启动MCP客户端失败: %v", err)
		}
	}

	// Additional safety check to ensure client is not nil
	if mcpClient == nil {
		return fmt.Errorf("MCP客户端创建失败: client is nil")
	}

	// Initialize the client
	duration := time.Duration(CONNECT_TIMEOUT)
	ctx, cancel := context.WithTimeout(context.Background(), duration*time.Second)
	defer cancel()

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.Capabilities = mcp.ClientCapabilities{}
	initReq.Params.ClientInfo = mcp.Implementation{
		Name: "Swiflow Client", Version: "1.0.0",
	}
	if res, err := mcpClient.Initialize(ctx, initReq); err != nil {
		return fmt.Errorf("初始化MCP客户端失败: %v", err)
	} else if err = mcpClient.Start(ctx); err != nil {
		return fmt.Errorf("启动MCP客户端失败: %v", err)
	} else {
		a.client = mcpClient
		log.Println("[MCP] SUCCESS: %w", res.Result)
	}
	return nil
}

func (a *McpClient) ListTools() ([]*McpTool, error) {
	log.Println("[MCP] Start List Tools:", a.server.UUID)
	if a.client == nil {
		if err := a.Initialize(); err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	result, err := a.client.ListTools(ctx, mcp.ListToolsRequest{})
	if result == nil || err != nil {
		return nil, err
	}

	tools := make([]*McpTool, 0)
	for _, tool := range result.Tools {
		tools = append(tools, &McpTool{
			Name:        tool.Name,
			Description: tool.Description,
		})
	}
	return tools, nil
}

func (a *McpClient) Close() error {
	if a.client != nil {
		err := a.client.Close()
		if err != nil {
			log.Println("[MCP] mcp close error:", err)
		}
	}
	delete(clients, a.server.UUID)
	return nil
}

func (a *McpClient) Execute(toolName string, args map[string]any) (string, error) {
	log.Println("[MCP] Start Execute:", toolName, support.ToJson(args))
	duration := time.Duration(EXECUTE_TIMEOUT)
	ctx, cancel := context.WithTimeout(context.Background(), duration*time.Second)
	defer cancel()

	if a.client == nil {
		if err := a.Initialize(); err != nil {
			return "", err
		}
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	}

	res, err := a.client.CallTool(ctx, request)
	if err != nil || res == nil {
		return "", fmt.Errorf("MCP工具调用失败: %v", err)
	}

	if len(res.Content) > 0 {
		switch v := res.Content[0].(type) {
		case *mcp.TextContent:
			return v.Text, nil
		default:
			b, _ := json.Marshal(v)
			return string(b), nil
		}
	}
	return "", nil
}
