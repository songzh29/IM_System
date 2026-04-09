package ws

import (
	"sync"
)

type ConnManager struct {
	clients map[uint]*Client
	mu      sync.RWMutex
}

// 全局唯一的ConnManager
var Manager = &ConnManager{
	clients: make(map[uint]*Client),
}

func (m *ConnManager) Register(c *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 在这里把 Client 存入 map
	// 这样 Manager 就知道了它的“下属”是谁
	m.clients[c.UserID] = c
}

func (m *ConnManager) Unregister(c *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 判断是否存在
	if existingClient, ok := m.clients[c.UserID]; ok {
		// 确保是同一个连接（防止覆盖问题）
		if existingClient == c {
			delete(m.clients, c.UserID)
		}
	}
}

func (m *ConnManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, client := range m.clients {
		client.Conn.Close()
	}
}
