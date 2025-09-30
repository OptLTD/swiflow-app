package action

import (
	"encoding/xml"
	"fmt"
	"swiflow/support"
	"time"
)

type ToolResult struct {
	XMLName xml.Name `xml:"tool-result" json:"-"`
	Content string   `xml:"content" json:"content"`
}

func (action *ToolResult) Input() (string, string) {
	return action.Content, "tool-result"
}

type UserInput struct {
	XMLName xml.Name `xml:"user-input" json:"-"`
	Content string   `xml:"content" json:"content"`
	Uploads []string `xml:"uploads" json:"uploads"`
}

func (action *UserInput) Input() (string, string) {
	return support.ToXML(action, nil), "user-input"
}

type Subtask struct {
	XMLName xml.Name `xml:"subtask" json:"-"`

	TaskDesc string `xml:"task-desc" json:"task-desc"`
	Context  string `xml:"context" json:"context"`
	Require  string `xml:"require" json:"require"`
}

func (action *Subtask) Input() (string, string) {
	return support.ToXML(action, nil), "subtask"
}

// wait todo somthing
type WaitTodo struct {
	XMLName xml.Name `xml:"wait-todo"`

	UUID string `xml:"uuid" json:"uuid"`
	Time string `xml:"time" json:"time"`
	Todo string `xml:"todo" json:"todo"`
}

func (act *WaitTodo) Input() (string, string) {
	now := time.Now().Format(time.DateTime)
	msg := fmt.Sprint("[TRIGGERED]触发时间", now)
	return support.ToXML(act, msg), "crontab"
}

// 直接完成工作
type Complete struct {
	XMLName xml.Name `xml:"complete"`

	BotName string `xml:"botname" json:"botname"`
	Content string `xml:"content" json:"content"`
}

type MakeAsk struct {
	XMLName  xml.Name `xml:"make-ask"`
	Question string   `xml:"question" json:"question"`
	Multiple string   `xml:"multiple" json:"multiple"`
	Options  []string `xml:"options>option" json:"options"`
}

type Thinking struct {
	XMLName xml.Name `xml:"thinking/think"`
	Content string   `xml:"content" json:"content"`
}

type Memorize struct {
	XMLName  xml.Name `xml:"memorize"`
	Subject  string   `xml:"subject" json:"subject"`
	Content  string   `xml:"content" json:"content"`
	Datetime string   `xml:"datetime" json:"datetime"`
}

// alias Summarise
type Context = Annotate

type Annotate struct {
	XMLName xml.Name `xml:"annotate"`
	Subject string   `xml:"subject" json:"subject"`
	Context string   `xml:"context" json:"context"`
}
