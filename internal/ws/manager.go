package ws

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/songzh29/IM_System/pkg/metrics"
	"github.com/songzh29/IM_System/pkg/node"
	redisdb "github.com/songzh29/IM_System/pkg/redis"
	"go.uber.org/zap"
)

type ConnManager struct {
	clients map[uint]*Client
	mu      sync.RWMutex
}

// PublishForwardFunc 是用于发布跨节点消息的回调函数类型
type PublishForwardFunc func(targetUserID uint, targetNodeID string, payload []byte) error

// publishForwardImpl 存储注册的跨节点消息发送函数
var publishForwardImpl PublishForwardFunc

// 全局唯一的ConnManager
var Manager = &ConnManager{
	clients: make(map[uint]*Client),
}

// RegisterPublishForward 注册跨节点消息发布函数
// 这个函数应该在应用初始化时由router包调用
func RegisterPublishForward(fn PublishForwardFunc) {
	publishForwardImpl = fn
}

// PublishForwardMsg 使用注册的发送函数来发布跨节点消息
func PublishForwardMsg(targetUserID uint, targetNodeID string, payload []byte) error {
	if publishForwardImpl == nil {
		return fmt.Errorf("PublishForward函数注册失败")
	}
	return publishForwardImpl(targetUserID, targetNodeID, payload)
}

func (m *ConnManager) Register(c *Client) {
	m.mu.Lock()
	// 在这里把 Client 存入 map
	// 这样 Manager 就知道了它的“下属”是谁
	m.clients[c.UserID] = c
	m.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("online:user:%d", c.UserID)
	value := node.GetNodeID() // 比如 "localhost:8080"
	err := redisdb.Rdb.Set(ctx, key, value, 90*time.Second).Err()
	if err != nil {
		// 本地已经注册了,这里只打日志不 return
		zap.L().Error("写在线状态到 Redis 失败", zap.Uint("user_id", c.UserID), zap.Error(err))
	}
	//设置在线人数+1
	metrics.OnlineUsers.Inc()
}

func (m *ConnManager) Unregister(c *Client) {
	m.mu.Lock()
	// 判断是否存在
	existingClient, ok := m.clients[c.UserID]
	if !ok || existingClient != c {
		// 已经被新连接顶替了,什么都不做
		m.mu.Unlock()
		return
	}
	delete(m.clients, c.UserID)
	m.mu.Unlock()
	//redis删除用户,只有"我确实是当前 active client"才清理 Redis
	key := fmt.Sprintf("online:user:%d", c.UserID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := redisdb.Rdb.Del(ctx, key).Err()
	if err != nil {
		zap.L().Error("Redis删除用户在线状态失败", zap.Uint("user_id", c.UserID), zap.Error(err))
	}
	//设置在线人数-1
	metrics.OnlineUsers.Dec()
}

func (m *ConnManager) CloseAll() {
	m.mu.Lock()
	if len(m.clients) == 0 {
		m.mu.Unlock()
		return
	}

	// 收集 keys
	keys := make([]string, 0, len(m.clients))
	conns := make([]*websocket.Conn, 0, len(m.clients))
	for userID, client := range m.clients {
		keys = append(keys, fmt.Sprintf("online:user:%d", userID))
		conns = append(conns, client.Conn)
	}
	m.mu.Unlock()

	// 批量 DEL
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisdb.Rdb.Del(ctx, keys...).Err(); err != nil {
		zap.L().Error("批量清理在线状态失败", zap.Error(err))
	}

	// 锁外:关连接
	for _, conn := range conns {
		conn.Close()
	}

}

// DeliverTotalMsgByUserID 通过用户ID将消息转发给用户
// 这个函数定义在manager中，避免circular import
func DeliverTotalMsgByUserID(targetUserID uint, msg []byte) bool {
	client, exists := Manager.GetClientByUserID(targetUserID)
	if !exists {
		// 用户不在线
		return false
	}
	return DeliverTotalMsg(client, msg)
}

// GetClientByUserID 根据用户ID获取连接的客户端
func (m *ConnManager) GetClientByUserID(userID uint) (*Client, bool) {
	m.mu.RLock()
	client, exists := m.clients[userID]
	m.mu.RUnlock()
	return client, exists
}

// 在 manager.go
func (m *ConnManager) SnapshotSendChanUsage(observer func(usage float64)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.clients {
		observer(float64(len(c.Send)))
	}
}
