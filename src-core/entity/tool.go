package entity

import (
	"gorm.io/gorm"
)

type ToolEntity struct {
	ID uint `gorm:"primarykey"`

	UUID string `json:"uuid" gorm:"uuid;size:16;not null;uniqueIndex"`
	Type string `json:"type" gorm:"column:type;size:10;not null"`
	Name string `json:"name" gorm:"column:name;size:50;not null"`
	Desc string `json:"desc" gorm:"column:desc;size:50"`
	Code string `json:"code" gorm:"column:code"`
	Deps string `json:"deps" gorm:"column:deps"`

	gorm.Model `json:"-"`
}

func (m *ToolEntity) TableName() string {
	return "llm_tool"
}

func (r *ToolEntity) ToMap() map[string]any {
	return map[string]any{
		"uuid": r.UUID, "type": r.Type,
		"name": r.Name, "desc": r.Desc,
		"code": r.Code, "deps": r.Deps,
	}
}
