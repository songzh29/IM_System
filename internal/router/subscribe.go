package router

import (
	"context"
	"encoding/json"

	"github.com/songzh29/IM_System/pkg/metrics"

	"github.com/redis/go-redis/v9"
	"github.com/songzh29/IM_System/internal/ws"
	"github.com/songzh29/IM_System/pkg/node"
	redisdb "github.com/songzh29/IM_System/pkg/redis"
	"go.uber.org/zap"
)

var Pubsub *redis.PubSub

// 开启redis订阅
func StartSubscribe() {
	Pubsub = redisdb.Rdb.Subscribe(context.Background(), ForwardChannel)
	ch := Pubsub.Channel()
	nodeID := node.GetNodeID()
	zap.L().Info("已经成功订阅Redis频道",
		zap.String("node_id", nodeID),
		zap.String("channel", ForwardChannel),
	)
	for msg := range ch {
		handlerForwardMsg(msg)
	}
}

func handlerForwardMsg(msg *redis.Message) {
	// sub数+1
	metrics.ForwardReceivedTotal.Inc()

	msgBody := []byte(msg.Payload)
	recieveMsg := ForwardMessage{}
	if err := json.Unmarshal([]byte(msgBody), &recieveMsg); err != nil {
		zap.L().Error("ForwardMessage 反序列化失败", zap.Error(err))
		return
	}
	if recieveMsg.TargetNodeID != node.GetNodeID() {
		//不是发给我的，不管
		return
	}
	ws.DeliverTotalMsgByUserID(recieveMsg.TargetUserID, recieveMsg.Payload)
}

// 关闭redis订阅
func StopSubscribe() error {
	if Pubsub != nil {
		return Pubsub.Close()
	}
	return nil
}
