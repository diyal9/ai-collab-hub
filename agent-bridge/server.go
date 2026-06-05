package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// TaskResult 任务执行结果
type TaskResult struct {
	Status string `json:"status"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// SandboxInfo 记录连接的 Sandbox 信息
type SandboxInfo struct {
	ID           string
	Capabilities []string
	Conn         *websocket.Conn
}

// SandboxManager 管理所有连接的 Sandbox 节点
type SandboxManager struct {
	mu          sync.RWMutex
	sandboxes   map[string]*SandboxInfo
	pendingTasks map[string]chan TaskResult // taskID -> result channel
	taskLogs    sync.Map                   // taskID -> log string
	upgrader    websocket.Upgrader
}

var Manager = &SandboxManager{
	sandboxes:    make(map[string]*SandboxInfo),
	pendingTasks: make(map[string]chan TaskResult),
	upgrader: websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	},
}

func (m *SandboxManager) Register(id string, conn *websocket.Conn, caps []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sandboxes[id] = &SandboxInfo{ID: id, Capabilities: caps, Conn: conn}
	log.Printf("[Manager] Sandbox '%s' registered. Capabilities: %v. Total: %d", id, caps, len(m.sandboxes))
}

func (m *SandboxManager) Unregister(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sandboxes, id)
	log.Printf("[Manager] Sandbox '%s' disconnected. Total: %d", id, len(m.sandboxes))
}

func (m *SandboxManager) GetSandbox(id string) *SandboxInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sandboxes[id]
}

// RegisterTask 注册一个等待中的任务
func (m *SandboxManager) RegisterTask(taskID string) chan TaskResult {
	ch := make(chan TaskResult, 1)
	m.mu.Lock()
	m.pendingTasks[taskID] = ch
	m.mu.Unlock()
	return ch
}

// CompleteTask 完成任务，通知等待方
func (m *SandboxManager) CompleteTask(taskID string, result TaskResult) {
	m.mu.Lock()
	ch, exists := m.pendingTasks[taskID]
	if exists {
		delete(m.pendingTasks, taskID)
	}
	m.mu.Unlock()

	if exists {
		ch <- result
		close(ch)
	}
}

// HandleSandboxMessage 处理来自 Sandbox 的消息（含结果回传）
func (m *SandboxManager) HandleSandboxMessage(data []byte, conn *websocket.Conn) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	// 1. 处理日志消息 -> 缓存日志
	if msg.Type == MsgTaskLog && msg.TaskID != "" {
		var logData LogPayload
		if err := json.Unmarshal(msg.Payload, &logData); err == nil {
			// 追加到现有日志
			currentLogs, _ := m.taskLogs.LoadOrStore(msg.TaskID, "")
			m.taskLogs.Store(msg.TaskID, currentLogs.(string)+logData.Data)
		}
	}

	// 2. 处理任务状态/结果消息，尝试完成 pending task
	if msg.Type == MsgTaskStatus && msg.TaskID != "" {
		var status StatusPayload
		if err := json.Unmarshal(msg.Payload, &status); err == nil {
			if status.Status == "completed" || status.Status == "failed" {
				result := TaskResult{Status: status.Status}
				if msg.Error != "" {
					result.Error = msg.Error
				}
				// 附加收集的日志
				if logs, ok := m.taskLogs.LoadAndDelete(msg.TaskID); ok {
					result.Output = logs.(string)
				}
				m.CompleteTask(msg.TaskID, result)
			}
		}
	}

	// 透传给 Hub (可选，用于日志展示)
	HubClient.SendToHub(data)
}

// Dispatch 将任务分发给指定 Sandbox
func (m *SandboxManager) Dispatch(sandboxID string, taskID string, payload TaskPayload) (TaskResult, error) {
	si := m.GetSandbox(sandboxID)
	if si == nil {
		return TaskResult{}, fmt.Errorf("sandbox '%s' not found", sandboxID)
	}

	// 检查能力
	hasExecutor := false
	for _, cap := range si.Capabilities {
		if cap == payload.Executor {
			hasExecutor = true
			break
		}
	}
	if !hasExecutor {
		return TaskResult{}, fmt.Errorf("sandbox '%s' does not support executor '%s'", sandboxID, payload.Executor)
	}

	// 注册等待 channel
	resultCh := m.RegisterTask(taskID)

	// 下发任务
	data, err := json.Marshal(struct {
		Type    string      `json:"type"`
		TaskID  string      `json:"task_id"`
		Payload TaskPayload `json:"payload"`
	}{
		Type:    MsgTaskAssign,
		TaskID:  taskID,
		Payload: payload,
	})
	if err != nil {
		return TaskResult{}, err
	}

	if err := si.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return TaskResult{}, err
	}

	log.Printf("[Dispatch] Task %s dispatched to sandbox %s", taskID, sandboxID)

	// 阻塞等待结果 (带超时)
	select {
	case result := <-resultCh:
		return result, nil
	case <-time.After(5 * time.Minute): // 默认超时 5 分钟
		return TaskResult{Status: "failed", Error: "Task timeout"}, fmt.Errorf("task %s timed out", taskID)
	}
}

// ListSandboxes 返回所有已注册的 Sandbox
func (m *SandboxManager) ListSandboxes() []map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []map[string]interface{}
	for id, si := range m.sandboxes {
		list = append(list, map[string]interface{}{
			"id":           id,
			"capabilities": si.Capabilities,
		})
	}
	return list
}

// HandleSandboxConnection 处理来自 Sandbox 的 WebSocket 连接
func HandleSandboxConnection(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = "unknown-sandbox"
	}

	conn, err := Manager.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Server] Upgrade error: %v", err)
		return
	}

	// 等待注册消息
	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		log.Printf("[Server] Failed to read registration message from %s: %v", id, err)
		return
	}

	var msg Message
	if err := json.Unmarshal(msgBytes, &msg); err != nil || msg.Type != MsgRegister {
		log.Printf("[Server] Invalid registration message from %s", id)
		return
	}

	var regPayload RegisterPayload
	if err := json.Unmarshal(msg.Payload, &regPayload); err != nil {
		log.Printf("[Server] Invalid register payload: %v", err)
		return
	}

	// 注册
	Manager.Register(regPayload.SandboxID, conn, regPayload.Capabilities)

	// 保持连接活跃并处理消息
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				log.Printf("[Server] Read error from %s: %v", regPayload.SandboxID, err)
				Manager.Unregister(regPayload.SandboxID)
				conn.Close()
				return
			}
			// 处理消息（含结果回传逻辑）
			Manager.HandleSandboxMessage(data, conn)
		}
	}()
}

// HandleDispatch HTTP 接口：Hub 通过此接口下发任务 (同步阻塞等待结果)
func HandleDispatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SandboxID string      `json:"sandbox_id"`
		TaskID    string      `json:"task_id"`
		Payload   TaskPayload `json:"payload"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := Manager.Dispatch(req.SandboxID, req.TaskID, req.Payload)
	if err != nil {
		log.Printf("[Dispatch] Failed to dispatch task %s: %v", req.TaskID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 返回执行结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleListSandboxes HTTP 接口：列出可用 Sandbox
func HandleListSandboxes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Manager.ListSandboxes())
}
