package logger

import (
	"fmt"
	"os"

	"github.com/songzh29/IM_System/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2" // ← 日志切割库
)

var L *zap.Logger

func Init() {
	level := config.ConfigInfo.Zap.Level
	isDev := config.ConfigInfo.Zap.IsDev

	// 1. 配置编码器
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	var consoleEncoder, fileEncoder zapcore.Encoder
	if isDev {
		// 开发环境:控制台用彩色
		consoleEncoderConfig := encoderConfig
		consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		consoleEncoder = zapcore.NewConsoleEncoder(consoleEncoderConfig)
		// 文件用纯文本
		fileEncoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		// 生产环境:都用 JSON
		consoleEncoder = zapcore.NewJSONEncoder(encoderConfig)
		fileEncoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 2. 解析日志级别
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zap.InfoLevel
	}
	filename := fmt.Sprintf("logs/im_system_%d.log", config.ConfigInfo.Server.Port)

	// 3. 文件 writer(带切割)
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filename, // 日志文件路径
		MaxSize:    100,      // 单文件最大 MB
		MaxBackups: 10,       // 保留最多旧文件数
		MaxAge:     7,        // 保留天数
		Compress:   true,     // 是否压缩
	})

	// 4. 构建多 Core
	fileCore := zapcore.NewCore(fileEncoder, fileWriter, zapLevel)
	// 控制台只打 Warn 及以上
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.ErrorLevel)

	core := zapcore.NewTee(fileCore, consoleCore) // ← 关键:Tee 把日志同时分发到两个 core

	// 5. 生成实例
	L = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(0))
	zap.ReplaceGlobals(L)
}

func Sync() {
	_ = L.Sync()
}
