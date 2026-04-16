package model

import "time"

type Conversation struct {
	ID             uint       `gorm:"column:id;primaryKey" redis:"id"`
	Type           int        `gorm:"column:type" redis:"type"`
	Name           string     `gorm:"column:name" redis:"name"`
	CreatorID      uint       `gorm:"column:creator_id" redis:"creator_id"`
	LastMsgID      *uint      `gorm:"column:last_msg_id" redis:"last_msg_id"`
	LastMsgContent *string    `gorm:"column:last_msg_content" redis:"last_msg_content"`
	LastMsgTime    *time.Time `gorm:"column:last_msg_time" redis:"last_msg_time"`
	CreatedAt      time.Time  `redis:"created_at"`
}
