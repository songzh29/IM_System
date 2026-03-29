package ws

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/songzh29/IM_System/internal/model"
	"github.com/songzh29/IM_System/internal/repository"
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

type JsonMessage struct {
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
		// 2. 关闭连接
		c.Conn.Close()
	}()
	for {
		msgType, msg, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}

		// 解析 JSON
		var m JsonMessage
		if err := json.Unmarshal(msg, &m); err != nil {
			fmt.Println("json解析失败:", err)
			continue
		}

		// 找目标用户
		c.Manager.mu.RLock()
		targetClient, ok := c.Manager.clients[m.ToUserID]
		c.Manager.mu.RUnlock()
		//不管用户在不在线都要创建会话,从数据看有没有这个用户，没有则报错
		targetUser, err := repository.GetUserByUserID(targetClient.UserID)
		if err != nil {
			fmt.Println("用户不存在")
			continue
		}
	// 	//创建消息
	// 	msgDb := model.Message{SenderID: c.UserID, MsgType: msgType, Content: m.Content}
	// 	//创建会话
	// 	convName := fmt.Sprintf("%d & %d", c.UserID, targetUser.ID)
	// 	conv := model.Conversation{CreatorID: c.UserID, Type: msgType, Name: convName}
	// 	if !ok {
	// 		fmt.Println("用户不在线")
	// 		//把消息存入MySQL

	// 		continue
	// 	}

	// 	// 转发消息
	// 	select {
	// 	case targetClient.Send <- msg:
	// 	default:
	// 		// 对方阻塞了 → 踢掉
	// 		go c.Manager.Unregister(targetClient)
	// 	}
	// }
}

// 假设从websocket发过来的数据全是JSON
// func (c *Client) DeliverMsg() {
// 	for msg := range c.Send {
// 		err := c.Conn.WriteMessage(websocket.TextMessage, msg)
// 		if err != nil {
// 			return
// 		}
// 	}
// }
