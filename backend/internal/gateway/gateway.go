package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"ai-collab-hub/internal/db"
	"ai-collab-hub/internal/model"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// ─── 消息协议 (JSON-RPC 2.0 style) ───

type WSMessage struct {
	ID      string                 `json:"id,omitempty"`      // 请求 ID
	Method  string                 `json:"method"`            // 消息类型
	Payload map[string]interface{} `json:"payload,omitempty"` // 数据
}

// Agent → Hub methods:
//   agent.register, agent.heartbeat, task.progress, task.complete, task.error, approval.request
// Hub → Agent methods:
//   task.start, task.cancel, approval.response, system.ping

// ─── Agent Gateway ───

type AgentGateway struct {
	mu       sync.RWMutex
	agents   map[uint]*AgentClient  // agent ID → client
	tasks    map[string]uint        // task ID → agent ID
}

type AgentClient struct {
	ID       uint
	Name     string
	Type     string
	Conn     *websocket.Conn
	Send     chan []byte
	mu       sync.Mutex
}

func NewAgentGateway() *AgentGateway {
	gw := &AgentGateway{
		agents: make(map[uint]*AgentClient),
		tasks:  make(map[string]uint),
	}
	// 启动心跳检查
	go gw.heartbeatLoop()
	// 启动任务超时检查
	go gw.timeoutCheckerLoop()
	return gw
}

var Gateway *AgentGateway

func InitGateway() {
	Gateway = NewAgentGateway()
}

// ─── WebSocket Handler ───

func (gw *AgentGateway) HandleWS(conn *websocket.Conn) {
	client := &AgentClient{
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	// 启动读写 goroutine
	go client.writePump()
	go client.readPump(gw)
}

// ─── Agent → Hub 消息处理 ───

func (client *AgentClient) readPump(gw *AgentGateway) {
	defer func() {
		gw.Unregister(client)
		client.Conn.Close()
	}()

	for {
		_, msgBytes, err := client.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}

		gw.handleMessage(client, &msg)
	}
}

func (gw *AgentGateway) handleMessage(client *AgentClient, msg *WSMessage) {
	switch msg.Method {
	case "agent.register":
		gw.handleRegister(client, msg.Payload)
	case "agent.heartbeat":
		gw.handleHeartbeat(client)
	case "task.progress":
		gw.handleProgress(msg.Payload)
	case "task.complete":
		gw.handleComplete(msg.Payload)
	case "task.error":
		gw.handleError(msg.Payload)
	case "approval.request":
		gw.handleApprovalRequest(client, msg.Payload)
	}
}

// ─── 注册 ───

func (gw *AgentGateway) handleRegister(client *AgentClient, payload map[string]interface{}) {
	name, _ := payload["name"].(string)
	agentType, _ := payload["type"].(string)
	token, _ := payload["token"].(string)
	capabilities, _ := json.Marshal(payload["capabilities"])
	platform, _ := payload["platform"].(string)
	host, _ := payload["host"].(string)

	// Token 认证
	var agent model.AgentInstance
	err := db.DB.Where("name = ? AND type = ?", name, agentType).First(&agent).Error
	if err != nil {
		// 首次注册：创建新 Agent 记录
		agent = model.AgentInstance{
			Name:         name,
			Type:         agentType,
			Status:       "online",
			Platform:     platform,
			Host:         host,
			Capabilities: string(capabilities),
			Token:        token,
			LastHeartbeat: time.Now(),
		}
		db.DB.Create(&agent)
	} else {
		// 已有记录：验证 token
		if agent.Token != "" && agent.Token != token {
			log.Printf("[Gateway] Token mismatch for agent %s", name)
			gw.sendTo(client, WSMessage{
				Method: "system.error",
				Payload: map[string]interface{}{"error": "Invalid token"},
			})
			client.Conn.Close()
			return
		}
		// 更新状态和能力（能力可能变化）
		db.DB.Model(&agent).Updates(map[string]interface{}{
			"status":         "online",
			"last_heartbeat": time.Now(),
			"capabilities":   string(capabilities),
		})
	}

	client.ID = agent.ID
	client.Name = agent.Name
	client.Type = agent.Type

	gw.mu.Lock()
	gw.agents[agent.ID] = client
	gw.mu.Unlock()

	log.Printf("[Gateway] Agent registered: %s (%s) id=%d", name, agentType, agent.ID)

	// 回复注册成功
	gw.sendTo(client, WSMessage{
		Method: "system.ack",
		Payload: map[string]interface{}{
			"agent_id": agent.ID,
			"status":   "registered",
		},
	})

	// 如果有排队任务，立即下发
	gw.dispatchQueuedTasks(agent.ID)
	// 同时尝试能力匹配分发其他排队任务
	gw.DispatchByCapabilityMatching()
}

