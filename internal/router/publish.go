package router

import (
	"context"
	"encoding/json"
	"time"

	redisdb "github.com/songzh29/IM_System/pkg/redis"
)

func PublishForward(targetUserID uint, targetNodeID string, payload []byte) error {
	forwardMsg := ForwardMessage{TargetUserID: targetUserID, TargetNodeID: targetNodeID, Payload: payload}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	data, err := json.Marshal(forwardMsg)
	if err != nil {
		return err
	}
	return redisdb.Rdb.Publish(ctx, ForwardChannel, data).Err()
}
