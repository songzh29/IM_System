package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/songzh29/IM_System/internal/model"
	mysqldb "github.com/songzh29/IM_System/pkg/mysql"
	redisdb "github.com/songzh29/IM_System/pkg/redis"
	"gorm.io/gorm"
)

func CreateConversation(conv *model.Conversation) error {
	result := mysqldb.DB.Create(conv)
	return result.Error
}

func CacheConversationID(useridA uint, useridB uint, convtype uint, convID uint) error {
	//强制排序，ID小的在前，消除方向性差异
	minID, maxID := useridA, useridB
	if minID > maxID {
		minID, maxID = useridB, useridA
	}

	redisKey := fmt.Sprintf("conv:exist:type:%d:user:%d_%d", convtype, minID, maxID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // 释放 context 资源

	return redisdb.Rdb.Set(ctx, redisKey, strconv.Itoa(int(convID)), 24*time.Hour).Err()
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
