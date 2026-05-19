# AI Agent Bridge

将 AI Agent (Hermes, Codex, Cursor) 连接到 AI Collab Hub 的桥接服务。

## 架构

```
AI Collab Hub (本平台)
  └── WebSocket Gateway (/ws/agent)
        ├── Hermes Agent  ← agent-bridge (Go)
        ├── Codex CLI     ← agent-bridge (Go)  
        └── Cursor        ← cursor-mcp-bridge (Node.js)
```

## 快速部署

### 方式一: 一键部署脚本

```bash
# 在本平台获取 Token:
#   登录 → Agent 管理 → 添加 Agent → 复制命令

# 在 Agent 机器上运行:
curl -sSL https://47.107.172.201:8089/hermesshare/agent-bridge/deploy.sh | bash -s -- \
  HUB_URL=wss://47.107.172.201:8089/ws/agent \
  AGENT_NAME=my-agent \
  AGENT_TYPE=hermes \
  AGENT_TOKEN=agt_hermes_xxx
```

### 方式二: 手动部署

#### Hermes Agent

```bash
# 1. 复制 agent-bridge 到目标机器
scp agent-bridge user@remote:/opt/agent-bridge/

# 2. 启动
./agent-bridge \
  --hub-url wss://47.107.172.201:8089/ws/agent \
  --name hermes-main \
  --type hermes \
  --token agt_hermes_xxx
```

#### Codex CLI

```bash
./agent-bridge \
  --hub-url wss://47.107.172.201:8089/ws/agent \
  --name codex-worker-1 \
  --type codex \
  --token agt_codex_xxx
```

#### Cursor (MCP Server)

```bash
# 1. 安装依赖
cd cursor-mcp-bridge
npm install

# 2. 配置 Cursor MCP
# 在 Cursor 的 MCP 配置中添加:
{
  "mcpServers": {
    "ai-collab-hub": {
      "command": "node",
      "args": ["/path/to/cursor-mcp-bridge/index.js"],
      "env": {
        "HUB_URL": "wss://47.107.172.201:8089/ws/agent",
        "AGENT_NAME": "cursor-main",
        "AGENT_TYPE": "cursor",
        "AGENT_TOKEN": "agt_cursor_xxx"
      }
    }
  }
}
```

## 消息协议

### Agent → Hub

| 方法 | 说明 |
|------|------|
| `agent.register` | 注册上线 (name, type, token, capabilities) |
| `agent.heartbeat` | 心跳保活 (每 15s) |
| `task.progress` | 进度上报 (task_id, type, content) |
| `task.complete` | 任务完成 (task_id, output) |
| `task.error` | 任务失败 (task_id, error) |
| `approval.request` | 请求审批 (task_id, question, context) |

### Hub → Agent

| 方法 | 说明 |
|------|------|
| `task.start` | 下发任务 (task_id, title, prompt) |
| `task.cancel` | 取消任务 (task_id) |
| `approval.response` | 审批结果 (task_id, approved, reply) |
| `system.ping` | 心跳探测 |

## 审批流

```
Agent 执行 → 检测关键词 → 发 approval.request
  → Hub 弹出审批 → 用户批准/拒绝/回复
  → 发 approval.response → Agent 继续/中止
```

**自动触发关键词**: `/approve`, `需要确认`, `请审批`, `dangerous`, `DROP`, `rm -rf`, `sudo`

## API

| 端点 | 方法 | 说明 |
|------|------|------|
| `/ws/agent` | WS | Agent 连接端点 |
| `/api/agent-instances` | GET | Agent 列表 |
| `/api/agent-instances` | POST | 创建 Agent (生成 Token) |
| `/api/agent-tasks` | POST | 下发任务 |
| `/api/agent-tasks` | GET | 任务列表 |
| `/api/agent-tasks/:id/logs` | GET | 任务日志 |
| `/api/agent-tasks/:id/cancel` | POST | 取消任务 |
| `/api/approvals` | GET | 待审批列表 |
| `/api/approvals/:id/reply` | POST | 回复审批 |

## 文件结构

```
agent-bridge/
├── client.go          # Agent Bridge 客户端 (Go)
├── deploy.sh          # 一键部署脚本
├── go.mod
├── go.sum
└── agent-bridge       # 编译后的二进制

cursor-mcp-bridge/
├── index.js           # Cursor MCP Server (Node.js)
├── package.json
└── node_modules/

backend/
├── internal/gateway/  # WebSocket Gateway
│   └── gateway.go
├── internal/model/    # Agent 数据模型
│   └── agent_models.go
└── main.go            # API 路由

frontend/
└── src/views/
    └── AgentManager.vue  # Agent 管理页面
```

## 多 Agent 管理策略

### 角色划分

| Agent | 角色 | 能力 |
|-------|------|------|
| hermes-main | 主调度 | terminal, browser, file, web |
| codex-worker-1 | 代码执行 | code-execution, terminal, file |
| cursor-main | 编辑器 | editor, mcp, file, terminal |

### 任务分配

1. **串行**: 任务队列按优先级依次分发
2. **专用**: 指定 agent_id 下发到特定 Agent
3. **负载均衡**: 空闲 Agent 自动拉取排队任务

### 建议部署

- **开发环境**: 1 台机器跑 hermes-main
- **生产环境**: 
  - hermes-main → 主控机器 (长期运行)
  - codex-worker-N → 代码执行集群 (按需扩缩)
  - cursor-main → 开发者本地 (MCP 集成)
