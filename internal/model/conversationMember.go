package model

import "time"

type ConversationMember struct {
	ID             uint      `gorm:"column:id;primarykey"`
	ConversationID uint      `gorm:"column:conversation_id"`
	UserID         uint      `gorm:"column:user_id"`
	Role           int       `gorm:"column:role"`
	LastReadMsgID  uint      `gorm:"column:last_read_msg_id"`
	JoinedAt       time.Time `gorm:"column:joined_at"`
}
