package ws

import (
	"encoding/json"
	"errors"
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

type CollectMessage struct {
	SenderID       uint   `json:"sender_id"`
	ConversationID uint   `json:"conversation_id"`
	Content        string `json:"content"`
	MsgID          uint   `json:"msg_id"`
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
		close(c.Send)
	}()
	for {
		_, msg, err := c.Conn.ReadMessage()
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
			c.Send <- []byte("发送失败")
			continue
		}

		//查看用户之间有没有会话
		convID, err := repository.CheckConversationExist(c.UserID, targetUser.ID, 1)
		if err != nil {
			fmt.Println("查找对话失败")
		}
		//没有对话则创建对话
		if convID == 0 {
			fmt.Println("两人之间无对话，正在创建对话")
			convName := fmt.Sprintf("%d & %d", c.UserID, targetUser.ID)
			conv := model.Conversation{Type: 1, Name: convName, CreatorID: c.UserID}
			err := repository.CreateConversation(&conv)
			if err != nil {
				fmt.Println("创建对话失败")
				c.Send <- []byte("发送失败")
				continue
			}
			convID = conv.ID
			//将两人加入会话成员表（拉入会话）
			err = repository.AddMember(conv.ID, c.UserID, 0)
			if err != nil {
				fmt.Printf("%d 加入会话失败", c.UserID)
				c.Send <- []byte("发送失败")
				continue
			}
			err = repository.AddMember(conv.ID, targetUser.ID, 0)
			if err != nil {
				fmt.Printf("%d 加入会话失败", targetUser.ID)
				c.Send <- []byte("发送失败")
				continue
			}
		}

		//消息入库
		sendMsg := model.Message{ConversationID: convID, SenderID: c.UserID, MsgType: 1, Content: m.Content, ClientMsgID: m.ClientMsgID}
		err = repository.CreateMessage(&sendMsg)
		if err != nil {
			fmt.Println("消息入库失败")
			c.Send <- []byte("发送失败")
			continue
		}
		//更新会话表
		err = repository.UpdateConversation(convID, &sendMsg)
		if err != nil {
			fmt.Println("会话表更新失败")
			c.Send <- []byte("发送失败")
			continue
		}

		// 找目标用户有没有在线
		c.Manager.mu.RLock()
		targetClient, ok := c.Manager.clients[m.ToUserID]
		c.Manager.mu.RUnlock()
		if !ok {
			fmt.Println("用户不在线")
			c.Send <- []byte("用户不在线,发送失败")
			continue
		}

		collectMsg := CollectMessage{SenderID: c.UserID, ConversationID: convID, MsgID: sendMsg.ID, Content: m.Content}
		collectMsgByte, err := json.Marshal(collectMsg)
		if err != nil {
			fmt.Println("消息序列化失败")
			c.Send <- []byte("发送失败")
			continue
		}

		// 转发消息
		select {
		case targetClient.Send <- collectMsgByte:
		default:
			// 对方阻塞了 → 踢掉
			go c.Manager.Unregister(targetClient)
		}

	}

}

// 假设从websocket发过来的数据全是JSON
func (c *Client) DeliverMsg() {
	for msg := range c.Send {
		// 消息反序列化，准备进行最后阅读消息的更新
		collectMsg := CollectMessage{}
		err := json.Unmarshal(msg, &collectMsg)
		if err != nil {
			fmt.Println("消息反序列化失败")
			return
		}
		//把消息发送给用户
		err = c.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			return
		}
		//如果用户成功接受了消息，则更新last_read_msg_id
		err = repository.UpdateLastReadMsgID(collectMsg.ConversationID, c.UserID, collectMsg.MsgID)
		if err != nil {
			if errors.Is(err, errors.New("没有更新任何数据")) {
				fmt.Println("没有可更新消息")
			}
			fmt.Println("消息更新失败")
			return
		}

	}
}

func (c *Client) DeliverUnreadMsg() {
	//先根据用户ID去查看用户的conversation_id 和 last_read_msg_id
	//利用conversation_id将conversation_member和conversation表连接起来，若last_read_msg_id<last_msg_id，则推送消息
	unreadMsgs, err := repository.GetConversationMsgByUserID(c.UserID)
	if err != nil {
		fmt.Println("离线消息推送失败")
		return
	}
	if len(unreadMsgs) == 0 {
		fmt.Println("暂无离线消息")
		c.Send <- []byte("暂无离线消息")
	}
	for _, msg := range unreadMsgs {
		sendMsg := CollectMessage{
			MsgID:          msg.ID,
			Content:        msg.Content,
			ConversationID: msg.ConversationID,
			SenderID:       msg.SenderID,
		}
		collectMsgByte, err := json.Marshal(sendMsg)
		if err != nil {
			fmt.Println("消息序列化失败")
			continue
		}
		c.Send <- collectMsgByte
	}

}
