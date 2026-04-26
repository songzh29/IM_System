package mq

import (
	"errors"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/songzh29/IM_System/pkg/rabbitmq"
	"go.uber.org/zap"
)

// 监听 Connection 断开并触发重连
func WatchAndReconnect() {
	for {
		// 1. 注册 NotifyClose，监听当前 Connection
		notifyClose := make(chan *amqp.Error, 1)
		rabbitmq.Conn.NotifyClose(notifyClose)

		// 2. 阻塞等待断开信号
		err := <-notifyClose
		if err == nil {
			// 主动 Close（进程退出时），不重连
			return
		}

		zap.L().Error("MQ 连接断开，开始重连", zap.Error(err))

		// 3. 执行重连（带退避）
		if reconnectErr := reconnectWithBackoff(); reconnectErr != nil {
			zap.L().Error("MQ 重连失败，触发进程退出")
			syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
			return
		}

		zap.L().Info("MQ 重连成功")
		// 循环继续，重新注册 NotifyClose
	}
}

// 指数退避重连
func reconnectWithBackoff() error {
	backoffs := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		30 * time.Second,
	}
	for i, wait := range backoffs {
		zap.L().Info("尝试重连", zap.Int("attempt", i+1))
		if err := rabbitmq.Reconnect(); err != nil {
			zap.L().Warn("重连失败", zap.Error(err), zap.Duration("next_backoff", wait))
			time.Sleep(wait)
			continue
		}

		// Reconnect 成功后，重建业务层
		if err := rebuildAll(); err != nil {
			zap.L().Warn("业务层重建失败", zap.Error(err))
			time.Sleep(wait)
			continue
		}

		return nil // 成功
	}
	return errors.New("重连耗尽次数")
}

// 重建业务层：拓扑 + Publisher + Consumer
func rebuildAll() error {
	if err := DeclareTopology(); err != nil {
		return err
	}
	if err := InitPublisher(); err != nil {
		return err
	}
	if err := StartConversationUpdateConsumer(); err != nil {
		return err
	}
	return nil
}
