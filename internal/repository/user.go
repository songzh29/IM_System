package repository

import (
	"errors"

	"github.com/songzh29/IM_System/internal/model"
	mysqldb "github.com/songzh29/IM_System/pkg/mysql"
	"gorm.io/gorm"
)

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
			return nil, nil // 用户不存在
		}
		return nil, result.Error // 真正的数据库错误
	}
	// 查询成功
	return user, nil
}

// 根据用户ID查找用户
func GetUserByUserID(userID uint) (*model.User, error) {
	user := &model.User{}
	result := mysqldb.DB.First(user, "id = ?", userID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // 用户不存在
		}
		return nil, result.Error // 真正的数据库错误
	}
	// 查询成功
	return user, nil
}
