package entity

import (
	"gorm.io/gorm"
)

type TaskEntity struct {
	ID uint `gorm:"primarykey"`

	UUID  string `json:"-" gorm:"column:uuid;size:36;not null;uniqueIndex"`
	Name  string `json:"name" gorm:"column:name;size:200;not null"`
	Home  string `json:"home" gorm:"column:home;size:200"`
	Desc  string `json:"desc" gorm:"column:desc;size:200"`
	Group string `json:"group" gorm:"column:group;size:36"`
	BotId string `json:"botid" gorm:"column:botid;size:36"`
	// 任务状态 (process, running, completed, failed, canceled)
	State string `json:"state" gorm:"column:state;size:10"`
	// session, from feishu or another bot
	SessID string `json:"sessid" gorm:"column:sessid;size:36"`
	Source string `json:"source" gorm:"column:source;size:36"`

	Context string `json:"context" gorm:"column:context"`
	Process int32  `json:"process" gorm:"column:process"`
	Command string `json:"command" gorm:"column:command"`

	IsDebug bool `gorm:"-:all"`

	gorm.Model `json:"-"`
}

func (m *TaskEntity) TableName() string {
	return "llm_task"
}

func (m *TaskEntity) ToMap() map[string]any {
	return map[string]any{
		"uuid": m.UUID, "name": m.Name, "home": m.Home,
		"botid": m.BotId, "group": m.Group, "state": m.State,
		"sessid": m.SessID, "source": m.Source, "desc": m.Desc,
		"context": m.Context, "command": m.Command, "process": m.Process,
	}
}
