package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/songzh29/IM_System/config"
	"github.com/songzh29/IM_System/internal/handler"
	mysqldb "github.com/songzh29/IM_System/pkg/mysql"
	redisdb "github.com/songzh29/IM_System/pkg/redis"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("zap初始化失败: %v", err)
	}

	// ✅ 先初始化配置
	err = config.Init()
	if err != nil {
		logger.Panic("配置初始化失败", zap.Error(err))
	}

	//mysql连接
	err = mysqldb.Init()
	if err != nil {
		logger.Panic("MySQL连接失败", zap.Error(err))
	}

	//redis连接
	err = redisdb.Init()
	if err != nil {
		logger.Panic("Redis连接失败", zap.Error(err))
	}

	//gin连接
	r := gin.Default()
	r.LoadHTMLGlob("../../template/*.html")

	public := r.Group("/public")
	{
		public.GET("/", func(ctx *gin.Context) {
			ctx.JSON(200, gin.H{"msg": "hello!!"})
		})

		public.GET("/register", func(ctx *gin.Context) {
			ctx.JSON(200, gin.H{"msg": "欢迎来到注册界面"})
		})

		public.GET("/login", func(ctx *gin.Context) {
			ctx.JSON(200, gin.H{"msg": "欢迎来到登录界面"})
		})

		public.POST("/register", handler.Register)

		public.POST("/login", handler.Login)

	}
	private := r.Group("/private")
	private.Use(handler.AuthMiddleware())
	{
		private.POST("/profile", func(ctx *gin.Context) {
			ctx.JSON(200, gin.H{"msg": "查看个人信息成功"})
		})
		private.GET("/ws", handler.WsConnect)
	}

	serverPort := fmt.Sprintf(":%d", config.ConfigInfo.Server.Port)
	err = r.Run(serverPort)
	if err != nil {
		logger.Panic("端口启动失败", zap.Error(err))
	}

}
