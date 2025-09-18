package entity

import (
	"gorm.io/gorm"
)

type BotEntity struct {
	ID uint `gorm:"primarykey"`

	UUID string `json:"uuid" gorm:"uuid;size:16;not null;uniqueIndex"`
	Type string `json:"type" gorm:"column:type;size:16;not null"`
	Name string `json:"name" gorm:"column:name;size:50;not null"`
	Home string `json:"home" gorm:"column:home;size:200"`
	Desc string `json:"desc" gorm:"column:desc;size:200"`

	Emoji string   `json:"emoji" gorm:"column:emoji;size:50"`
	Tools []string `json:"tools" gorm:"tools;serializer:json;"`

	UsePrompt string `json:"usePrompt" gorm:"column:use_prompt"`
	SysPrompt string `json:"sysPrompt" gorm:"column:sys_prompt"`

	Provider string `json:"provider" gorm:"provider;size:50"`
	// Endpoint  string `json:"endpoint" gorm:"endpoint;size:200"`
	// ApiSecret string `json:"apiSecret" gorm:"api_secret;size:50"`
	// ModelName string `json:"modelName" gorm:"model_name;size:50"`

	// Bot Memories
	Memories []MemEntity `json:"-" gorm:"foreignKey:Bot;references:UUID"`

	// Use Mcp Servers
	McpServers map[string]any `json:"-" gorm:"-"`

	gorm.Model `json:"-"`
}

func (m *BotEntity) TableName() string {
	return "llm_bot"
}

func (r *BotEntity) ToMap() map[string]any {
	return map[string]any{
		"uuid": r.UUID, "type": r.Type, "name": r.Name,
		"home": r.Home, "tools": r.Tools, "desc": r.Desc,
		"provider": r.Provider, "emoji": r.Emoji,
	}
}
