package entity

import (
	"database/sql"
	"fmt"
	"swiflow/errors"
	"time"

	"gorm.io/gorm"
)

type object = map[string]any

type CfgEntity struct {
	ID uint `gorm:"primarykey"`

	Type string `json:"type" gorm:"size:30;not null;uniqueIndex:idx_uniq"`
	Name string `json:"name" gorm:"size:80;not null;uniqueIndex:idx_uniq"`
	Data object `json:"data" gorm:"type:text;serializer:json;not null"`

	gorm.Model `json:"-"`
}

func (m *CfgEntity) TableName() string {
	return "llm_cfg"
}

func (m *CfgEntity) ToMap() map[string]any {
	return map[string]any{
		"type": m.Type,
		"name": m.Name,
		"data": m.Data,
	}
}

func (m *CfgEntity) GetMySQL() (*sql.DB, error) {
	if m.Name != "MYSQL" || m.Type != KEY_CFG_DATA {
		return nil, fmt.Errorf("%w: %s", errors.ErrorConfig, "wrong config")
	}
	var host, port, name, user, pass string
	if host, _ = m.Data["host"].(string); host == "" {
		return nil, fmt.Errorf("%w: %s", errors.ErrorConfig, "loss db host")
	}
	if name, _ = m.Data["name"].(string); name == "" {
		return nil, fmt.Errorf("%w: %s", errors.ErrorConfig, "loss db name")
	}
	if user, _ = m.Data["user"].(string); user == "" {
		return nil, fmt.Errorf("%w: %s", errors.ErrorConfig, "loss username")
	}
	if pass, _ = m.Data["pass"].(string); pass == "" {
		return nil, fmt.Errorf("%w: %s", errors.ErrorConfig, "loss password")
	}
	if port, _ = m.Data["port"].(string); port == "" {
		port = "3306"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, pass, host, port, name)

	if client, err := sql.Open("mysql", dsn); err == nil {
		// See "Important settings" section.
		client.SetMaxIdleConns(10)
		client.SetMaxOpenConns(100)
		client.SetConnMaxLifetime(time.Hour)
		return client, nil
	} else {
		return nil, fmt.Errorf("%w: %v", errors.ErrorConnect, err)
	}
}
