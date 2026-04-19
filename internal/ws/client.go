package ws

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/songzh29/IM_System/internal/model"
	"github.com/songzh29/IM_System/internal/mq"
	"github.com/songzh29/IM_System/internal/repository"
	"go.uber.org/zap"
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

// 负责监听客户端发送来的消息
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
			zap.L().Error("json解析失败:", zap.Error(err))
			continue
		}
		//查看用户是否存在
		targetUser, err := repository.GetUserByUserID(m.ToUserID)

		if err == nil && targetUser == nil {
			zap.L().Warn("目标用户不存在",
				zap.Uint("sender_id", c.UserID),
				zap.Uint("to_user_id", m.ToUserID),
				zap.Error(err),
			)
			continue
		} else if err != nil {
			zap.L().Error("查找目标用户出错",
				zap.Uint("sender_id", c.UserID),
				zap.Uint("to_user_id", m.ToUserID),
				zap.Error(err),
			)
			continue
		}

		//查看用户之间有没有会话
		convID, err := repository.CheckConversationExist(c.UserID, targetUser.ID, 1)
		if err == nil && convID == 0 {
			zap.L().Warn("两人之间无对话",
				zap.Uint("sender_id", c.UserID),
				zap.Uint("to_user_id", m.ToUserID),
			)
		} else if err != nil {
			zap.L().Error("查找对话失败",
				zap.Uint("sender_id", c.UserID),
				zap.Uint("to_user_id", m.ToUserID),
				zap.Error(err),
			)
		}

		//没有对话则创建对话
		if convID == 0 {
			zap.L().Info("正在创建对话")
			convName := fmt.Sprintf("%d & %d", c.UserID, targetUser.ID)
			conv := model.Conversation{Type: 1, Name: convName, CreatorID: c.UserID}
			//存入MySQL
			err := repository.CreateConversation(&conv)
			if err != nil {
				zap.L().Error("创建对话失败", zap.Error(err))
				continue
			}
			zap.L().Info("对话创建成功")
			convID = conv.ID

			//存入redis
			err = repository.CacheConversationID(c.UserID, targetUser.ID, 1, convID)
			if err != nil {
				zap.L().Error("Redis写入对话失败", zap.Error(err))
			}

			//将两人加入会话成员表（拉入会话）
			err = repository.AddMember(conv.ID, c.UserID, 0)
			if err != nil {
				zap.L().Error("加入会话失败", zap.Uint("user_id", c.UserID), zap.Error(err))
				continue
			}
			err = repository.AddMember(conv.ID, targetUser.ID, 0)
			if err != nil {
				zap.L().Error("加入会话失败", zap.Uint("user_id", targetUser.ID), zap.Error(err))
				continue
			}
		}

		//消息入库
		sendMsg := model.Message{ConversationID: convID, SenderID: c.UserID, MsgType: 1, Content: m.Content, ClientMsgID: m.ClientMsgID}
		err = repository.CreateMessage(&sendMsg)
		if err != nil {
			zap.L().Error("消息入库失败", zap.Error(err))
			continue
		}

		// //更新会话表
		// err = repository.UpdateConversation(convID, &sendMsg)
		// if err != nil {
		// 	zap.L().Error("会话表更新失败", zap.Error(err))
		// 	continue
		// }
		// MQ生产端推送消息
		mqMsg := &model.MqMessage{ConvID: convID, SendMsg: sendMsg}
		err = mq.PublishMessageSent(mqMsg)
		if err != nil {
			zap.L().Error("MQ发布消息会话表更新消息失败", zap.Error(err))
			continue
		}
		// 开协程消费消息
		go func() {
			err := mq.StartConversationUpdateConsumer()
			if err != nil {
				zap.L().Error("会话表更新失败", zap.Error(err))
			}
		}()

		//把已读消息更新到自己发送的消息，自己发送的肯定是已读
		err = repository.UpdateLastReadMsgID(convID, c.UserID, sendMsg.ID)
		if err != nil {
			zap.L().Error("尝试将已读消息更新为自己刚刚发送的消息ID,但是更新失败", zap.Error(err))
			continue
		}

		// 找目标用户有没有在线
		c.Manager.mu.RLock()
		targetClient, ok := c.Manager.clients[m.ToUserID]
		c.Manager.mu.RUnlock()
		if !ok {
			zap.L().Warn("用户不在线")
			continue
		}

		collectMsg := CollectMessage{SenderID: c.UserID, ConversationID: convID, MsgID: sendMsg.ID, Content: m.Content}
		collectMsgByte, err := json.Marshal(collectMsg)
		if err != nil {
			zap.L().Error("消息序列化失败", zap.Error(err))
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
			zap.L().Error("消息反序列化失败", zap.Error(err))
			continue
		}
		//把消息发送给用户
		err = c.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			zap.L().Error("消息发送失败",
				zap.Uint("target_id", c.UserID),
				zap.Error(err))
			return
		}
		//如果用户成功接受了消息，则更新last_read_msg_id
		err = repository.UpdateLastReadMsgID(collectMsg.ConversationID, c.UserID, collectMsg.MsgID)
		if err != nil {
			if errors.Is(err, repository.ErrUpdateNoneLastReadMsg) {
				zap.L().Warn("没有可更新消息")
			}
			zap.L().Error("消息更新失败", zap.Uint("target_id", c.UserID), zap.Error(err))
			continue
		}

	}
}

func (c *Client) DeliverUnreadMsg() {
	//先根据用户ID去查看用户的conversation_id 和 last_read_msg_id
	//利用conversation_id将conversation_member和conversation表连接起来，若last_read_msg_id<last_msg_id，则推送消息
	unreadMsgs, err := repository.GetUnreadConversationMsgByUserID(c.UserID)
	if err != nil {
		zap.L().Error("离线消息查找失败", zap.Uint("user_id", c.UserID), zap.Error(err))
		return
	}
	if len(unreadMsgs) == 0 {
		zap.L().Warn("暂无离线消息")
	}
	//创建一个map，key和value分别对应conv_id和max_msg_id
	msgMap := make(map[uint]uint)
	for _, msg := range unreadMsgs {
		sendMsg := CollectMessage{
			MsgID:          msg.ID,
			Content:        msg.Content,
			ConversationID: msg.ConversationID,
			SenderID:       msg.SenderID,
		}
		collectMsgByte, err := json.Marshal(sendMsg)
		if err != nil {
			zap.L().Error("消息序列化失败",
				zap.Uint("msg_id", sendMsg.MsgID),
				zap.Uint("user_id", c.UserID),
				zap.Uint("sender_id", sendMsg.SenderID),
				zap.Error(err),
			)
			continue
		}
		err = c.Conn.WriteMessage(websocket.TextMessage, collectMsgByte)
		if err != nil {
			zap.L().Error("离线消息推送失败",
				zap.Uint("msg_id", sendMsg.MsgID),
				zap.Uint("user_id", c.UserID),
				zap.Uint("sender_id", sendMsg.SenderID),
				zap.Error(err),
			)
			return
		}
		if msgMap[msg.ConversationID] < msg.ID {
			msgMap[msg.ConversationID] = msg.ID
		}

	}
	for convID, lastmsgID := range msgMap {
		err := repository.UpdateLastReadMsgID(convID, c.UserID, lastmsgID)
		if err != nil {
			zap.L().Error("数据库更新last_read_msg_id失败", zap.Error(err))
			return
		}
	}

}
