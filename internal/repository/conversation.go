package repository

import (
	"errors"

	"github.com/songzh29/IM_System/internal/model"
	mysqldb "github.com/songzh29/IM_System/pkg/mysql"
	"gorm.io/gorm"
)

func CreateConversation(conv *model.Conversation) error {
	result := mysqldb.DB.Create(conv)
	return result.Error
}

func GetConversationByID(convID uint) (*model.Conversation, error) {
	conv := &model.Conversation{}
	result := mysqldb.DB.First(conv, "id = ?", convID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("会话不存在") // 会话不存在
		}
		return nil, result.Error // 真正的数据库错误
	}
	// 查询成功
	return conv, nil
}

func UpdateConversation(convID uint, msg *model.Message) error {
	result := mysqldb.DB.Model(&model.Conversation{}).
		Where("id = ?", convID).
		Updates(map[string]interface{}{
			"last_msg_id":      &msg.ID,
			"last_msg_content": &msg.Content,
			"last_msg_time":    &msg.CreatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("更新了0个字段")
	}
	return nil
}
