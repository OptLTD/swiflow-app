package action

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"swiflow/amcp"
)

// UseMcpTool 用于调用MCP服务器提供的工具
type UseMcpTool struct {
	XMLName xml.Name `xml:"use-mcp-tool"`
	Title   string   `xml:"title" json:"title"`
	Server  string   `xml:"server" json:"server"`
	Tool    string   `xml:"tool" json:"tool"`
	Args    string   `xml:"args" json:"args"`

	Result any `xml:"result" json:"result"`
}

func (act *UseMcpTool) Handle(super *SuperAction) any {
	var client *amcp.McpClient
	var server = &amcp.McpServer{UUID: act.Server}
	if client = amcp.NewMcpClient(server); client == nil {
		act.Result = fmt.Errorf("server unservice")
		return act.Result
	}
	var args = map[string]any{}
	var data = []byte(act.Args)
	json.Unmarshal(data, &args)
	resp, err := client.Execute(
		act.Tool, args,
	)
	if err == nil && resp != "" {
		act.Result = resp
	} else {
		act.Result = fmt.Errorf("error: %s", err)
	}
	return act.Result
}
