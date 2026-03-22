package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/songzh29/IM_System/config"
	"go.uber.org/zap"
)

func main() {

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("zap初始化失败: %v", err)
	}
	//gin连接
	err = config.Init()
	if err != nil {
		logger.Panic("配置初始化失败", zap.Error(err))
	}

	r := gin.Default()

	r.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"msg": "hello world"})
	})

	serverPort := fmt.Sprintf(":%d", config.ConfigInfo.Server.Port)
	err = r.Run(serverPort)
	if err != nil {
		logger.Panic("端口启动失败", zap.Error(err))
	}

}
