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

	Desc string `xml:"desc" json:"desc"`
	Name string `xml:"name" json:"name"`
	Tool string `xml:"tool" json:"tool"`
	Args string `xml:"args" json:"args"`

	Result any `xml:"result" json:"result"`
}

func (act *UseMcpTool) Handle(super *SuperAction) any {
	var client *amcp.McpClient
	var server = &amcp.McpServer{UUID: act.Name}
	if client = amcp.NewMcpClient(server); client == nil {
		act.Result = fmt.Errorf(
			"mcp server[%s][%s] not in service, err: %s",
			act.Name, act.Tool, server.Status.ErrMsg,
		)
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