// ─── 心跳 ───

func (gw *AgentGateway) handleHeartbeat(client *AgentClient) {
	if client.ID == 0 {
		return
	}
	db.DB.Model(&model.AgentInstance{}).Where("id = ?", client.ID).
		Update("last_heartbeat", time.Now())
}

// ─── 任务进度 ───

func (gw *AgentGateway) handleProgress(payload map[string]interface{}) {
	taskID, _ := payload["task_id"].(string)
	content, _ := payload["content"].(string)
	logType, _ := payload["type"].(string)
	if logType == "" {
		logType = "stdout"
	}

	// 写入日志
	taskLog := model.AgentTaskLog{
		TaskID:  taskID,
		Type:    logType,
		Content: content,
	}
	db.DB.Create(&taskLog)

	// 更新任务状态为 running
	db.DB.Model(&model.AgentTask{}).Where("id = ?", taskID).
		Update("status", "running")

	// 广播给 SSE 订阅者
	broadcastTaskLog(taskID, logType, content)

	log.Printf("[Task %s] [%s] %s", taskID, logType, truncate(content, 200))
}

// ─── 任务完成 ───

// CompleteTask 公开方法：完成任务（供 Webhook 等外部调用）
func (gw *AgentGateway) CompleteTask(taskID string, output string) {
	now := time.Now()
	db.DB.Model(&model.AgentTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status":       "completed",
		"completed_at": now,
	})

	db.DB.Create(&model.AgentTaskLog{
		TaskID:  taskID,
		Type:    "system",
		Content: fmt.Sprintf("[TASK COMPLETE] %s", truncate(output, 500)),
	})
	broadcastTaskLog(taskID, "system", fmt.Sprintf("[TASK COMPLETE] %s", truncate(output, 500)))

	// 释放 Agent
	if agentID, ok := gw.tasks[taskID]; ok {
		gw.mu.Lock()
		delete(gw.tasks, taskID)
		gw.mu.Unlock()

		db.DB.Model(&model.AgentInstance{}).Where("id = ?", agentID).Updates(map[string]interface{}{
			"status":         "online",
			"current_task_id": "",
		})

		gw.dispatchQueuedTasks(agentID)
	}
}

func (gw *AgentGateway) handleComplete(payload map[string]interface{}) {
	taskID, _ := payload["task_id"].(string)
	output, _ := payload["output"].(string)

	now := time.Now()
	db.DB.Model(&model.AgentTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status":       "completed",
		"completed_at": now,
	})

	// 记录最终输出
	db.DB.Create(&model.AgentTaskLog{
		TaskID:  taskID,
		Type:    "system",
		Content: fmt.Sprintf("[TASK COMPLETE] %s", truncate(output, 500)),
	})

	// 释放 Agent
	if agentID, ok := gw.tasks[taskID]; ok {
		gw.mu.Lock()
		delete(gw.tasks, taskID)
		gw.mu.Unlock()

		db.DB.Model(&model.AgentInstance{}).Where("id = ?", agentID).Updates(map[string]interface{}{
			"status":         "online",
			"current_task_id": "",
		})

		// 下发下一个排队任务
		gw.dispatchQueuedTasks(agentID)
	}
}

// ─── 任务错误 ───

