package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/songzh29/IM_System/internal/model"
	"github.com/songzh29/IM_System/internal/repository"
	"github.com/songzh29/IM_System/pkg/rabbitmq"
	"go.uber.org/zap"
)

var ErrPublishTimeOut = errors.New("confirm 超时或失败")
var ErrPublishNACK = errors.New("broker NACK")

// Publisher 持有自己的 Channel
var (
	publishCh *amqp.Channel
	publishMu sync.Mutex
)

func InitPublisher() error {
	var err error
	publishCh, err = rabbitmq.GetChannel() // 自己开一个
	if err != nil {
		return err
	}
	err = publishCh.Confirm(false)
	if err != nil {
		return err
	}
	return nil
}

func PublishMessageSent(msg *model.MqMessage) error {
	mqMsg, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	publishMu.Lock()
	defer publishMu.Unlock()
	conf, err := publishCh.PublishWithDeferredConfirm("",
		"q.conversation.update",
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         mqMsg,
		},
	)
	if err != nil {
		return err
	}

	// 阻塞等 broker 确认
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ok, err := conf.WaitContext(ctx)
	if err != nil {
		// ctx.DeadlineExceeded 或者 channel closed
		return fmt.Errorf("%w: %v", ErrPublishTimeOut, err)
	}
	if !ok {
		return ErrPublishNACK
	}
	return nil
}

// 如果生产端推送消息到rabbitmq失败则尝试重新发送三次，失败后就直接用MySQL进行操作
func PublishWithFallback(mqMsg *model.MqMessage) {
	const maxRetries = 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := PublishMessageSent(mqMsg)
		if err == nil {
			return
		}
		if errors.Is(err, ErrPublishNACK) {
			zap.L().Error("broker NACK，不重试", zap.Error(err))
			break
		}
		if attempt < maxRetries {
			backoff := time.Duration(100*(1<<(attempt-1))) * time.Millisecond
			zap.L().Warn("publish 失败，准备重试",
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoff),
				zap.Error(err))
			time.Sleep(backoff)
		} else {
			zap.L().Error("publish 最后一次尝试仍失败",
				zap.Int("attempt", attempt),
				zap.Error(err))
		}
	}
	// 三次全失败，降级同步写 DB
	zap.L().Error("MQ 三次重试均失败，降级同步写 DB")
	if err := repository.UpdateConversation(mqMsg.ConvID, &mqMsg.SendMsg); err != nil {
		zap.L().Error("降级同步写 DB 也失败", zap.Error(err))
	}
}
