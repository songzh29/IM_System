package mq

import (
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/songzh29/IM_System/internal/model"
	"github.com/songzh29/IM_System/pkg/rabbitmq"
)

// Publisher 持有自己的 Channel
var publishCh *amqp.Channel

func InitPublisher() error {
	var err error
	publishCh, err = rabbitmq.GetChannel() // 自己开一个
	return err
}

func PublishMessageSent(msg *model.MqMessage) error {
	mqMsg, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return publishCh.Publish("",
		"q.conversation.update",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        mqMsg},
	)
}