func (gw *AgentGateway) handleError(payload map[string]interface{}) {
	taskID, _ := payload["task_id"].(string)
	errMsg, _ := payload["error"].(string)

	db.DB.Model(&model.AgentTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status": "failed",
		"error":  errMsg,
	})

	db.DB.Create(&model.AgentTaskLog{
		TaskID:  taskID,
		Type:    "stderr",
		Content: fmt.Sprintf("[ERROR] %s", errMsg),
	})

	// 释放 Agent
	if agentID, ok := gw.tasks[taskID]; ok {
		gw.mu.Lock()
		delete(gw.tasks, taskID)
		gw.mu.Unlock()

		db.DB.Model(&model.AgentInstance{}).Where("id = ?", agentID).Updates(map[string]interface{}{
			"status":          "online",
			"current_task_id": "",
		})
	}
}

// ─── 审批请求 ───

func (gw *AgentGateway) handleApprovalRequest(client *AgentClient, payload map[string]interface{}) {
	taskID, _ := payload["task_id"].(string)
	question, _ := payload["question"].(string)
	context, _ := payload["context"].(string)
	suggested, _ := payload["suggested"].(string)

	req := model.ApprovalRequest{
		TaskID:    taskID,
		AgentID:   client.ID,
		Question:  question,
		Context:   context,
		Suggested: suggested,
		Status:    "pending",
	}
	db.DB.Create(&req)

	// 更新任务状态
	db.DB.Model(&model.AgentTask{}).Where("id = ?", taskID).
		Update("status", "waiting_input")

	log.Printf("[Approval] Task %s needs input: %s", taskID, question)
}

// ─── Hub → Agent 消息发送 ───

func (gw *AgentGateway) sendTo(client *AgentClient, msg WSMessage) {
	data, _ := json.Marshal(msg)
	select {
	case client.Send <- data:
	default:
		log.Printf("[Gateway] Send buffer full for agent %d", client.ID)
	}
}

func (gw *AgentGateway) SendToAgent(agentID uint, msg WSMessage) error {
	gw.mu.RLock()
	client, ok := gw.agents[agentID]
	gw.mu.RUnlock()
	if !ok {
		return fmt.Errorf("agent %d not connected", agentID)
	}
	gw.sendTo(client, msg)
	return nil
}

// ─── 任务分发 (原子操作防竞态 + 能力匹配 + 上下文链) ───

// findMatchingAgent 为指定任务查找匹配的在线 Agent
// 返回 (agentID, matched bool, err)
func (gw *AgentGateway) findMatchingAgent(task *model.AgentTask) (uint, bool, error) {
	gw.mu.RLock()
	onlineAgents := make([]uint, 0, len(gw.agents))
	for id := range gw.agents {
		onlineAgents = append(onlineAgents, id)
	}
	gw.mu.RUnlock()

	if len(onlineAgents) == 0 {
		return 0, false, nil
	}

	// 解析任务所需技能
	var requiredSkills []string
	if task.RequiredSkills != "" {
		json.Unmarshal([]byte(task.RequiredSkills), &requiredSkills)
	}

	// 如果没有指定技能要求，随机分配第一个空闲 Agent
	if len(requiredSkills) == 0 {
		for _, agentID := range onlineAgents {
			var agent model.AgentInstance
			if db.DB.First(&agent, agentID).Error == nil && agent.Status == "online" {
				return agentID, true, nil
			}
		}
		return 0, false, nil
	}

	// 按技能匹配
	for _, agentID := range onlineAgents {
		var agent model.AgentInstance
		if db.DB.First(&agent, agentID).Error != nil {
			continue
		}
		if agent.Status != "online" {
			continue
		}

		// 检查 Agent 是否满足所有 required_skills
		if agentHasAllCapabilities(agent.Capabilities, requiredSkills) {
			return agentID, true, nil
		}
	}

	return 0, false, nil // 无匹配 Agent
}

