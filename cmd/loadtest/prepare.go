//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	serverURL   = "http://localhost:8080"
	userCount   = 1000
	password    = "loadtest123" // 所有用户共用
	concurrency = 50            // 并发注册数,防止打挂服务
)

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

func main() {
	usernames := make([]string, 0, userCount)
	var mu sync.Mutex

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	startTime := time.Now()
	successCount := 0

	for i := 0; i < userCount; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			username := fmt.Sprintf("loadtest_user_%05d", idx)
			req := RegisterRequest{
				Username: username,
				Password: password,
				Nickname: fmt.Sprintf("LT%d", idx),
			}

			body, _ := json.Marshal(req)
			resp, err := http.Post(serverURL+"/public/register", "application/json", bytes.NewReader(body))
			if err != nil {
				fmt.Printf("[%d] register failed: %v\n", idx, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == 200 {
				mu.Lock()
				usernames = append(usernames, username)
				successCount++
				mu.Unlock()
			} else {
				// 用户已存在也算成功(幂等)
				mu.Lock()
				usernames = append(usernames, username)
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// 写入文件
	output := struct {
		Password string   `json:"password"`
		Users    []string `json:"users"`
	}{
		Password: password,
		Users:    usernames,
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	os.WriteFile("users.json", data, 0644)

	elapsed := time.Since(startTime)
	fmt.Printf("\n注册完成:%d/%d 成功,耗时 %v\n", successCount, userCount, elapsed)
	fmt.Println("用户列表已保存到 users.json")
}
