package model

import "time"

// ─── 原有模型 ───

type User struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	Username string `json:"username" gorm:"uniqueIndex"`
	Password string `json:"-"`
	Role     string `json:"role" gorm:"default:user"` // admin, user
	Token    string `json:"token" gorm:"column:api_token"`
}

type Agent struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	AgentID    string    `json:"agent_id" gorm:"uniqueIndex"`
	Name       string    `json:"name"`
	Status     string    `json:"status" gorm:"default:offline"` // online, offline, busy
	LastPing   time.Time `json:"last_ping"`
}

type Task struct {
	ID               string    `json:"id" gorm:"primaryKey"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Status           string    `json:"status" gorm:"default:pending"`
	Type             string    `json:"type"` // serial, parallel, flow
	CreatorID        uint      `json:"creator_id"`
	AgentID          string    `json:"agent_id"`
	FlowID           uint      `json:"flow_id"`       // 关联的编排流程 ID
	FlowExecutionID  uint      `json:"flow_execution_id"` // 关联的具体流程执行记录 ID
	FlowParams       string    `json:"flow_params"`   // 流程运行时的参数 JSON
	Source           string    `json:"source" gorm:"default:manual"` // manual, flow
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type TaskStep struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TaskID    string    `json:"task_id"`
	AgentID   string    `json:"agent_id"`
	Command   string    `json:"command"`
	Status    string    `json:"status" gorm:"default:pending"`
	Output    string    `json:"output"`
	InputReq  string    `json:"input_req,omitempty"`
	UserReply string    `json:"user_reply,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type File struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	FileName  string    `json:"file_name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	Uploader  string    `json:"uploader"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── 新增: Agent 终端管理 ───

// AgentTerminal: 远程 AI 终端连接实例
type AgentTerminal struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"`
	Endpoint    string    `json:"endpoint"`       // ws://host:port 或 wss://
	Token       string    `json:"token"`          // 认证令牌
	Status      string    `json:"status"`         // connected, disconnected, error
	LastConnect time.Time `json:"last_connect"`
	Config      string    `json:"config"`         // JSON 扩展配置
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ─── 新增: Agent 编排 ───

// AgentFlow: 可视化编排的流程定义
type AgentFlow struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Graph       string    `json:"graph"`             // JSON: 节点/边定义
	Type        string    `json:"type"`              // linear, dag, parallel
	Status      string    `json:"status" gorm:"default:draft"` // draft, active, archived
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AgentNode: 编排中的单个节点
type AgentNode struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	FlowID     uint   `json:"flow_id" gorm:"index"`
	NodeID     string `json:"node_id"`               // 前端画布上的节点 ID
	Name       string `json:"name"`
	Type       string `json:"type"`                  // agent, condition, merge, trigger
	AgentID    string `json:"agent_id"`              // 关联的 Agent
	Config     string `json:"config"`                // JSON 节点配置
	PositionX  float64 `json:"position_x"`
	PositionY  float64 `json:"position_y"`
}

// AgentEdge: 编排中的连接边
type AgentEdge struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	FlowID     uint   `json:"flow_id" gorm:"index"`
	SourceID   string `json:"source_id"`
	TargetID   string `json:"target_id"`
	Condition  string `json:"condition"`             // 条件表达式 (可选)
}

// ─── 新增: 中枢知识管理 (Memory Hub) ───

// KnowledgeEntry: 知识条目
type KnowledgeEntry struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`           // Markdown 内容
	Summary     string    `json:"summary"`           // AI 生成的摘要
	Type        string    `json:"type"`              // concept, procedure, fact, decision
	Category    string    `json:"category"`          // 分类
	Tags        string    `json:"tags"`              // 逗号分隔的标签
	Source      string    `json:"source"`            // 来源 (chat, upload, manual, agent)
	Status      string    `json:"status" gorm:"default:pending"` // pending, approved, rejected, archived
	Confidence  float64   `json:"confidence"`        // AI 置信度 0-1
	ReviewerID  uint      `json:"reviewer_id"`       // 审核人
	ReviewedAt  time.Time `json:"reviewed_at"`
	CreatedBy   uint      `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MemorySession: 会话记忆 (用于沉淀业务知识)
type MemorySession struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	SessionID   string    `json:"session_id" gorm:"uniqueIndex"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	Tags        string    `json:"tags"`
	Domain      string    `json:"domain"`            // 知识领域
	Messages    string    `json:"messages"`          // JSON: 对话记录
	Extracted   string    `json:"extracted"`         // 提取的知识点 (关联 KnowledgeEntry IDs)
	CreatedAt   time.Time `json:"created_at"`
}

// AuditLog: 审核日志
type AuditLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	EntryID   uint      `json:"entry_id"`
	Action    string    `json:"action"`              // approve, reject, edit, tag
	UserID    uint      `json:"user_id"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── 自动迁移注册 ───

func AutoMigrateList() []interface{} {
	return []interface{}{
		&User{}, &Agent{}, &Task{}, &TaskStep{}, &File{},
		&AgentTerminal{}, &AgentFlow{}, &AgentNode{}, &AgentEdge{},
		&KnowledgeEntry{}, &MemorySession{}, &AuditLog{},
	}
}