// agentHasAllCapabilities 检查 Agent 是否具备所有所需技能
func agentHasAllCapabilities(capabilitiesJSON string, requiredSkills []string) bool {
	var caps []string
	if capabilitiesJSON != "" {
		json.Unmarshal([]byte(capabilitiesJSON), &caps)
	}

	capSet := make(map[string]bool)
	for _, c := range caps {
		capSet[c] = true
	}

	for _, req := range requiredSkills {
		if !capSet[req] {
			return false
		}
	}
	return true
}

// buildPromptWithContext 为任务构建包含上下文的完整 Prompt
func (gw *AgentGateway) buildPromptWithContext(task *model.AgentTask) string {
	prompt := task.Prompt

	// 如果有父任务，注入上下文
	if task.ParentTaskID != "" {
		var parentTask model.AgentTask
		if err := db.DB.Where("id = ?", task.ParentTaskID).First(&parentTask).Error; err == nil {
			// 获取父任务的最终输出（最后一条系统日志或用户输出）
			var lastLog model.AgentTaskLog
			db.DB.Where("task_id = ?", parentTask.ID).
				Order("timestamp desc").
				First(&lastLog)

			context := fmt.Sprintf("\n\n--- 上下文（来自父任务: %s）---\n任务: %s\n", parentTask.ID, parentTask.Title)
			if task.ContextSnapshot != "" {
				context += fmt.Sprintf("上下文快照: %s\n", task.ContextSnapshot)
			}
			if lastLog.Content != "" {
				context += fmt.Sprintf("父任务输出: %s\n", truncate(lastLog.Content, 2000))
			}
			context += "--- 上下文结束 ---\n\n"

			// 将上下文前置到 Prompt
			prompt = context + prompt
		}
	}

	return prompt
}

func (gw *AgentGateway) dispatchQueuedTasks(agentID uint) {
	// 如果指定了 agentID（Agent 刚上线），优先分配给该 Agent
	if agentID != 0 {
		gw.DispatchToSpecificAgent(agentID)
		return
	}

	// 否则按能力匹配分发
	gw.DispatchByCapabilityMatching()
}

// DispatchToSpecificAgent 将排队任务分发给指定 Agent (公开方法)
func (gw *AgentGateway) DispatchToSpecificAgent(agentID uint) {
	var dispatchedTask model.AgentTask

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var task model.AgentTask
		if err := tx.Where("status = ?", "queued").
			Order("priority desc, created_at asc").
			First(&task).Error; err != nil {
			return err
		}

		// 检查技能匹配（如果任务有要求）
		var agent model.AgentInstance
		if tx.First(&agent, agentID).Error != nil {
			return errors.New("agent not found")
		}

		if task.RequiredSkills != "" {
			var requiredSkills []string
			json.Unmarshal([]byte(task.RequiredSkills), &requiredSkills)
			if !agentHasAllCapabilities(agent.Capabilities, requiredSkills) {
				return errors.New("agent lacks required skills")
			}
		}

		result := tx.Model(&model.AgentTask{}).
			Where("id = ? AND status = ?", task.ID, "queued").
			Updates(map[string]interface{}{
				"status":     "running",
				"agent_id":   agentID,
				"started_at": time.Now(),
				"timeout_at": time.Now().Add(time.Duration(task.TimeoutMinutes) * time.Minute),
			})
		if result.Error != nil || result.RowsAffected == 0 {
			return errors.New("task already claimed")
		}

		tx.Create(&model.AgentTaskLog{
			TaskID:  task.ID,
			Type:    "system",
			Content: fmt.Sprintf("[SYSTEM] Task assigned to agent %d", agentID),
		})

		gw.mu.Lock()
		gw.tasks[task.ID] = agentID
		gw.mu.Unlock()

		dispatchedTask = task
		return nil
	})

	if err != nil {
		return
	}

	db.DB.Model(&model.AgentInstance{}).Where("id = ?", agentID).Updates(map[string]interface{}{
		"status":          "busy",
		"current_task_id": "",
	})

	finalPrompt := gw.buildPromptWithContext(&dispatchedTask)

	log.Printf("[Gateway] Dispatched task %s to agent %d", dispatchedTask.ID, agentID)

	gw.SendToAgent(agentID, WSMessage{
		Method: "task.start",
		Payload: map[string]interface{}{
			"task_id": dispatchedTask.ID,
			"title":   dispatchedTask.Title,
			"prompt":  finalPrompt,
		},
	})
}

