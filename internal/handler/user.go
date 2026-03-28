package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songzh29/IM_System/internal/model"
	"github.com/songzh29/IM_System/internal/service"
	"github.com/songzh29/IM_System/pkg/jwt"
)

func Register(c *gin.Context) {
	user := model.User{}
	if err := c.ShouldBind(&user); err != nil {
		c.JSON(400, gin.H{"msg": "用户信息输入错误"})
		return
	}

	err := service.Register(user.Username, user.Password)
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"msg": "注册成功"})

}

// 将用户名和信息进行对比
func Login(c *gin.Context) {
	user := model.User{}
	if err := c.ShouldBind(&user); err != nil {
		c.JSON(400, gin.H{"msg": "服务器出错"})
		return
	}
	uerid, err := service.Login(user.Username, user.Password)
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	//签发JWT并设置cookie
	token, err := jwt.GenerateToken(uerid)
	if err != nil {
		c.JSON(400, gin.H{"msg": "JWT签发失败"})
		return
	}
	// c.SetCookie("token", token, 72*3600, "/", "localhost", false, true)
	c.JSON(200, gin.H{
		"msg":   "登录成功",
		"token": token,
	})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从Header取token
		tokenStr := c.GetHeader("Authorization")
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

		// token为空
		if tokenStr == "" {
			c.AbortWithStatusJSON(401, gin.H{"msg": "未登录"})
			return
		}

		// 解析token
		userID, err := jwt.ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"msg": "token无效"})
			return
		}

		// 存入context，放行
		c.Set("user_id", userID)
		c.Next()
	}
}
