package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/songzh29/IM_System/config"
)

// 不要在包级别初始化
func getSecret() []byte {
	return []byte(config.ConfigInfo.JWT.Secret)
}

// 生成token，存入userID
func GenerateToken(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Duration(config.ConfigInfo.JWT.Expire) * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getSecret())
}

// 解析token，返回userID
func ParseToken(tokenStr string) (uint, error) {
	// 声明解译出来的载体
	clamis := jwt.MapClaims{}
	//解析token
	token, err := jwt.ParseWithClaims(tokenStr, clamis, func(token *jwt.Token) (interface{}, error) {
		// 确认签名算法是我们期望的 HS256，防止算法篡改攻击
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("非法加密")
		}
		return getSecret(), nil
	})
	if err != nil {
		return 0, err
	}
	if token.Valid {
		userID, ok := clamis["user_id"].(float64)
		if !ok {
			return 0, errors.New("非法ID")
		}
		return uint(userID), nil
	}
	return 0, errors.New("非法token")
}
