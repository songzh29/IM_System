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
	ClientMsgID string `json:"client_msg_id"`
	ToUserID    uint   `json:"to_user_id"`
	Content     string `json:"content"`
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
		//查看用户是否存在
		targetUser, err := repository.GetUserByUserID(m.ToUserID)
		if err != nil {
			fmt.Println("用户不存在")
			continue
		}

		//查看用户之间有没有会话
		convID, err := repository.CheckConversationExist(c.UserID, targetUser.ID, msgType)
		if err != nil {
			fmt.Println("查找对话失败")
		}
		//没有对话则创建对话
		if convID == 0 {
			fmt.Println("两人之间无对话，正在创建对话")
			convName := fmt.Sprintf("%d & %d", c.UserID, targetUser.ID)
			conv := model.Conversation{Type: msgType, Name: convName, CreatorID: c.UserID}
			err := repository.CreateConversation(&conv)
			if err != nil {
				fmt.Println("创建对话失败")
				continue
			}
			//将两人加入会话成员表（拉入会话）
			err = repository.AddMember(conv.ID, c.UserID, 0)
			if err != nil {
				fmt.Printf("%d 加入会话失败", c.UserID)
				continue
			}
			err = repository.AddMember(conv.ID, targetUser.ID, 0)
			if err != nil {
				fmt.Printf("%d 加入会话失败", targetUser.ID)
				continue
			}
		}

		//消息入库
		sendMsg := model.Message{ConversationID: convID, SenderID: c.UserID, MsgType: msgType, Content: m.Content}
		err = repository.CreateMessage(&sendMsg)
		if err != nil {
			fmt.Println("消息入库失败")
			continue
		}
		//更新会话表
		err = repository.UpdateConversation(convID, &sendMsg)
		if err != nil {
			fmt.Println("会话表更新失败")
			continue
		}

		// 找目标用户有没有在线
		c.Manager.mu.RLock()
		targetClient, ok := c.Manager.clients[m.ToUserID]
		c.Manager.mu.RUnlock()
		if !ok {
			fmt.Println("用户不在线")
			continue
		}

		// 转发消息
		select {
		case targetClient.Send <- []byte(m.Content):
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
