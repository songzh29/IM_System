package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/songzh29/IM_System/config"
)

var Conn *amqp.Connection

func Init() error {
	return Reconnect()
}

func Reconnect() error {
	if Conn != nil && !Conn.IsClosed() {
		Conn.Close()
	}
	host := config.ConfigInfo.RabbitMQ.Host
	password := config.ConfigInfo.RabbitMQ.Password
	port := config.ConfigInfo.RabbitMQ.Port
	user := config.ConfigInfo.RabbitMQ.User
	vhost := config.ConfigInfo.RabbitMQ.Vhost
	var err error
	addr := fmt.Sprintf("amqp://%s:%s@%s:%d%s", user, password, host, port, vhost)
	Conn, err = amqp.Dial(addr)
	if err != nil {
		return err
	}
	return nil
}

func Close() error {
	if Conn == nil || Conn.IsClosed() {
		return nil
	}
	return Conn.Close()
}

// 新增一个工具函数：每次需要用的时候，动态申请一个 Channel
func GetChannel() (*amqp.Channel, error) {
	if Conn == nil || Conn.IsClosed() {
		return nil, fmt.Errorf("rabbitmq tcp connection is not ready")
	}
	// 瞬间就能创建好，极其轻量
	return Conn.Channel()
}
