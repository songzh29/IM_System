package model

import "time"

type Conversation struct {
	ID             uint `gorm:"primaryKey"`
	Type           int
	Name           string
	CreatorID      uint
	LastMsgID      uint
	LastMsgContent string
	LastMsgTime    time.Time
	CreatedAt      time.Time
}
