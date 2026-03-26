package redisdb

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/songzh29/IM_System/config"
)

var Rdb *redis.Client

func Init() error {
	host := config.ConfigInfo.Redis.Host
	password := config.ConfigInfo.Redis.Password
	port := config.ConfigInfo.Redis.Port

	addr := fmt.Sprintf("%s:%d", host, port)
	Rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
	})
	err := Rdb.Ping(context.Background()).Err()
	return err
}
