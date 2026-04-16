package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/songzh29/IM_System/internal/model"
	mysqldb "github.com/songzh29/IM_System/pkg/mysql"
	redisdb "github.com/songzh29/IM_System/pkg/redis"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("用户不存在")

// 创建用户（注册时用）
func CreateUser(user *model.User) error {
	result := mysqldb.DB.Create(user)
	return result.Error
}

// 根据用户名查找用户
func GetUserByUsername(username string) (*model.User, error) {
	user := &model.User{}
	result := mysqldb.DB.First(user, "username = ?", username)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound // 用户不存在
		}
		return nil, result.Error // 真正的数据库错误
	}
	// 查询成功
	return user, nil
}

// 根据用户ID查找用户
func GetUserByUserID(userID uint) (*model.User, error) {
	user := &model.User{}

	hashUser := fmt.Sprintf("user:%d", userID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // 释放 context 资源

	// 先查redis,redis有就返回
	hgetAllcmd := redisdb.Rdb.HGetAll(ctx, hashUser)
	if err := hgetAllcmd.Err(); err != nil {
		zap.L().Error("Redis查询用户信息出错:", zap.Error(err))
	} else if len(hgetAllcmd.Val()) > 0 {
		if err := hgetAllcmd.Scan(user); err != nil {
			zap.L().Error("Redis数据Scan出错:", zap.Error(err))
		} else {
			if hgetAllcmd.Val()["id"] == "0" {
				zap.L().Info("Redis没有查到用户", zap.Uint("UserID", userID))
				return nil, nil //设置ID为0并返回，防止穿透
			}
			zap.L().Info("Redis查到了用户", zap.Uint("UserID", userID))
			return user, nil // 缓存命中，直接返回
		}
	}

	// redis没有就查Mysql
	zap.L().Info("MySQL正在查询用户", zap.Uint("UserID", userID))
	result := mysqldb.DB.First(user, "id = ?", userID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			//同样写回redis,设置ID为0防止穿透
			_, err := redisdb.Rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.HSet(ctx, hashUser, model.User{ID: 0})
				pipe.Expire(ctx, hashUser, 5*time.Minute) // 务必设置过期时间！
				return nil
			})
			if err != nil {
				// 缓存写失败不影响业务，打个日志就行
				zap.L().Error("Redis写入用户信息失败:", zap.Error(err))
			}
			return nil, nil // 用户不存在
		}
		return nil, result.Error // 真正的数据库错误
	}

	// 查询成功,先写回redis再return
	// 写回 Redis (使用 Pipeline 保证 HSet 和 Expire 一起执行)
	// 技巧：使用 Pipelined 可以减少一次网络通信开销
	_, err := redisdb.Rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, hashUser, user)
		pipe.Expire(ctx, hashUser, 5*time.Hour) // 务必设置过期时间！
		return nil
	})
	zap.L().Info("用户信息写到了redis", zap.Uint("UserID", userID))

	if err != nil {
		// 缓存写失败不影响业务，打个日志就行
		zap.L().Error("Redis写入用户信息失败:", zap.Error(err))
	}
	return user, nil
}
