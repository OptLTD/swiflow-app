package action

import (
	"encoding/xml"
)

// UseSelfTool 用于调用已发布的自研工具
type UseSelfTool struct {
	XMLName    xml.Name `xml:"use-self-tool"`
	UUID       string   `xml:"uuid" json:"uuid"`
	Name       string   `xml:"name" json:"name"`
	Version    string   `xml:"version" json:"version"`
	Parameters string   `xml:"parameters" json:"parameters"`

	Result interface{} `xml:"result" json:"result"`
}

// SetSelfTool 用于发布新工具或更新已存在的工具到工具库中
type SetSelfTool struct {
	XMLName      xml.Name `xml:"set-self-tool"`
	UUID         string   `xml:"uuid" json:"uuid"`
	Name         string   `xml:"name" json:"name"`
	Description  string   `xml:"description" json:"description"`
	Version      string   `xml:"version" json:"version"`
	Author       string   `xml:"author" json:"author"`
	Category     string   `xml:"category" json:"category"`
	Tags         string   `xml:"tags" json:"tags"`
	Code         string   `xml:"code" json:"code"`
	Dependencies string   `xml:"dependencies" json:"dependencies"`
	Examples     string   `xml:"examples" json:"examples"`
	Changelog    string   `xml:"changelog" json:"changelog"`

	Result interface{} `xml:"result" json:"result"`
}

// PublishAsApp 用于调用已发布的自研工具
type PublishAsApp struct {
	XMLName xml.Name `xml:"publish-as-app"`

	Title   string `xml:"title" json:"title"`
	Command string `xml:"command" json:"command"`
	Address string `xml:"address" json:"address"`

	Result interface{} `xml:"result" json:"result"`
}
