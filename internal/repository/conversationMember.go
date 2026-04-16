package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/songzh29/IM_System/internal/model"
	mysqldb "github.com/songzh29/IM_System/pkg/mysql"
	redisdb "github.com/songzh29/IM_System/pkg/redis"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func AddMember(conversationID uint, userID uint, role int) error {
	conMem := model.ConversationMember{ConversationID: conversationID, UserID: userID, Role: role, JoinedAt: time.Now()}
	result := mysqldb.DB.Create(&conMem)
	return result.Error
}

func GetConversationsByUserID(userID uint) ([]model.Conversation, error) {
	var conMem []model.ConversationMember
	result := mysqldb.DB.
		Select("conversation_id").
		Where("user_id = ?", userID).
		Find(&conMem)
	if result.Error != nil {
		return nil, result.Error
	}
	if len(conMem) == 0 {
		return nil, nil
	}

	// 提取ID
	var ids []uint
	for _, m := range conMem {
		ids = append(ids, m.ConversationID)
	}

	var convs []model.Conversation
	conResults := mysqldb.DB.
		Where("id IN ?", ids).
		Find(&convs)

	if conResults.Error != nil {
		return nil, conResults.Error
	}
	if len(convs) == 0 {
		return nil, nil
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

func CheckConversationExist(useridA uint, useridB uint, convtype int) (uint, error) {
	var conv model.Conversation
	var convmem model.ConversationMember

	//强制排序，ID小的在前，消除方向性差异
	minID, maxID := useridA, useridB
	if minID > maxID {
		minID, maxID = useridB, useridA
	}

	redisKey := fmt.Sprintf("conv:exist:type:%d:user:%d_%d", convtype, minID, maxID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // 释放 context 资源

	//先查redis
	val, err := redisdb.Rdb.Get(ctx, redisKey).Result()
	if err == nil {
		// 缓存命中，没有出任何错
		// 拿到真实的 conversation_id，转成 uint 返回
		convID, parseErr := strconv.ParseUint(val, 10, 32)
		if parseErr == nil {
			if convID == 0 {
				return 0, nil //没有对话
			}
			return uint(convID), nil
		}
		zap.L().Error("Redis中会话ID解析失败", zap.Error(parseErr))
	} else if err != redis.Nil {
		// Redis 发生了网络异常，打印日志，降级去查 MySQL（出现了除没查到之外的其他错误）
		zap.L().Error("Redis读取对话缓存失败", zap.Error(err))
	}

	//如果没查到
	result := mysqldb.DB.Select("conversation_id").
		Where("user_id IN ?", []uint{useridA, useridB}).
		Group("conversation_id").
		Having("COUNT(DISTINCT user_id) = 2").
		Find(&convmem)
	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected == 0 {
		return 0, nil
	}

	conRes := mysqldb.DB.First(&conv, "type = ? AND id = ?", convtype, convmem.ConversationID)
	if conRes.Error != nil {
		if errors.Is(conRes.Error, gorm.ErrRecordNotFound) {
			return 0, nil // 对话不存在
		}
		return 0, conRes.Error // 真正的数据库错误
	}

	//写入redis
	if err := redisdb.Rdb.Set(ctx, redisKey, conv.ID, 24*time.Hour).Err(); err != nil {
		zap.L().Error("Redis写入对话失败", zap.Error(err))
	}

	return conv.ID, nil

}

func CheckUnreadMessage(userID uint) error {
	//利用用户的id查看用户的未读消息
	convmem := model.ConversationMember{}
	result := mysqldb.DB.First(&convmem, "user_id = ?", userID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return errors.New("会话记录不存在") // 会话记录不存在
		}
		return result.Error // 真正的数据库错误
	}
	return nil
}

func GetUnreadConversationMsgByUserID(userID uint) ([]model.Message, error) {
	var msgs []model.Message
	result := mysqldb.DB.Table("messages").
		Joins("JOIN conversations AS conv ON messages.conversation_id = conv.id").
		Joins("JOIN conversation_members AS cm on conv.id = cm.conversation_id").
		Where("cm.user_id = ?", userID).
		Where("messages.id > cm.last_read_msg_id").
		Find(&msgs)
	if result.Error != nil {
		return msgs, result.Error
	}
	return msgs, nil
}
