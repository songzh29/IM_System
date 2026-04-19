package mq

import (
	"github.com/songzh29/IM_System/pkg/rabbitmq"
)

func DeclareTopology() error {

	//创建rabbit信道
	ch, err := rabbitmq.GetChannel()
	if err != nil {
		return err
	}
	defer ch.Close()
	// ch.ExchangeDeclare(
	// 	"",
	// 	"fanout",
	// 	true,
	// 	false,
	// 	false,
	// 	false,
	// 	nil,
	// )

	_, err = ch.QueueDeclare(
		"q.conversation.update", // name
		true,                    // durability
		false,                   // delete when unused
		false,                   // exclusive
		false,                   // no-wait
		nil,
	)
	return err

}