// DispatchByCapabilityMatching 按能力匹配自动分发排队任务 (公开方法)
func (gw *AgentGateway) DispatchByCapabilityMatching() {
	var tasks []model.AgentTask
	db.DB.Where("status = ?", "queued").
		Order("priority desc, created_at asc").
		Find(&tasks)

	for i := range tasks {
		task := &tasks[i]
		agentID, matched, err := gw.findMatchingAgent(task)
		if err != nil || !matched {
			continue
		}

		// 原子抢占
		result := db.DB.Model(&model.AgentTask{}).
			Where("id = ? AND status = ?", task.ID, "queued").
			Updates(map[string]interface{}{
				"status":     "running",
				"agent_id":   agentID,
				"started_at": time.Now(),
				"timeout_at": time.Now().Add(time.Duration(task.TimeoutMinutes) * time.Minute),
			})
		if result.Error != nil || result.RowsAffected == 0 {
			continue
		}

		db.DB.Create(&model.AgentTaskLog{
			TaskID:  task.ID,
			Type:    "system",
			Content: fmt.Sprintf("[SYSTEM] Task auto-assigned to agent %d (capability match)", agentID),
		})

		gw.mu.Lock()
		gw.tasks[task.ID] = agentID
		gw.mu.Unlock()

		db.DB.Model(&model.AgentInstance{}).Where("id = ?", agentID).Updates(map[string]interface{}{
			"status":          "busy",
			"current_task_id": "",
		})

		finalPrompt := gw.buildPromptWithContext(task)

		log.Printf("[Gateway] Auto-dispatched task %s to agent %d (capability match)", task.ID, agentID)

		gw.SendToAgent(agentID, WSMessage{
			Method: "task.start",
			Payload: map[string]interface{}{
				"task_id": task.ID,
				"title":   task.Title,
				"prompt":  finalPrompt,
			},
		})
	}
}

// ─── 取消任务 ───

func (gw *AgentGateway) CancelTask(taskID string) error {
	gw.mu.RLock()
	agentID, ok := gw.tasks[taskID]
	gw.mu.RUnlock()
	if !ok {
		// 任务未分配，直接更新状态
		db.DB.Model(&model.AgentTask{}).Where("id = ?", taskID).Update("status", "cancelled")
		return nil
	}

	err := gw.SendToAgent(agentID, WSMessage{
		Method: "task.cancel",
		Payload: map[string]interface{}{
			"task_id": taskID,
		},
	})
	if err != nil {
		return err
	}

	db.DB.Model(&model.AgentTask{}).Where("id = ?", taskID).Update("status", "cancelled")
	return nil
}

// ─── 回复审批 ───

func (gw *AgentGateway) ReplyToApproval(approvalID uint, reply string, approved bool) error {
	var req model.ApprovalRequest
	if err := db.DB.First(&req, approvalID).Error; err != nil {
		return err
	}

	status := "rejected"
	if approved {
		status = "approved"
	}

	db.DB.Model(&req).Updates(map[string]interface{}{
		"status":      status,
		"reply":       reply,
		"resolved_at": time.Now(),
	})

	// 通知 Agent
	gw.SendToAgent(req.AgentID, WSMessage{
		Method: "approval.response",
		Payload: map[string]interface{}{
			"task_id":  req.TaskID,
			"approved": approved,
			"reply":    reply,
		},
	})

	// 恢复任务执行
	if approved {
		db.DB.Model(&model.AgentTask{}).Where("id = ?", req.TaskID).
			Update("status", "running")
	}

	return nil
}

// ─── 心跳检查 ───

