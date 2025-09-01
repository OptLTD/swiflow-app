package action

import (
	"encoding/xml"
	"swiflow/ability"
)

// StartBotTask 用于调用其他bot作为工具来执行特定任务
type StartBotTask struct {
	XMLName xml.Name `xml:"start-bot-task"`
	UUID    string   `xml:"uuid" json:"uuid"`
	Name    string   `xml:"name" json:"name"`
	Task    string   `xml:"task" json:"task"`
	Context string   `xml:"context" json:"context"`

	Result any `xml:"result" json:"result"`
}

func (act *StartBotTask) Handle(super *SuperAction) any {
	// TODO: 实现自研Bot调用逻辑
	// 这里需要调用相应的ability来处理bot任务
	botAbility := ability.SelfBotAbility{
		UUID: act.UUID, Name: act.Name,
		Task: act.Task, Context: act.Context,
	}
	if data, err := botAbility.Execute(); err != nil {
		act.Result = err
	} else {
		act.Result = data
	}
	return act.Result
}

// QueryBotTask 用于查询bot的状态、能力和可用性
type QueryBotTask struct {
	XMLName xml.Name `xml:"query-bot-task"`
	UUID    string   `xml:"uuid" json:"uuid"`
	Name    string   `xml:"name" json:"name"`
	Type    string   `xml:"type" json:"type"`

	Result any `xml:"result" json:"result"`
}

func (act *QueryBotTask) Handle(super *SuperAction) any {
	// TODO: 实现自研Bot查询逻辑
	botAbility := ability.SelfBotAbility{
		UUID: act.UUID, Name: act.Name, Type: act.Type,
	}
	if data, err := botAbility.Query(); err != nil {
		act.Result = err
	} else {
		act.Result = data
	}
	return act.Result
}

// AbortBotTask 用于中止正在执行的bot任务
type AbortBotTask struct {
	XMLName xml.Name `xml:"abort-bot-task"`
	UUID    string   `xml:"uuid" json:"uuid"`
	Name    string   `xml:"name" json:"name"`
	Reason  string   `xml:"reason" json:"reason"`

	Result interface{} `xml:"result" json:"result"`
}

func (act *AbortBotTask) Handle(super *SuperAction) any {
	// TODO: 实现自研Bot中止逻辑
	botAbility := ability.SelfBotAbility{
		UUID: act.UUID, Name: act.Name, Reason: act.Reason,
	}
	if data, err := botAbility.Abort(); err != nil {
		act.Result = err
	} else {
		act.Result = data
	}
	return act.Result
}
