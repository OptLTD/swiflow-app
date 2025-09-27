package action

import (
	"encoding/xml"
	"fmt"
	"swiflow/builtin"
)

// UseBuiltinTool 用于调用MCP服务器提供的工具
type UseBuiltinTool struct {
	XMLName xml.Name `xml:"use-inbuilt-tool"`

	Desc string `xml:"desc" json:"desc"`
	Tool string `xml:"tool" json:"tool"`
	Args string `xml:"args" json:"args"`

	Result any `xml:"result" json:"result"`
}

func (act *UseBuiltinTool) Handle(super *SuperAction) any {
	var manager = builtin.GetManager()
	tool, err := manager.Query(act.Tool)
	if err != nil {
		act.Result = fmt.Errorf("error: %s", err)
		return act.Result
	}
	if resp, err := tool.Handle(act.Args); err != nil {
		act.Result = fmt.Errorf("error: %s", err)
	} else {
		act.Result = resp
	}
	return act.Result
}
