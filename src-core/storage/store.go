package storage

import (
	"strings"
)

type MyStore interface {
	AutoMigrate() error

	InitTask(*TaskEntity) error
	FindTask(*TaskEntity) error
	SaveTask(*TaskEntity) error
	LoadTask() ([]*TaskEntity, error)

	FindMsg(*MsgEntity) error
	SaveMsg(*MsgEntity) error
	LoadMsg(*TaskEntity) ([]*MsgEntity, error)

	FindBot(*BotEntity) error
	SaveBot(*BotEntity) error
	LoadBot() ([]*BotEntity, error)

	FindCfg(*CfgEntity) error
	SaveCfg(*CfgEntity) error
	LoadCfg() ([]*CfgEntity, error)

	FindMem(*MemEntity) error
	SaveMem(*MemEntity) error
	LoadMem() ([]*MemEntity, error)

	FindTool(*ToolEntity) error
	SaveTool(*ToolEntity) error
	LoadTool() ([]*ToolEntity, error)

	FindTodo(*TodoEntity) error
	SaveTodo(*TodoEntity) error
	LoadTodo() ([]*TodoEntity, error)
	LoadDone() ([]*TodoEntity, error)
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
