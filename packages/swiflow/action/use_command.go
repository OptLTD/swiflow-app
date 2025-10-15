package action

import (
	"encoding/xml"
	"swiflow/ability"
	"time"
)

type ExecuteCommand struct {
	XMLName xml.Name `xml:"execute-command"`

	Command string `xml:"command" json:"command"`

	Result any `xml:"result" json:"result"`
}

func (act *ExecuteCommand) Handle(super *SuperAction) any {
	if err := super.Payload.InitHome(); err != nil {
		act.Result = err
		return err
	}
	command := ability.DevCommandAbility{Home: super.Payload.Home}
	if _, err := command.Exec(act.Command, 10*time.Second); err != nil {
		act.Result = command.Logs()
	} else {
		act.Result = command.Logs()
	}
	return act.Result
}

type StartAsyncCmd struct {
	XMLName xml.Name `xml:"start-async-cmd"`

	Session string `xml:"session" json:"session"`
	Command string `xml:"command" json:"command"`

	Result any `xml:"result" json:"result"`
}

func (act *StartAsyncCmd) Handle(super *SuperAction) any {
	if err := super.Payload.InitHome(); err != nil {
		act.Result = err
		return err
	}
	command := ability.DevAsyncCmdAbility{
		Name: act.Session, Home: super.Payload.Home,
	}
	if err := command.Start(act.Command); err != nil {
		act.Result = command.Logs()
	} else {
		act.Result = command.Logs()
	}
	return act.Result
}

type QueryAsyncCmd struct {
	XMLName xml.Name `xml:"query-async-cmd"`

	Session string `xml:"session" json:"session"`

	Result any `xml:"result" json:"result"`
}

func (act *QueryAsyncCmd) Handle(super *SuperAction) any {
	if err := super.Payload.InitHome(); err != nil {
		return err
	}
	command := ability.DevAsyncCmdAbility{
		Name: act.Session, Home: super.Payload.Home,
	}
	if logs, err := command.Query(); err != nil {
		act.Result = err
	} else {
		act.Result = logs
	}
	return act.Result
}

type AbortAsyncCmd struct {
	XMLName xml.Name `xml:"abort-async-cmd"`

	Session string `xml:"session" json:"session"`

	Result any `xml:"result" json:"result"`
}

func (act *AbortAsyncCmd) Handle(super *SuperAction) any {
	if err := super.Payload.InitHome(); err != nil {
		act.Result = err
		return err
	}
	command := ability.DevAsyncCmdAbility{
		Name: act.Session, Home: super.Payload.Home,
	}
	if err := command.Abort(); err != nil {
		act.Result = err
	} else {
		act.Result = "abort session success"
	}
	return act.Result
}
