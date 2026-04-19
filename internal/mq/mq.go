package mq

import "github.com/songzh29/IM_System/pkg/rabbitmq"

func Start() error {
	// 初始化rabbitmq服务
	err := rabbitmq.Init()
	if err != nil {
		return err
	}
	// 初始化rabbitmq拓扑结构
	err = DeclareTopology()
	if err != nil {
		return err
	}
	// 初始化生产者的channel
	err = InitPublisher()
	if err != nil {
		return err
	}
	return nil

}
