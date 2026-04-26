package node

import (
	"fmt"
	"os"

	"github.com/songzh29/IM_System/config"
	"go.uber.org/zap"
)

var nodeID string

func Init() {
	hostname, err := os.Hostname()
	if err != nil {
		zap.L().Error("获取 hostname 失败", zap.Error(err))
		hostname = "unknown"
	}
	port := config.ConfigInfo.Server.Port
	nodeID = fmt.Sprintf("%s:%d", hostname, port)
	zap.L().Info("node_id 初始化完成", zap.String("node_id", nodeID))
}

func GetNodeID() string {
	return nodeID
}
