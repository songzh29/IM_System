package model

import "time"

type Conversation struct {
	ID             uint       `gorm:"column:id;primaryKey"`
	Type           int        `gorm:"column:type"`
	Name           string     `gorm:"column:name"`
	CreatorID      uint       `gorm:"column:creator_id"`
	LastMsgID      *uint      `gorm:"column:last_msg_id"`
	LastMsgContent *string    `gorm:"column:last_msg_content"`
	LastMsgTime    *time.Time `gorm:"column:last_msg_time"`
	CreatedAt      time.Time
}
