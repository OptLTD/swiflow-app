package action

import (
	"encoding/xml"
)

// StartSubtask 用于调用其他agent作为工具来执行特定任务
type StartSubtask struct {
	XMLName xml.Name `xml:"start-subtask"`
	UUID    string   `xml:"uuid" json:"uuid"`
	Name    string   `xml:"name" json:"name"`
	Task    string   `xml:"task" json:"task"`
	Context string   `xml:"context" json:"context"`

	Result any `xml:"result" json:"result"`
}

// QuerySubtask 用于查询subtask的状态
type QuerySubtask struct {
	XMLName xml.Name `xml:"query-subtask"`
	UUID    string   `xml:"uuid" json:"uuid"`
	Name    string   `xml:"name" json:"name"`
	Type    string   `xml:"type" json:"type"`

	Result any `xml:"result" json:"result"`
}

// AbortSubtask 用于中止正在执行的subtask
type AbortSubtask struct {
	XMLName xml.Name `xml:"abort-subtask"`
	UUID    string   `xml:"uuid" json:"uuid"`
	Name    string   `xml:"name" json:"name"`
	Reason  string   `xml:"reason" json:"reason"`

	Result interface{} `xml:"result" json:"result"`
}
