package model

import "time"

// AgentInstance: 注册的 AI Agent 实例 (Hermes/Codex/Cursor)
type AgentInstance struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	Name            string    `json:"name"`                    // 显示名称，如 "hermes-main", "codex-worker-1"
	Type            string    `json:"type"`                    // hermes, codex, cursor
	Status          string    `json:"status"`                  // online, offline, busy, error
	Platform        string    `json:"platform"`                // linux, macos, windows
	Host            string    `json:"host"`                    // 来源 IP 或主机名
	Capabilities    string    `json:"capabilities"`            // JSON: ["terminal", "browser", "file", "web"]
	CurrentTaskID   string    `json:"current_task_id"`         // 当前执行的任务 ID
	LastHeartbeat   time.Time `json:"last_heartbeat"`
	Token           string    `json:"-"`                       // 认证 token (不返回前端)
	Config          string    `json:"config"`                  // JSON: 运行配置 (workdir, env, etc.)
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// AgentSession: Agent 与 Hub 的一次连接会话
type AgentSession struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	AgentID    uint      `json:"agent_id" gorm:"index"`
	Status     string    `json:"status"`           // active, closed, timeout
	StartedAt  time.Time `json:"started_at"`
	EndedAt    time.Time `json:"ended_at"`
	MessageLog string    `json:"message_log"`      // JSON: 简要消息统计 {total: N, types: {...}}
}

// AgentTask: Hub 下发给 Agent 的具体任务
type AgentTask struct {
	ID          string    `json:"id" gorm:"primaryKey"`           // 任务 UUID
	AgentID     uint      `json:"agent_id" gorm:"index"`
	Title       string    `json:"title"`
	Prompt      string    `json:"prompt"`                         // 给 Agent 的完整指令
	Status      string    `json:"status"`                         // queued, running, waiting_input, completed, failed, cancelled
	Priority    int       `json:"priority"`                       // 0=low, 1=normal, 2=high
	RequiredSkills string `json:"required_skills" gorm:"type:text"` // JSON: ["coding", "git", "browser"]
	ParentTaskID string   `json:"parent_task_id"`                 // 父任务 ID (用于上下文链)
	ContextSnapshot string `json:"context_snapshot" gorm:"type:text"` // 上下文快照 (父任务输出等)
	// v2 Agent Bridge 扩展字段
	Executor        string    `json:"executor" gorm:"default:shell"`        // 执行环境: "shell", "llm", "cursor", "browser"
	ExecutorConfig  string    `json:"executor_config" gorm:"type:text"`     // JSON: { model, work_dir, timeout, env, allowed_cmds, max_memory, project_path, target_file }
	ConversationMode bool     `json:"conversation_mode" gorm:"default:false"` // 是否多轮对话模式
	Result          string    `json:"result" gorm:"type:text"`              // 任务最终输出/结果
	Steps           string    `json:"steps" gorm:"type:text"`               // JSON: 多步骤定义
	Progress        int       `json:"progress" gorm:"default:0"`            // 当前进度百分比 (0-100)
	CreatedAt       time.Time `json:"created_at"`
	StartedAt       time.Time `json:"started_at"`
	CompletedAt     time.Time `json:"completed_at"`
	Error           string    `json:"error"`
	TimeoutAt       time.Time `json:"timeout_at"`                   // 任务超时时间点
	TimeoutMinutes  int       `json:"timeout_minutes" gorm:"default:30"` // 默认30分钟超时
}

// LLMProviderConfig: LLM 服务提供者配置
type LLMProviderConfig struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name"`                   // openai, anthropic, ollama
	BaseURL      string    `json:"base_url"`               // API 基础地址
	APIKey       string    `json:"api_key"`                // API 密钥 (不返回前端)
	DefaultModel string    `json:"default_model"`          // 默认模型
	MaxTokens    int       `json:"max_tokens"`             // 最大输出 tokens
	Timeout      int       `json:"timeout"`                // 请求超时 (秒)
	Enabled      bool      `json:"enabled" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AgentTaskLog: Agent 任务的实时日志流
type AgentTaskLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TaskID    string    `json:"task_id" gorm:"index"`
	Type      string    `json:"type"`           // stdout, stderr, system, user_input, approval_request
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp" gorm:"autoCreateTime"`
}

// ApprovalRequest: 人类介入审批请求
type ApprovalRequest struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TaskID    string    `json:"task_id" gorm:"index"`
	AgentID   uint      `json:"agent_id"`
	Question  string    `json:"question"`           // Agent 提出的问题
	Context   string    `json:"context"`            // 上下文 (当前工作、代码片段等)
	Suggested string    `json:"suggested"`          // Agent 建议的答案 (可选)
	Status    string    `json:"status"`             // pending, approved, rejected
	Reply     string    `json:"reply"`              // 用户的回答
	ReplyBy   uint      `json:"reply_by"`           // 回答者 ID
	CreatedAt time.Time `json:"created_at"`
	ResolvedAt time.Time `json:"resolved_at"`
}

func AutoMigrateListExtended() []interface{} {
	return []interface{}{
		&AgentInstance{}, &AgentSession{}, &AgentTask{}, &AgentTaskLog{}, &ApprovalRequest{},
		&LLMProviderConfig{},
	}
}
