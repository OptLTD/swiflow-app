package entity

import (
	"gorm.io/gorm"
)

type TodoEntity struct {
	ID uint `gorm:"primarykey"`

	UUID string `json:"uuid" gorm:"uuid;size:16;not null;uniqueIndex"`
	Task string `json:"task" gorm:"column:task;size:16;not null"`
	Time string `json:"time" gorm:"column:time;size:16;not null"`
	Todo string `json:"todo" gorm:"column:todo;not null;"`
	Done uint8  `json:"done" gorm:"column:done;default 0"`

	gorm.Model `json:"-"`
}

func (m *TodoEntity) TableName() string {
	return "llm_todo"
}

func (r *TodoEntity) ToMap() map[string]any {
	return map[string]any{
		"uuid": r.UUID, "task": r.Task,
		"time": r.Time, "todo": r.Todo,
		"done": r.Done,
	}
}
