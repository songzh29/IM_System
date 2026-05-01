package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// 命令行参数
var (
	mode       = flag.String("mode", "cross", "cross or local")
	serverHTTP = flag.String("http", "http://localhost", "HTTP base URL")
	port1      = flag.Int("port1", 8080, "instance 1 port")
	port2      = flag.Int("port2", 8081, "instance 2 port")
	usersFile  = flag.String("users", "users.json", "user list file")
	pairs      = flag.Int("pairs", 100, "number of user pairs (so total users = 2 * pairs)")
	rate       = flag.Float64("rate", 1.0, "messages per second per user")
	duration   = flag.Duration("duration", 60*time.Second, "test duration")
	output     = flag.String("output", "loadtest_result.csv", "output CSV path")
)

type AckMsgPayload struct {
	ServerMsgID uint `json:"server_msg_id"`
	ConvID      uint `json:"conv_id"`
}

type UsersFile struct {
	Password string   `json:"password"`
	Users    []string `json:"users"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Msg    string `json:"msg"`
	Token  string `json:"token"`
	UserID uint   `json:"user_id"` // ← 加
}

// 协议结构(必须和 ws/client.go 对齐)
type ClientMessage struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type ChatMsgPayload struct {
	ClientMsgID string `json:"client_msg_id"`
	ToUserID    uint   `json:"to_user_id"`
	Content     string `json:"content"`
}

type CollectMessage struct {
	SenderID       uint   `json:"sender_id"`
	ConversationID uint   `json:"conversation_id"`
	Content        string `json:"content"`
	MsgID          uint   `json:"msg_id"`
	ClientMsgID    string `json:"client_msg_id"` // ← 为了压测加
}

// 测试虚拟用户
type VirtualUser struct {
	Username string
	UserID   uint
	Token    string
	Port     int
	Conn     *websocket.Conn
	WriteMu  sync.Mutex // ← 加
}

// 全局统计
type Stats struct {
	Sent          atomic.Int64
	Received      atomic.Int64
	SendErrors    atomic.Int64
	ConnectErrors atomic.Int64
	LatenciesMu   sync.Mutex
	Latencies     []time.Duration
	StartTime     time.Time
}

var stats = &Stats{Latencies: make([]time.Duration, 0, 100000)}

// 用 client_msg_id 做发送时间戳关联
type pendingMsg struct {
	sentAt time.Time
}

var pending sync.Map // map[string]pendingMsg

func main() {
	flag.Parse()

	// 1. 读用户列表
	data, err := os.ReadFile(*usersFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读用户列表失败: %v\n", err)
		os.Exit(1)
	}
	var uf UsersFile
	if err := json.Unmarshal(data, &uf); err != nil {
		fmt.Fprintf(os.Stderr, "解析用户列表失败: %v\n", err)
		os.Exit(1)
	}

	if len(uf.Users) < *pairs*2 {
		fmt.Fprintf(os.Stderr, "用户不够:需要 %d,只有 %d\n", *pairs*2, len(uf.Users))
		os.Exit(1)
	}

	// 2. 串行登录,拿 JWT(避免并发打挂 bcrypt)
	fmt.Printf("正在登录 %d 个用户...\n", *pairs*2)
	users := make([]*VirtualUser, *pairs*2)
	for i := 0; i < *pairs*2; i++ {
		username := uf.Users[i]
		port := *port1
		if *mode == "cross" && i%2 == 1 {
			port = *port2
		}
		token, userID, err := login(*serverHTTP, *port1, username, uf.Password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "登录 %s 失败: %v\n", username, err)
			continue
		}
		users[i] = &VirtualUser{
			Username: username,
			UserID:   userID,
			Token:    token,
			Port:     port,
		}
	}
	fmt.Printf("登录完成\n")

	// 3. 并行连接 WebSocket
	fmt.Printf("建立 WebSocket 连接...\n")
	var wg sync.WaitGroup
	for _, u := range users {
		if u == nil {
			continue
		}
		wg.Add(1)
		go func(u *VirtualUser) {
			defer wg.Done()
			conn, err := connectWS(*serverHTTP, u.Port, u.Token)
			if err != nil {
				stats.ConnectErrors.Add(1)
				return
			}
			u.Conn = conn
			go receiveLoop(u)
		}(u)
	}
	wg.Wait()

	connected := 0
	for _, u := range users {
		if u != nil && u.Conn != nil {
			connected++
		}
	}
	fmt.Printf("WebSocket 连接完成:%d/%d\n", connected, len(users))
	if connected < *pairs*2 {
		fmt.Println("⚠️ 部分连接失败,实际压测对数会减少")
	}

	// 4. 等所有连接稳定 + Prometheus 抓一次指标
	time.Sleep(2 * time.Second)

	// 5. 启动压测
	fmt.Printf("\n开始压测:%d 对用户,每用户 %.1f 条/秒,持续 %v\n", *pairs, *rate, *duration)
	stats.StartTime = time.Now()

	ctx := make(chan struct{})
	for i := 0; i < *pairs; i++ {
		userA := users[i*2]
		userB := users[i*2+1]
		if userA == nil || userA.Conn == nil || userB == nil || userB.Conn == nil {
			continue
		}
		go sendLoop(userA, userB.UserID, *rate, ctx)
		go sendLoop(userB, userA.UserID, *rate, ctx)
	}

	// 进度打印
	go progressReport(ctx)

	// 6. 等待结束
	time.Sleep(*duration)
	close(ctx)
	time.Sleep(2 * time.Second) // 等最后的消息收尾

	// 7. 关连接
	for _, u := range users {
		if u != nil && u.Conn != nil {
			u.Conn.Close()
		}
	}

	// 8. 输出结果
	printResults()
	exportCSV(*output)
}

func login(httpURL string, port int, username, password string) (string, uint, error) {
	loginURL := fmt.Sprintf("%s:%d/public/login", httpURL, port)
	body, _ := json.Marshal(LoginRequest{Username: username, Password: password})
	resp, err := http.Post(loginURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("login failed status=%d body=%s", resp.StatusCode, string(respBody))
	}
	var lr LoginResponse
	json.NewDecoder(resp.Body).Decode(&lr)
	// JWT 解析 user_id 留给 receiveLoop 第一次收消息时,或调用其他 API 拿
	// 这里简化:不从 JWT 解 user_id,留 0,后面如果需要再加
	return lr.Token, lr.UserID, nil
}

func connectWS(httpURL string, port int, token string) (*websocket.Conn, error) {
	wsURL := fmt.Sprintf("%s:%d/private/ws", strings.Replace(httpURL, "http", "ws", 1), port)
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func sendLoop(u *VirtualUser, targetUserID uint, ratePerSec float64, ctx <-chan struct{}) {
	interval := time.Duration(float64(time.Second) / ratePerSec)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	counter := 0
	for {
		select {
		case <-ctx:
			return
		case <-ticker.C:
			counter++
			clientMsgID := fmt.Sprintf("%s-%d-%d", u.Username, time.Now().UnixNano(), counter)

			payload := ChatMsgPayload{
				ClientMsgID: clientMsgID,
				ToUserID:    targetUserID,
				Content:     fmt.Sprintf("msg from %s seq=%d", u.Username, counter),
			}
			payloadBytes, _ := json.Marshal(payload)
			envelope := ClientMessage{
				Action: "chat",
				Data:   payloadBytes,
			}
			msgBytes, _ := json.Marshal(envelope)

			pending.Store(clientMsgID, pendingMsg{sentAt: time.Now()})

			u.WriteMu.Lock()
			err := u.Conn.WriteMessage(websocket.TextMessage, msgBytes)
			u.WriteMu.Unlock()
			if err != nil {
				stats.SendErrors.Add(1)
				pending.Delete(clientMsgID)
				return
			}
			stats.Sent.Add(1)
		}
	}
}

func receiveLoop(u *VirtualUser) {
	for {
		_, msg, err := u.Conn.ReadMessage()
		if err != nil {
			return
		}
		stats.Received.Add(1)

		var cm CollectMessage
		if err := json.Unmarshal(msg, &cm); err != nil {
			continue
		}

		// 关联发送时间戳
		if v, ok := pending.LoadAndDelete(cm.ClientMsgID); ok {
			sentAt := v.(pendingMsg).sentAt
			latency := time.Since(sentAt)
			stats.LatenciesMu.Lock()
			stats.Latencies = append(stats.Latencies, latency)
			stats.LatenciesMu.Unlock()
		}

		// 立即回 ACK,避免下次压测时重推
		ackPayload := AckMsgPayload{ServerMsgID: cm.MsgID, ConvID: cm.ConversationID}
		ackBytes, _ := json.Marshal(ackPayload)
		envelope := ClientMessage{Action: "ack", Data: ackBytes}
		envBytes, _ := json.Marshal(envelope)
		// 发 ACK 不加锁的话可能并发写,gorilla 不允许
		// 但你用的是单 goroutine receiveLoop,这里 WriteMessage 和 sendLoop 在不同 goroutine
		// → 需要互斥
		// 简化:整个客户端连接一把锁
		u.WriteMu.Lock()
		u.Conn.WriteMessage(websocket.TextMessage, envBytes)
		u.WriteMu.Unlock()
	}
}

func progressReport(ctx <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx:
			return
		case <-ticker.C:
			elapsed := time.Since(stats.StartTime).Seconds()
			sent := stats.Sent.Load()
			recv := stats.Received.Load()
			fmt.Printf("[+%.0fs] 已发送 %d (%.0f/s),已接收 %d (%.0f/s),发送错误 %d\n",
				elapsed, sent, float64(sent)/elapsed, recv, float64(recv)/elapsed, stats.SendErrors.Load())
		}
	}
}

func printResults() {
	elapsed := time.Since(stats.StartTime).Seconds()
	sent := stats.Sent.Load()
	recv := stats.Received.Load()

	fmt.Println("\n========== 压测结果 ==========")
	fmt.Printf("持续时间:    %.1f 秒\n", elapsed)
	fmt.Printf("总发送:      %d (%.0f msg/s)\n", sent, float64(sent)/elapsed)
	fmt.Printf("总接收:      %d (%.0f msg/s)\n", recv, float64(recv)/elapsed)
	fmt.Printf("丢包率:      %.2f%%\n", float64(sent-recv)/float64(sent)*100)
	fmt.Printf("发送错误:    %d\n", stats.SendErrors.Load())
	fmt.Printf("连接错误:    %d\n", stats.ConnectErrors.Load())
	fmt.Println("================================")
	fmt.Println("\n服务端分段延迟数据请查看日志: logs/im_system_8080.log")
}

func exportCSV(path string) {
	// 留给后续:压测开始时间 / 结束时间 / 总数 等
	// 现在只是留个占位
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "metric,value\n")
	fmt.Fprintf(f, "duration_seconds,%.1f\n", time.Since(stats.StartTime).Seconds())
	fmt.Fprintf(f, "total_sent,%d\n", stats.Sent.Load())
	fmt.Fprintf(f, "total_received,%d\n", stats.Received.Load())
	fmt.Fprintf(f, "send_errors,%d\n", stats.SendErrors.Load())
	fmt.Fprintf(f, "connect_errors,%d\n", stats.ConnectErrors.Load())
	fmt.Println("\n结果已导出到", path)

	stats.LatenciesMu.Lock()
	sort.Slice(stats.Latencies, func(i, j int) bool {
		return stats.Latencies[i] < stats.Latencies[j]
	})
	n := len(stats.Latencies)
	if n > 0 {
		p50 := stats.Latencies[n*50/100]
		p95 := stats.Latencies[n*95/100]
		p99 := stats.Latencies[n*99/100]
		fmt.Fprintf(f, "p50_ms,%.2f\n", float64(p50.Microseconds())/1000)
		fmt.Fprintf(f, "p95_ms,%.2f\n", float64(p95.Microseconds())/1000)
		fmt.Fprintf(f, "p99_ms,%.2f\n", float64(p99.Microseconds())/1000)
	}
	stats.LatenciesMu.Unlock()
}
