package main

import "encoding/json"

// Bridge 侧的消息类型定义 (与 Sandbox 对齐)
const (
	MsgHeartbeat  = "heartbeat"
	MsgRegister   = "register"
	MsgTaskAssign = "task.assign"
	MsgTaskStatus = "task.status"
	MsgTaskLog    = "task.log"
	MsgTaskResult = "task.result"
)

type Message struct {
	Type    string          `json:"type"`
	TaskID  string          `json:"task_id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type RegisterPayload struct {
	SandboxID    string   `json:"sandbox_id"`
	Capabilities []string `json:"capabilities"`
}

type TaskPayload struct {
	Executor string            `json:"executor"`
	Command  string            `json:"command"`
	WorkDir  string            `json:"work_dir"`
	Env      map[string]string `json:"env"`
	Timeout  int               `json:"timeout"`
}

type StatusPayload struct {
	Status string `json:"status"`
}

type LogPayload struct {
	Stream string `json:"stream"`
	Data   string `json:"data"`
}
