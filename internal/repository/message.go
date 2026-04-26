package repository

import (
	"errors"

	"github.com/songzh29/IM_System/internal/model"
	mysqldb "github.com/songzh29/IM_System/pkg/mysql"
	"gorm.io/gorm"
)

func CreateMessage(msg *model.Message) error {
	result := mysqldb.DB.Create(msg)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			// 幂等返回，但要把 msg.ID 查回来供上游使用
			existing := &model.Message{}
			if err := mysqldb.DB.Where("client_msg_id = ?", msg.ClientMsgID).First(existing).Error; err != nil {
				return err
			}
			*msg = *existing // 把查出来的 ID 等字段回填
			return nil
		}
		return result.Error
	}
	return nil
}

// // userID是自己
// func GetUnreadMessages(userID uint, conversationID uint, lastReadMsgID uint) ([]model.Message, error) {
// 	var unreadMsg []model.Message
// 	result := mysqldb.DB.Where("sender_id != ? AND conversation_id = ? AND id > ?", userID, conversationID, lastReadMsgID).Find(&unreadMsg)
// 	if result.Error != nil {
// 		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
// 			return nil, nil // 无未读消息
// 		}
// 		return nil, result.Error // 真正的数据库错误
// 	}
// 	return unreadMsg, nil
// }
