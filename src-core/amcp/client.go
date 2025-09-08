package amcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"swiflow/config"
	"swiflow/support"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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
	server    *McpServer
	session   *mcp.ClientSession
	client    *mcp.Client
	transport mcp.Transport
}

func (a *McpClient) buildTransport() (mcp.Transport, error) {
	// 只支持 command/stdio
	switch a.server.Type {
	case "memory":
		memTransport, _ := mcp.NewInMemoryTransports()
		return memTransport, nil
	case "debug":
		if cmd, err := a.server.GetCmd(); err != nil {
			return nil, err
		} else {
			pr, pw := io.Pipe()
			stdio := &mcp.CommandTransport{Command: cmd}
			support.WatchOutput(a.server.UUID, pr)
			return &mcp.LoggingTransport{
				Transport: stdio, Writer: pw,
			}, nil
		}
	default:
		if cmd, err := a.server.GetCmd(); err != nil {
			return nil, err
		} else {
			return &mcp.CommandTransport{Command: cmd}, nil
		}
	}
}

func (a *McpClient) Initialize() error {
	log.Println("[MCP] Start Init Mcp Server:", a.server.UUID)
	transport, err := a.buildTransport()
	if err != nil {
		return fmt.Errorf("创建MCP客户端失败: %v", err)
	}
	a.transport = transport
	a.client = mcp.NewClient(&mcp.Implementation{
		Name: "swiflow-app", Version: config.GetVersion(),
	}, nil)

	duration := time.Duration(CONNECT_TIMEOUT)
	ctx, cancel := context.WithTimeout(
		context.Background(), duration*time.Second,
	)
	defer cancel()

	session, err := a.client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("启动MCP客户端失败: %v", err)
	}
	a.session = session

	log.Println("[MCP] SUCCESS: connect ok")
	return nil
}

func (a *McpClient) ListTools() (*mcp.ListToolsResult, error) {
	log.Println("[MCP] Start List Tools:", a.server.UUID)
	if a.session == nil {
		if err := a.Initialize(); err != nil {
			return nil, err
		}
	}
	ctx := context.Background()
	return a.session.ListTools(ctx, &mcp.ListToolsParams{})
}

func (a *McpClient) Close() error {
	if a.session != nil {
		err := a.session.Close()
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
	if a.session == nil {
		if err := a.Initialize(); err != nil {
			return "", err
		}
	}
	params := &mcp.CallToolParams{
		Name: toolName, Arguments: args,
	}
	res, err := a.session.CallTool(ctx, params)
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
