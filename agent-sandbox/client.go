package main

import (
	"encoding/json"
	"log"
	"os/exec"
	"time"

	"github.com/gorilla/websocket"
)

type SandboxClient struct {
	URL        string
	SandboxID  string
	Conn       *websocket.Conn
	Executors  map[string]Executor
	Capabilities []string
	retryCount int
	maxRetries int
}

func NewClient(url, id string) *SandboxClient {
	return &SandboxClient{
		URL:        url,
		SandboxID:  id,
		Executors:  make(map[string]Executor),
		maxRetries: 10,
	}
}

func (c *SandboxClient) Start() {
	// 注册执行器
	c.Executors["shell"] = &ShellExecutor{}
	c.Executors["cursor"] = &CursorExecutor{}
	c.Executors["aider"] = &AiderExecutor{}

	// 声明支持的能力 (排除未安装的)
	caps := []string{"shell"}
	// 检查 cursor 是否可用
	if _, err := exec.LookPath("cursor"); err == nil {
		caps = append(caps, "cursor")
	}
	// 检查 aider 是否可用
	if _, err := exec.LookPath("aider"); err == nil {
		caps = append(caps, "aider")
	}
	c.Capabilities = caps

	for {
		if err := c.connect(); err != nil {
			log.Printf("[Client] Connect failed: %v. Retrying...", err)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Printf("[Client] Connected to Bridge: %s", c.URL)
		c.retryCount = 0
		c.run()
	}
}

func (c *SandboxClient) connect() error {
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial(c.URL, nil)
	if err != nil {
		return err
	}
	c.Conn = conn

	// 发送注册信息
	regPayload, _ := json.Marshal(RegisterPayload{
		SandboxID:    c.SandboxID,
		Capabilities: c.Capabilities,
	})
	regMsg := Message{
		Type:    MsgRegister,
		Payload: regPayload,
	}
	return c.Send(regMsg)
}

func (c *SandboxClient) run() {
	go c.startHeartbeat()

	for {
		_, msgBytes, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("[Client] Connection lost: %v", err)
			return
		}
		c.handleMessage(msgBytes)
	}
}

func (c *SandboxClient) handleMessage(data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	switch msg.Type {
	case MsgTaskAssign:
		var payload TaskPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("[Client] Failed to unmarshal payload: %v", err)
			return
		}
		// 查找对应执行器
		executor, exists := c.Executors[payload.Executor]
		if !exists {
			c.SendError(msg.TaskID, "Unknown executor: "+payload.Executor)
			return
		}
		go func() {
			if err := executor.Execute(msg.TaskID, payload, c); err != nil {
				log.Printf("[Client] Executor %s failed for task %s: %v", payload.Executor, msg.TaskID, err)
			}
		}()
	}
}

func (c *SandboxClient) Send(msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.Conn.WriteMessage(websocket.TextMessage, data)
}

func (c *SandboxClient) SendStatus(taskID, status string) error {
	payload, _ := json.Marshal(StatusPayload{Status: status})
	return c.Send(Message{Type: MsgTaskStatus, TaskID: taskID, Payload: payload})
}

func (c *SandboxClient) SendLog(taskID, data string) error {
	payload, _ := json.Marshal(LogPayload{Stream: "stdout", Data: data})
	return c.Send(Message{Type: MsgTaskLog, TaskID: taskID, Payload: payload})
}

func (c *SandboxClient) SendError(taskID, errMsg string) error {
	return c.Send(Message{Type: MsgTaskStatus, TaskID: taskID, Error: errMsg, Payload: json.RawMessage(`{"status":"failed"}`)})
}

func (c *SandboxClient) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := c.Send(Message{Type: MsgHeartbeat}); err != nil {
			return
		}
	}
}
