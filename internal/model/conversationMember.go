package model

import "time"

type ConversationMember struct {
	ID             uint
	ConversationID uint
	UserID         uint
	Role           int
	LastReadMsgID  uint
	JoinedAt       time.Time
}
