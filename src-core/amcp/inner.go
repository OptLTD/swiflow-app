package amcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewInnerMCPServer 创建内置的 MCP Server（go-sdk实现）
func NewInnerMCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name: "inner-mail-server", Version: "1.0.0",
	}, nil)

	mailRecvTool := &mcp.Tool{
		Name: "mail_recv", Description: "接收邮件",
		// InputSchema: mailRecvSchema,
	}

	mailRecvTool.InputSchema, _ = ToJsonSchema(`{
		"type": "object",
		"properties": {
			"mbox":   { "type": "string", "description": "邮箱文件夹" },
			"maxUid": { "type": "number", "description": "最大UID" }
		},
		"required": ["mbox", "maxUid"]
	}`)
	server.AddTool(mailRecvTool, nil)
	return server
}
