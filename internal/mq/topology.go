package mq

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/songzh29/IM_System/pkg/rabbitmq"
)

func DeclareTopology() error {

	//创建rabbit信道
	ch, err := rabbitmq.GetChannel()
	if err != nil {
		return err
	}
	defer ch.Close()
	dlxName := "ex.conversation.update.dlx"
	dlqName := "q.conversation.update.dlq"
	dlxRoutingKey := "conversation.update.dead"
	//声明dlx
	err = ch.ExchangeDeclare(dlxName,
		"direct",
		true,
		false,
		false,
		false,
		nil)
	if err != nil {
		return err
	}
	//声明dlq
	_, err = ch.QueueDeclare(dlqName, true, false, false, false, nil)
	if err != nil {
		return err
	}
	//绑定dlq与dlx
	err = ch.QueueBind(dlqName, dlxRoutingKey, dlxName, false, nil)
	if err != nil {
		return err
	}

	_, err = ch.QueueDeclare(
		"q.conversation.update", // name
		true,                    // durability
		false,                   // delete when unused
		false,                   // exclusive
		false,                   // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    dlxName,       // 指定死信交换机
			"x-dead-letter-routing-key": dlxRoutingKey, // 指定死信路由键
		},
	)
	if err != nil {
		return err
	}
	return nil

}
