package storage

import (
	"strings"
)

type MyStore interface {
	AutoMigrate() error

	InitTask(*TaskEntity) error
	FindTask(*TaskEntity) error
	SaveTask(*TaskEntity) error
	LoadTask(query ...any) ([]*TaskEntity, error)

	FindMsg(*MsgEntity) error
	SaveMsg(*MsgEntity) error
	LoadMsg(*TaskEntity) ([]*MsgEntity, error)

	FindBot(*BotEntity) error
	SaveBot(*BotEntity) error
	LoadBot(query ...any) ([]*BotEntity, error)

	FindCfg(*CfgEntity) error
	SaveCfg(*CfgEntity) error
	LoadCfg(query ...any) ([]*CfgEntity, error)

	FindMem(*MemEntity) error
	SaveMem(*MemEntity) error
	LoadMem(query ...any) ([]*MemEntity, error)

	FindTool(*ToolEntity) error
	SaveTool(*ToolEntity) error
	LoadTool(query ...any) ([]*ToolEntity, error)

	FindTodo(*TodoEntity) error
	SaveTodo(*TodoEntity) error
	LoadTodo(query ...any) ([]*TodoEntity, error)
}

func NewStorage(kind string, config map[string]any) (MyStore, error) {
	switch strings.ToLower(kind) {
	case "mock":
		return NewMockStore(), nil
	case "mysql":
		return NewMySQLStorage(config)
	case "sqlite":
		return NewSQLiteStorage(config)
	default:
		return NewSQLiteStorage(config)
	}
}
