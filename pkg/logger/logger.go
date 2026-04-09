package logger

import (
	"os"

	"github.com/songzh29/IM_System/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// L 是全局 Logger，在业务代码中直接 logger.L.Info(...) 调用
var L *zap.Logger

// Init 初始化日志配置
func Init() {
	level := config.ConfigInfo.Zap.Level
	isDev := config.ConfigInfo.Zap.IsDev

	var encoder zapcore.Encoder

	// 1. 配置编码器
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder   // 时间格式
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder // 级别大写

	if isDev {
		// 开发环境：彩色终端输出
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		// 生产环境：标准 JSON 输出
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 2. 设置日志级别
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zap.InfoLevel
	}

	// 3. 构建 Core
	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapLevel)

	// 4. 生成实例
	L = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(0))
	zap.L()

	// 替换官方全局变量,以后可直接通过zap.L()使用
	zap.ReplaceGlobals(L)
}

// Sync 刷新缓冲区，建议在 main 函数退出前执行
func Sync() {
	_ = L.Sync()
}
