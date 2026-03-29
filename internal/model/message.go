package model

import "time"

type Message struct {
	ID             uint
	ConversationID uint
	SenderID       uint
	MsgType        int
	Content        string `json:"content"`
	ClientMsgID    string
	Status         string
	CreatedAt      time.Time
}
