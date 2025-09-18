package action

import (
	"encoding/xml"
	"swiflow/support"
)

// StartSubtask 用于调用其他agent作为工具来执行特定任务
type StartSubtask struct {
	XMLName xml.Name `xml:"start-subtask"`

	SubAgent string `xml:"sub-agent" json:"sub-agent"`
	TaskDesc string `xml:"task-desc" json:"task-desc"`
	Context  string `xml:"context" json:"context"`
	Require  string `xml:"require" json:"require"`

	Result any `xml:"result" json:"result"`
}

func (action *StartSubtask) Input() (string, string) {
	return support.ToXML(action, action.Result), "start-subtask"
}

// QuerySubtask 用于查询subtask的状态
type QuerySubtask struct {
	XMLName xml.Name `xml:"query-subtask"`

	SubAgent string `xml:"sub-agent" json:"sub-agent"`

	Result any `xml:"result" json:"result"`
}

func (action *QuerySubtask) Input() (string, string) {
	return support.ToXML(action, action.Result), "query-subtask"
}

// AbortSubtask 用于中止正在执行的subtask
type AbortSubtask struct {
	XMLName xml.Name `xml:"abort-subtask"`

	SubAgent string `xml:"sub-agent" json:"sub-agent"`

	Result any `xml:"result" json:"result"`
}

func (action *AbortSubtask) Input() (string, string) {
	return support.ToXML(action, action.Result), "abort-subtask"
}
