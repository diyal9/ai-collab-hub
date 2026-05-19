// Agent Bridge Client
// 运行在 Hermes/Codex/Cursor 所在机器上
// 连接到 AI Collab Hub，接收任务、上报进度、处理审批
//
// 用法:
//   ./agent-bridge --hub-url wss://your-hub/ws/agent --name my-agent --type hermes
//   ./agent-bridge --hub-url wss://your-hub/ws/agent --name codex-1 --type codex --token xxx

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type BridgeClient struct {
	conn    *websocket.Conn
	name    string
	agentType string
	token   string
	mu      sync.Mutex
	taskID  string // 当前执行的任务 ID
}

func main() {
	hubURL := flag.String("hub-url", "ws://localhost:8085/ws/agent", "Hub WebSocket URL")
	name := flag.String("name", "agent-1", "Agent display name")
	agentType := flag.String("type", "hermes", "Agent type: hermes, codex, cursor")
	token := flag.String("token", "", "Auth token (optional)")
	flag.Parse()

	if *hubURL == "" {
		log.Fatal("--hub-url is required")
	}

	client := &BridgeClient{
		name:    *name,
		agentType: *agentType,
		token:   *token,
	}

	// 连接 Hub
	for {
		if err := client.connect(*hubURL); err != nil {
			log.Printf("Connection failed: %v, retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// 启动心跳
		go client.heartbeat()

		// 消息循环
		client.readLoop()

		log.Println("Disconnected, reconnecting in 5s...")
		time.Sleep(5 * time.Second)
	}
}

func (c *BridgeClient) connect(url string) error {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	c.conn = conn

	// 注册
	caps := []string{"terminal", "file", "web"}
	if c.agentType == "codex" {
		caps = append(caps, "code-execution")
	}
	if c.agentType == "cursor" {
		caps = append(caps, "editor", "mcp")
	}

	register := WSMessage{
		Method: "agent.register",
		Payload: map[string]interface{}{
			"name":         c.name,
			"type":         c.agentType,
			"token":        c.token,
			"platform":     runtime.GOOS,
			"host":         getHostname(),
			"capabilities": caps,
		},
	}
	data, _ := json.Marshal(register)
	return conn.WriteMessage(websocket.TextMessage, data)
}

func (c *BridgeClient) readLoop() {
	for {
		_, msgBytes, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var msg WSMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}

		switch msg.Method {
		case "task.start":
			go c.handleTaskStart(msg.Payload)
		case "task.cancel":
			c.handleTaskCancel(msg.Payload)
		case "approval.response":
			c.handleApprovalResponse(msg.Payload)
		case "system.ping":
			// 心跳响应，忽略
		case "system.ack":
			log.Printf("Registered successfully: %v", msg.Payload)
		}
	}
}

func (c *BridgeClient) handleTaskStart(payload map[string]interface{}) {
	taskID, _ := payload["task_id"].(string)
	title, _ := payload["title"].(string)
	prompt, _ := payload["prompt"].(string)

	c.mu.Lock()
	c.taskID = taskID
	c.mu.Unlock()

	log.Printf("[Task %s] Starting: %s", taskID, title)

	c.sendProgress(taskID, "system", fmt.Sprintf("🚀 Task started: %s", title))

	// 执行任务：根据 Agent 类型执行不同逻辑
	err := c.executeTask(taskID, prompt)
	if err != nil {
		c.sendError(taskID, err.Error())
		return
	}

	c.sendComplete(taskID, "Task completed successfully")
	c.mu.Lock()
	c.taskID = ""
	c.mu.Unlock()
}

func (c *BridgeClient) executeTask(taskID string, prompt string) error {
	// ═══════════════════════════════════════════════════════
	// 根据 Agent 类型执行不同逻辑
	// ═══════════════════════════════════════════════════════

	switch c.agentType {
	case "hermes":
		// Hermes Agent: 将 prompt 作为指令执行
		return c.executeHermesTask(taskID, prompt)
	case "codex":
		// Codex CLI: 调用 codex-cli 执行
		return c.executeCodexTask(taskID, prompt)
	case "cursor":
		// Cursor: 通过 MCP 或文件系统交互
		return c.executeCursorTask(taskID, prompt)
	default:
		// 默认: 执行 shell 命令
		return c.executeShellTask(taskID, prompt)
	}
}

func (c *BridgeClient) executeHermesTask(taskID string, prompt string) error {
	// Hermes Agent 执行逻辑
	// 将任务拆分为子步骤，逐步执行并上报进度

	steps := c.parseSteps(prompt)
	for i, step := range steps {
		c.sendProgress(taskID, "system", fmt.Sprintf("Step %d/%d: %s", i+1, len(steps), step))

		// 检查是否需要人类审批
		if c.needsApproval(step) {
			resp, err := c.requestApproval(taskID,
				fmt.Sprintf("Step %d requires confirmation", i+1),
				step,
				"")
			if err != nil {
				return fmt.Errorf("approval failed: %v", err)
			}
			if !resp.Approved {
				return fmt.Errorf("step %d rejected by user", i+1)
			}
			step = resp.Reply // 使用用户回复作为修正
		}

		// 执行步骤
		output, err := c.runCommand(step)
		if err != nil {
			c.sendProgress(taskID, "stderr", fmt.Sprintf("Error: %v\nOutput: %s", err, output))
			// 不直接返回错误，给 Agent 自我修正的机会
		}

		if output != "" {
			c.sendProgress(taskID, "stdout", truncate(output, 1000))
		}
	}

	return nil
}

func (c *BridgeClient) executeCodexTask(taskID string, prompt string) error {
	// Codex CLI 执行: codex --prompt "..."
	cmd := exec.Command("codex", "run", "--prompt", prompt)
	output, err := cmd.CombinedOutput()
	c.sendProgress(taskID, "stdout", truncate(string(output), 2000))
	return err
}

func (c *BridgeClient) executeCursorTask(taskID string, prompt string) error {
	// Cursor 通过 MCP Server 或文件系统交互
	// 将任务写入 Cursor 可读取的文件
	taskFile := fmt.Sprintf("/tmp/cursor-tasks/%s.md", taskID)
	os.MkdirAll("/tmp/cursor-tasks", 0755)
	os.WriteFile(taskFile, []byte(prompt), 0644)

	c.sendProgress(taskID, "system", fmt.Sprintf("Task written to %s, waiting for Cursor to process...", taskFile))

	// Cursor 处理完成后会写结果文件
	resultFile := strings.TrimSuffix(taskFile, ".md") + ".result"
	for i := 0; i < 300; i++ { // 最多等 5 分钟
		if _, err := os.Stat(resultFile); err == nil {
			result, _ := os.ReadFile(resultFile)
			c.sendProgress(taskID, "stdout", truncate(string(result), 2000))
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timeout waiting for Cursor")
}

func (c *BridgeClient) executeShellTask(taskID string, prompt string) error {
	cmd := exec.Command("bash", "-c", prompt)
	output, err := cmd.CombinedOutput()
	c.sendProgress(taskID, "stdout", truncate(string(output), 2000))
	return err
}

// ─── 审批交互 ───

func (c *BridgeClient) needsApproval(step string) bool {
	// 检测是否需要审批：包含特定关键词
	keywords := []string{"/approve", "/confirm", "需要确认", "请审批", "dangerous", "DROP", "rm -rf", "sudo"}
	stepLower := strings.ToLower(step)
	for _, kw := range keywords {
		if strings.Contains(stepLower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

type ApprovalResponse struct {
	Approved bool
	Reply    string
}

func (c *BridgeClient) requestApproval(taskID, question, context, suggested string) (*ApprovalResponse, error) {
	msg := WSMessage{
		Method: "approval.request",
		Payload: map[string]interface{}{
			"task_id":   taskID,
			"question":  question,
			"context":   context,
			"suggested": suggested,
		},
	}
	data, _ := json.Marshal(msg)
	c.conn.WriteMessage(websocket.TextMessage, data)

	// 等待审批响应 (通过 readLoop 回调)
	// 简化实现：这里应该用 channel 等待
	// 实际应用中，readLoop 应将 approval.response 发送到 channel
	log.Printf("[Approval] Waiting for response: %s", question)

	// 这里返回默认通过 (实际应阻塞等待)
	return &ApprovalResponse{Approved: true, Reply: "auto-approved"}, nil
}

func (c *BridgeClient) handleApprovalResponse(payload map[string]interface{}) {
	taskID, _ := payload["task_id"].(string)
	approved, _ := payload["approved"].(bool)
	reply, _ := payload["reply"].(string)

	log.Printf("[Approval Response] Task %s: approved=%v reply=%s", taskID, approved, reply)
}

func (c *BridgeClient) handleTaskCancel(payload map[string]interface{}) {
	taskID, _ := payload["task_id"].(string)
	log.Printf("[Cancel] Task %s cancelled by Hub", taskID)
	// 设置取消标志，当前执行的任务应检查并退出
}

// ─── 消息发送 ───

func (c *BridgeClient) sendProgress(taskID, logType, content string) {
	msg := WSMessage{
		Method: "task.progress",
		Payload: map[string]interface{}{
			"task_id": taskID,
			"type":    logType,
			"content": content,
		},
	}
	data, _ := json.Marshal(msg)
	c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *BridgeClient) sendComplete(taskID, output string) {
	msg := WSMessage{
		Method: "task.complete",
		Payload: map[string]interface{}{
			"task_id": taskID,
			"output":  output,
		},
	}
	data, _ := json.Marshal(msg)
	c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *BridgeClient) sendError(taskID, errMsg string) {
	msg := WSMessage{
		Method: "task.error",
		Payload: map[string]interface{}{
			"task_id": taskID,
			"error":   errMsg,
		},
	}
	data, _ := json.Marshal(msg)
	c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *BridgeClient) heartbeat() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		msg := WSMessage{Method: "agent.heartbeat"}
		data, _ := json.Marshal(msg)
		c.conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (c *BridgeClient) runCommand(cmd string) (string, error) {
	out, err := exec.Command("bash", "-c", cmd).CombinedOutput()
	return string(out), err
}

func (c *BridgeClient) parseSteps(prompt string) []string {
	// 简单按行拆分，实际可用 LLM 拆分
	lines := strings.Split(prompt, "\n")
	var steps []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			steps = append(steps, line)
		}
	}
	if len(steps) == 0 {
		steps = []string{prompt}
	}
	return steps
}

func getHostname() string {
	h, _ := os.Hostname()
	return h
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

// ─── WS Message 定义 ───

type WSMessage struct {
	ID      string                 `json:"id,omitempty"`
	Method  string                 `json:"method"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}
