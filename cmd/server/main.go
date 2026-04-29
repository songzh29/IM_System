package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/songzh29/IM_System/config"
	"github.com/songzh29/IM_System/internal/handler"
	"github.com/songzh29/IM_System/internal/mq"
	"github.com/songzh29/IM_System/internal/router"
	"github.com/songzh29/IM_System/internal/ws"
	"github.com/songzh29/IM_System/pkg/logger"
	"github.com/songzh29/IM_System/pkg/metrics"
	mysqldb "github.com/songzh29/IM_System/pkg/mysql"
	"github.com/songzh29/IM_System/pkg/node"
	"github.com/songzh29/IM_System/pkg/rabbitmq"
	redisdb "github.com/songzh29/IM_System/pkg/redis"
	"go.uber.org/zap"
)

func main() {
	// 先初始化配置
	portFlag := flag.Int("port", 0, "override config port")
	flag.Parse()

	err := config.Init()
	if err != nil {
		log.Panic("配置初始化失败", zap.Error(err))
	}

	if *portFlag != 0 {
		config.ConfigInfo.Server.Port = *portFlag
	}
	//初始化日志
	logger.Init()

	//初始化node
	node.Init()

	//mysql连接
	err = mysqldb.Init()
	if err != nil {
		zap.L().Panic("MySQL连接失败", zap.Error(err))
	}

	//redis连接
	err = redisdb.Init()
	if err != nil {
		zap.L().Panic("Redis连接失败", zap.Error(err))
	}

	//rabbitmq相关内容初始化
	err = mq.Start()
	if err != nil {
		zap.L().Panic("RabbitMQ初始化失败", zap.Error(err))
	}

	//启动Prometheus
	metrics.Init()

	// 注册跨节点消息发布函数，解决循环导入问题
	ws.RegisterPublishForward(router.PublishForward)

	// 启动Redis订阅
	go router.StartSubscribe()

	go func() {
		ctx := context.Background()
		var count float64 = 0
		// 每次扫描 1000 个 key
		iter := redisdb.Rdb.Scan(ctx, 0, "online:user:*", 1000).Iterator()

		// 循环迭代直到结束
		for iter.Next(ctx) {
			count++
		}

		if err := iter.Err(); err != nil {
			fmt.Println("查询 Redis 失败:", err)
			return
		}

		// 将统计到的数量设置到 Prometheus 的 Gauge 中
		metrics.OnlineUsersRedis.Set(count)
	}()

	//gin连接
	r := gin.Default()
	r.LoadHTMLGlob("../../template/*.html")

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

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
	// err = r.Run(serverPort)
	// if err != nil {
	// 	logger.Panic("端口启动失败", zap.Error(err))
	// }

	// 1. 创建原生的 http.Server 对象
	srv := &http.Server{
		Addr:    serverPort,
		Handler: r,
	}
	// 2. 开启一个 Goroutine 异步启动服务
	// 为什么？因为 ListenAndServe 是阻塞的，如果不放进 goroutine，代码就会卡在这里，无法执行后面的监听关闭逻辑。
	go func() {
		zap.L().Info("服务器启动", zap.String("port", serverPort))
		// ErrServerClosed 是正常关闭服务器时返回的错误，不算真正的报错
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Info("服务器异常崩溃", zap.Error(err))
		}
	}()

	// 3. 准备接收操作系统的退出信号
	// 创建一个接收信号的通道 (channel)
	quit := make(chan os.Signal, 1)
	// 告诉 signal 包，当收到 SIGINT(Ctrl+C) 或 SIGTERM(Kill命令) 时，把信号塞进 quit 通道
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 4. 阻塞主线程，死等操作系统的关闭信号
	<-quit
	zap.L().Info("接收到关闭信号，准备打烊，正在处理最后的请求...")

	// 5. 创建一个带超时的 Context (例如 5 秒)
	// 意思是：我最多给正在处理的请求 5 秒钟的时间。5 秒一到，不管有没有处理完，立刻强制关机。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // 释放 context 资源

	// 6. 优雅关闭 HTTP 服务 (不再接收新请求，等待老请求处理完毕)
	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Error("HTTP服务关闭失败:", zap.Error(err))
	}

	// 7. 关闭websocket
	ws.Manager.CloseAll()

	// 8. 清理底层资源 (先关 HTTP，最后关数据库)
	zap.L().Info("HTTP 服务已关闭，开始清理底层资源...")

	// 这里调用 mysql、rabbitmq、redis 包里的关闭方法
	err = rabbitmq.Close()
	if err != nil {
		zap.L().Error("Rabbitmq服务关闭失败:", zap.Error(err))
	}
	//关闭pub/sub
	err = router.StopSubscribe()
	if err != nil {
		zap.L().Error("redis sub服务关闭失败:", zap.Error(err))
	}
	//再关redis
	err = redisdb.Close()
	if err != nil {
		zap.L().Error("redis服务关闭失败:", zap.Error(err))
	}
	err = mysqldb.Close()
	if err != nil {
		zap.L().Error("MySQL服务关闭失败:", zap.Error(err))
	}
	zap.L().Info("所有资源清理完毕，服务器安全退出。拜拜！")

}
