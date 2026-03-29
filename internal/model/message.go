package model

import "time"

type Message struct {
	ID             uint   `gorm:"column:id;primaryKey"`
	ConversationID uint   `gorm:"column:conversation_id"`
	SenderID       uint   `gorm:"column:sender_id"`
	MsgType        int    `gorm:"column:msg_type"`
	Content        string `json:"content"`
	ClientMsgID    string `gorm:"column:client_msg_id"`
	Status         int
	CreatedAt      time.Time
}
