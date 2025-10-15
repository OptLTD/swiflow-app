package entity

import (
	"time"

	"gorm.io/gorm"
)

type MsgEntity struct {
	ID uint `gorm:"primarykey"`

	IsSend  bool   `gorm:"-:all"`
	Group   string `gorm:"group;size:36;default ''"`
	OpType  string `gorm:"op_type;size:16;not null"`
	TaskId  string `gorm:"task_id;size:36;not null"`
	UniqId  string `gorm:"uniq_id;size:36;uniqueIndex"`
	PrevId  string `gorm:"prev_id;size:36;default null"`
	Request string `gorm:"request;"`
	Respond string `gorm:"respond;"`
	Context string `gorm:"context;"`

	RecvAt *time.Time `gorm:"recv_at;"`
	SendAt *time.Time `gorm:"send_at;"`

	gorm.Model `json:"-"`
}

func (m *MsgEntity) TableName() string {
	return "llm_msg"
}
