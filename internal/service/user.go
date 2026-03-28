package service

import (
	"errors"

	"github.com/songzh29/IM_System/internal/model"
	"github.com/songzh29/IM_System/internal/respository"
	"golang.org/x/crypto/bcrypt"
)

// 注册：检查用户名是否存在，密码加密，调用repository创建
func Register(username, password string) error {
	//先检查用户存不存在
	user, err := respository.GetUserByUsername(username)
	if err != nil {
		return err
	}
	if user != nil {
		return errors.New("用户名已被使用")
	}

	//没有用户则创建用户,对密码加密
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	//把用户信息存入MySQL，密码已加密
	userData := model.User{Username: username, Password: string(passwordHash)}
	if err := respository.CreateUser(&userData); err != nil {
		return err
	}
	return nil

}

func Login(username, password string) (uint, error) {
	//查找用户是否存在
	user, err := respository.GetUserByUsername(username)
	if err != nil {
		return 0, errors.New("用户未注册，登录失败")
	}
	// 用户不存在返回错误
	if user == nil {
		err = errors.New("用户未注册，登录失败")
		return 0, err
	}
	//输错密码返回错误
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		err = errors.New("密码输入错误，请重试")
	}
	return user.ID, nil

}
