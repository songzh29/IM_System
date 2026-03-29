package repository

import (
	"errors"
	"time"

	"github.com/songzh29/IM_System/internal/model"
	mysqldb "github.com/songzh29/IM_System/pkg/mysql"
	"gorm.io/gorm"
)

func AddMember(conversationID uint, userID uint, role int) error {
	conMem := model.ConversationMember{ConversationID: conversationID, UserID: userID, Role: role, JoinedAt: time.Now()}
	result := mysqldb.DB.Create(&conMem)
	return result.Error
}

func GetConversationsByUserID(userID uint) ([]model.Conversation, error) {
	var conMem []model.ConversationMember
	result := mysqldb.DB.Select("conversation_id").Where("user_id = ?", userID).Find(&conMem)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // 无未读消息
		}
		return nil, result.Error // 真正的数据库错误
	}
	var convs []model.Conversation
	conResults := mysqldb.DB.Where(conMem).Find(&convs)
	if conResults.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // 无未读消息
		}
		return nil, conResults.Error // 真正的数据库错误
	}
	return convs, nil
}

func UpdateLastReadMsgID(conversationID uint, userID uint, msgID uint) error {
	result := mysqldb.DB.Model(&model.ConversationMember{}).Where("user_id = ? AND conversation_id = ?", userID, conversationID).Update("last_read_msg_id", msgID)
	if result.Error != nil {
		return result.Error // 真正的数据库错误
	}
	if result.RowsAffected == 0 {
		return errors.New("没有更新任何数据")
	}
	return nil
}
