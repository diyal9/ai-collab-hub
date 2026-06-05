package main

import "encoding/json"

// MessageType 定义了消息类型
const (
	MsgHeartbeat  = "heartbeat"
	MsgRegister   = "register"
	MsgTaskAssign = "task.assign"
	MsgTaskStatus = "task.status"
	MsgTaskLog    = "task.log"
	MsgTaskResult = "task.result"
)

// Message 是 Sandbox 与 Bridge 通信的标准信封
type Message struct {
	Type    string          `json:"type"`
	TaskID  string          `json:"task_id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// RegisterPayload Sandbox 注册信息
type RegisterPayload struct {
	SandboxID    string   `json:"sandbox_id"`
	Capabilities []string `json:"capabilities"` // ["shell", "cursor", "aider"]
}

// TaskPayload 包含具体的执行参数
type TaskPayload struct {
	Executor string            `json:"executor"`
	Command  string            `json:"command"`
	WorkDir  string            `json:"work_dir"`
	Env      map[string]string `json:"env"`
	Timeout  int               `json:"timeout"`
}

// StatusPayload 进度状态
type StatusPayload struct {
	Status string `json:"status"`
}

// LogPayload 实时日志
type LogPayload struct {
	Stream string `json:"stream"`
	Data   string `json:"data"`
}
