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
			return nil, nil // 会话不存在
		}
		return nil, result.Error // 真正的数据库错误
	}
	// 查询成功
	return conv, nil
}