func (gw *AgentGateway) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		gw.mu.RLock()
		for _, client := range gw.agents {
			// 发送心跳
			gw.sendTo(client, WSMessage{Method: "system.ping"})
		}
		gw.mu.RUnlock()

		// 检查超时 (60s 无心跳)
		cutoff := time.Now().Add(-60 * time.Second)
		db.DB.Model(&model.AgentInstance{}).
			Where("status != 'offline' AND last_heartbeat < ?", cutoff).
			Update("status", "offline")
	}
}

// ─── 工具函数 ───

func (gw *AgentGateway) OnlineAgents() []uint {
	gw.mu.RLock()
	defer gw.mu.RUnlock()
	var ids []uint
	for id := range gw.agents {
		ids = append(ids, id)
	}
	return ids
}

func (client *AgentClient) writePump() {
	defer client.Conn.Close()
	for data := range client.Send {
		if err := client.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			break
		}
	}
}

func (gw *AgentGateway) Unregister(client *AgentClient) {
	if client.ID == 0 {
		return
	}
	gw.mu.Lock()
	delete(gw.agents, client.ID)
	gw.mu.Unlock()

	db.DB.Model(&model.AgentInstance{}).Where("id = ?", client.ID).
		Update("status", "offline")
	log.Printf("[Gateway] Agent disconnected: id=%d", client.ID)
}

// ─── 任务超时检查 ───

func (gw *AgentGateway) timeoutCheckerLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var tasks []model.AgentTask
		// 查找运行中且超时的任务
		db.DB.Where("status = ? AND timeout_at < ? AND timeout_at != ?", "running", time.Now(), "0001-01-01").
			Find(&tasks)

		for _, task := range tasks {
			log.Printf("[Gateway] Task %s timed out, marking as failed", task.ID)
			
			db.DB.Model(&model.AgentTask{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
				"status": "failed",
				"error":  "Task execution timed out",
			})
			db.DB.Create(&model.AgentTaskLog{
				TaskID:  task.ID,
				Type:    "stderr",
				Content: "[SYSTEM] Task execution timed out and was auto-cancelled",
			})
			broadcastTaskLog(task.ID, "stderr", "[SYSTEM] Task execution timed out and was auto-cancelled")

			// 释放 Agent
			gw.mu.Lock()
			if agentID, ok := gw.tasks[task.ID]; ok {
				delete(gw.tasks, task.ID)
				gw.mu.Unlock()

				db.DB.Model(&model.AgentInstance{}).Where("id = ?", agentID).Updates(map[string]interface{}{
					"status":          "online",
					"current_task_id": "",
				})
				// 下发下一个任务
				gw.dispatchQueuedTasks(agentID)
			} else {
				gw.mu.Unlock()
			}
		}
	}
}

// ─── 实时日志广播 (SSE) ───

type logSub struct {
	ch chan model.AgentTaskLog
}

var (
	logSubs   = make(map[string][]logSub)
	logSubsMu sync.RWMutex
)

// SubscribeTaskLogs 订阅指定任务的日志流
func SubscribeTaskLogs(taskID string) chan model.AgentTaskLog {
	ch := make(chan model.AgentTaskLog, 64)
	logSubsMu.Lock()
	logSubs[taskID] = append(logSubs[taskID], logSub{ch: ch})
	logSubsMu.Unlock()
	return ch
}

// UnsubscribeTaskLogs 取消订阅
func UnsubscribeTaskLogs(taskID string, ch chan model.AgentTaskLog) {
	logSubsMu.Lock()
	subs := logSubs[taskID]
	for i, s := range subs {
		if s.ch == ch {
			logSubs[taskID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	close(ch)
	logSubsMu.Unlock()
}

// broadcastTaskLog 广播日志给所有 SSE 订阅者
func broadcastTaskLog(taskID string, logType string, content string) {
	entry := model.AgentTaskLog{
		TaskID:    taskID,
		Type:      logType,
		Content:   content,
		Timestamp: time.Now(),
	}
	
	logSubsMu.RLock()
	subs := logSubs[taskID]
	logSubsMu.RUnlock()
	
	for _, s := range subs {
		select {
		case s.ch <- entry:
		default:
			// 缓冲区满，跳过 (客户端会断线重连)
		}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	// 简单截断
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
