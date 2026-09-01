package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// HubClient 负责连接主 Hub 平台
var HubClient = &HubConnection{}

type HubConnection struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (h *HubConnection) Connect(url string) {
	for {
		var dialer websocket.Dialer
		c, _, err := dialer.Dial(url, nil)
		if err != nil {
			log.Printf("[HubClient] Failed to connect to Hub (%s): %v. Retrying in 5s...", url, err)
			time.Sleep(5 * time.Second)
			continue
		}
		h.mu.Lock()
		h.conn = c
		h.mu.Unlock()
		log.Printf("[HubClient] Connected to Hub: %s", url)

		// 发送注册信息
		h.sendRegister()

		// 启动读循环
		go h.readLoop()

		// 启动心跳
		go h.heartbeat()

		// 阻塞等待断开
		select {}
	}
}

func (h *HubConnection) sendRegister() {
	cfg := GetConfig()
	msg := map[string]interface{}{
		"method": "agent.register",
		"payload": map[string]string{
			"name": cfg.Hub.AgentID,
			"type": "bridge",
			"token": cfg.Hub.APIToken,
			"host": cfg.Server.Host,
		},
	}
	data, _ := json.Marshal(msg)
	h.SendToHub(data)
}

func (h *HubConnection) readLoop() {
	for {
		_, msg, err := h.conn.ReadMessage()
		if err != nil {
			log.Printf("[HubClient] Read error: %v", err)
			h.conn.Close()
			return
		}
			// 收到 Sandbox 消息 -> 转发给 Hub
			HubClient.SendToHub(msg)
	}
}

func (h *HubConnection) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		h.SendToHub([]byte(`{"method":"agent.heartbeat"}`))
	}
}

func (h *HubConnection) SendToHub(data []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conn == nil {
		return nil
	}
	return h.conn.WriteMessage(websocket.TextMessage, data)
}
