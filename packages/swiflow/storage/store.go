package storage

import (
	"strings"
	"swiflow/config"
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

var mystore MyStore

func GetStorage() (MyStore, error) {
	if mystore != nil {
		return mystore, nil
	}
	needUpgrade := config.NeedUpgrade()
	kind := config.GetStr("STORAGE_TYPE", "sqlite")
	cfg := map[string]any{"path": config.GetWorkHome()}
	switch strings.ToLower(kind) {
	case "sqlite":
		cfg["path"] = config.SQLiteFile()
	case "mysql":
		dsn := config.MySQLDSN()
		switch dsn := dsn.(type) {
		case string:
			cfg["dsn"] = dsn
		case error:
			return nil, dsn
		}
	}
	store, err := NewStorage(kind, cfg)
	if store == nil || err != nil {
		return store, err
	}
	// handle migrate
	switch strings.ToLower(kind) {
	case "sqlite", "mysql":
		// Call without parameters to maintain existing behavior
		if _, err = store.LoadCfg(); err != nil {
			if strings.Contains(err.Error(), "no such table") {
				needUpgrade = true
			}
		}
	}
	if needUpgrade {
		err = store.AutoMigrate()
	}
	mystore = store
	return store, err
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
