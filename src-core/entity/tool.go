package entity

import (
	"gorm.io/gorm"
)

type ToolEntity struct {
	ID uint `json:"-" gorm:"primarykey"`

	UUID string `json:"uuid" gorm:"uuid;size:16;not null;uniqueIndex"`
	Type string `json:"type" gorm:"column:type;size:10;not null"`
	Name string `json:"name" gorm:"column:name;size:50;not null"`
	Desc string `json:"desc" gorm:"column:desc;size:500"`
	Data object `json:"data" gorm:"column:data;serializer:json;"`

	gorm.Model `json:"-"`
}

func (m *ToolEntity) TableName() string {
	return "llm_tool"
}

func (r *ToolEntity) ToMap() map[string]any {
	return map[string]any{
		"uuid": r.UUID, "type": r.Type, "name": r.Name,
		"desc": r.Desc, "data": r.Data,
	}
}
