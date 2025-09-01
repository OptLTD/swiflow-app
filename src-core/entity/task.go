package entity

import (
	"strings"

	"gorm.io/gorm"
)

type TaskEntity struct {
	ID uint `gorm:"primarykey"`

	UUID  string `json:"-" gorm:"uuid;size:36;not null;uniqueIndex"`
	Name  string `json:"name" gorm:"name;size:200;not null"`
	Home  string `json:"home" gorm:"home;size:200"`
	Bots  string `json:"bots" gorm:"bots;size:200"`
	Desc  string `json:"desc" gorm:"desc;size:200"`
	State string `json:"state" gorm:"state;size:10"`
	// 任务状态 (process, running, completed, failed, canceled)

	Context string `json:"context" gorm:"context"`
	Process int32  `json:"process" gorm:"process"`
	Command string `json:"command" gorm:"command"`

	gorm.Model `json:"-"`
}

func (m *TaskEntity) TableName() string {
	return "llm_task"
}

func (m *TaskEntity) ToMap() map[string]any {
	return map[string]any{
		"bots": strings.Split(m.Bots, ","),
		"uuid": m.UUID, "name": m.Name, "home": m.Home,
		"command": m.Command, "process": m.Process,
		"context": m.Context, "state": m.State,
	}
}
