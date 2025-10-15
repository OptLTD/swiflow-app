package entity

import (
	"gorm.io/gorm"
)

type MemEntity struct {
	ID uint `gorm:"primarykey"`

	Bot  string `gorm:"column:bot;size:16;"`
	Type string `gorm:"column:type;size:10;"`

	Subject string `gorm:"column:subject;not null;size:200"`
	Content string `gorm:"column:content;not null;longtext"`

	gorm.Model `json:"-"`
}

func (m *MemEntity) TableName() string {
	return "llm_mem"
}

func (m *MemEntity) ToMap() map[string]any {
	return map[string]any{
		"id": m.ID, "bot": m.Bot, "type": m.Type,
		"subject": m.Subject, "content": m.Content,
	}
}
