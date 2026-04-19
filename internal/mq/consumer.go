package mq

import (
	"encoding/json"

	"github.com/songzh29/IM_System/internal/model"
	"github.com/songzh29/IM_System/internal/repository"
	"github.com/songzh29/IM_System/pkg/rabbitmq"
	"go.uber.org/zap"
)

func StartConversationUpdateConsumer() error {
	ch, err := rabbitmq.GetChannel() // 自己开一个
	if err != nil {
		return err
	}
	msgs, err := ch.Consume("q.conversation.update",
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}
	go func() {
		for msg := range msgs {
			var MsgSent model.MqMessage
			err := json.Unmarshal(msg.Body, &MsgSent)
			if err != nil {
				zap.L().Error("JSON反序列化失败", zap.Error(err))
				continue
			}
			err = repository.UpdateConversation(MsgSent.ConvID, &MsgSent.SendMsg)
			if err != nil {
				zap.L().Error("更新消息表出错", zap.Error(err))
			}
		}
	}()
	return nil
}
