package ws

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

const (
	// 发送缓冲区大小
	SendBufferSize = 256
)

type Client struct {
	UserID  uint
	Conn    *websocket.Conn
	Send    chan []byte
	Manager *ConnManager
}

type Message struct {
	ToUserID uint   `json:"to_user_id"`
	Content  string `json:"content"`
}

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (c *Client) ListenMsg() {
	defer func() {
		// 1. 从管理器注销
		c.Manager.Unregister(c)

		// 2. 关闭发送通道（通知 WritePump 退出）
		close(c.Send)

		// 3. 关闭连接
		c.Conn.Close()
	}()
	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}

		// 解析 JSON
		var m Message
		if err := json.Unmarshal(msg, &m); err != nil {
			fmt.Println("json解析失败:", err)
			continue
		}

		// 找目标用户
		c.Manager.mu.RLock()
		targetClient, ok := c.Manager.clients[m.ToUserID]
		c.Manager.mu.RUnlock()
		if !ok {
			fmt.Println("用户不在线")
			continue
		}

		// 转发消息
		select {
		case targetClient.Send <- msg:
		default:
			// 对方阻塞了 → 踢掉
			go c.Manager.Unregister(targetClient)
		}
	}
}

// 假设从websocket发过来的数据全是JSON
func (c *Client) DeliverMsg() {
	for msg := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			return
		}
	}
}
