//go:build !windows
// +build !windows

package amcp

import (
	"context"
	"encoding/json"
	"errors"
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
	switch a.server.Type {
	case "streamable", "stream":
		headers := a.server.GetHeaders()
		client := support.NewHttpClient(headers)
		return &mcp.StreamableClientTransport{
			HTTPClient: client, Endpoint: a.server.Url,
		}, nil
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
	if transport, err := a.buildTransport(); err != nil {
		return fmt.Errorf("创建MCP客户端失败: %v", err)
	} else {
		a.transport = transport
	}

	a.client = mcp.NewClient(&mcp.Implementation{
		Name: "swiflow", Version: config.GetVersion(),
	}, nil)

	duration := time.Duration(CONNECT_TIMEOUT)
	ctx, cancel := context.WithTimeout(
		context.Background(), duration*time.Second,
	)
	defer cancel()

	session, err := a.client.Connect(ctx, a.transport, nil)
	if err != nil {
		return fmt.Errorf("启动MCP客户端失败: %v", err)
	}
	a.session = session

	// InitializeResult
	if resp := session.InitializeResult(); resp != nil {
		log.Println("[MCP] SUCCESS: %w", resp.ServerInfo)
	}

	return nil
}

func (a *McpClient) ListTools() ([]*McpTool, error) {
	log.Println("[MCP] Start List Tools:", a.server.UUID)
	if a.session == nil {
		if err := a.Initialize(); err != nil {
			return nil, err
		}
	}
	ctx := context.Background()
	param := &mcp.ListToolsParams{}
	result, err := a.session.ListTools(ctx, param)
	if result == nil || err != nil {
		log.Println("[MCP] List Tools Failed:", err)
		return nil, err
	}

	tools := make([]*McpTool, 0)
	for _, tool := range result.Tools {
		tools = append(tools, &McpTool{
			Name: tool.Name, Meta: tool.Meta,
			Description: tool.Description,
		})
	}
	return tools, nil
}

func (a *McpClient) Close() error {
	if a.session != nil {
		err := a.session.Close()
		if err != nil {
			log.Println("[MCP] mcp close error:", err)
		}
	}
	log.Println("[MCP] mcp closed:", a.server.UUID)
	delete(clients, a.server.UUID)
	return nil
}

func (a *McpClient) Execute(toolName string, args map[string]any) (string, error) {
	log.Println("[MCP] Start Execute:", toolName, support.ToJson(args))
	duration := time.Duration(EXECUTE_TIMEOUT)
	ctx, cancel := context.WithTimeout(
		context.Background(), duration*time.Second,
	)
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
	if errors.Is(err, mcp.ErrConnectionClosed) {
		log.Println("[MCP] Closed & Retry:", toolName)
		if err = a.Initialize(); err != nil {
			return "", err
		}
		res, err = a.session.CallTool(ctx, params)
	}
	if err != nil || res == nil {
		return "", fmt.Errorf("[MCP] 工具调用失败: %v", err)
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

func (a *McpClient) Resources() ([]*Resource, error) {
	log.Println("[MCP] List Resources:", a.server.UUID)
	if a.session == nil {
		if err := a.Initialize(); err != nil {
			return nil, err
		}
	}
	ctx := context.Background()
	param := &mcp.ListResourcesParams{}
	res, err := a.session.ListResources(ctx, param)
	if errors.Is(err, mcp.ErrConnectionClosed) {
		log.Println("[MCP] Closed & Retry:", param)
		if err = a.Initialize(); err != nil {
			return nil, err
		}
		res, err = a.session.ListResources(ctx, param)
	}
	if res == nil || err != nil {
		log.Println("[MCP] List Resources Failed:", err)
		return nil, err
	}

	list, name := make([]*Resource, 0), []string{}
	for _, res := range res.Resources {
		name = append(name, res.Name)
		list = append(list, &Resource{
			Meta: res.Meta, MIMEType: res.MIMEType,
			Name: res.Name, URI: res.URI, Title: res.Title,
			Size: res.Size, Description: res.Description,
		})
	}
	log.Println("[MCP] List Resources:", name)
	return list, nil
}

func (a *McpClient) Resource(uri string) (string, error) {
	log.Println("[MCP] Get Resource:", a.server.Name, uri)
	duration := time.Duration(EXECUTE_TIMEOUT)
	ctx, cancel := context.WithTimeout(
		context.Background(), duration*time.Second,
	)
	defer cancel()
	if a.session == nil {
		if err := a.Initialize(); err != nil {
			return "", err
		}
	}

	param := &mcp.ReadResourceParams{URI: uri}
	res, err := a.session.ReadResource(ctx, param)
	if errors.Is(err, mcp.ErrConnectionClosed) {
		log.Println("[MCP] Closed & Retry:", param)
		if err = a.Initialize(); err != nil {
			return "", err
		}
		res, err = a.session.ReadResource(ctx, param)
	}
	if err != nil || res == nil {
		log.Println("[MCP] Read Resource Failed:", err)
		return "", fmt.Errorf("[MCP] 资源读取失败: %v", err)
	}
	if len(res.Contents) > 0 {
		data := res.Contents[0]
		if data.Text != "" {
			return data.Text, nil
		}
		switch {
		case data.Text != "":
			return data.Text, nil
		case len(data.Blob) > 0:
			return string(data.Blob), nil
		default:
			return "", nil
		}
	}
	return "", nil
}
