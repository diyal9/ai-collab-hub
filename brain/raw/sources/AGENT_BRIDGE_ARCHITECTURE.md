# AI Agent Bridge 架构方案

## 总览

AI Collab Hub 作为 **Agent 编排中心**，通过 WebSocket Gateway 统一管理所有 AI Agent（Hermes、Codex、Cursor），实现：

1. **长任务调度** —— 下发 prompt，Agent 执行并实时上报进度
2. **人类介入审批** —— 敏感操作需要人工确认
3. **多 Agent 管理** —— 统一的注册、状态、任务队列

---

## 架构

```
┌────────────────────────────────────────────────────────────────┐
│                   AI Collab Hub (本平台)                        │
│                                                                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │ Task DAG │  │ Approval │  │ Log Hub  │  │ Agent Registry│  │
│  │  Engine  │  │  Queue   │  │(Realtime)│  │  (CRUD+Status)│  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └───────┬───────┘  │
│       │              │              │                │          │
│  ┌────┴──────────────┴──────────────┴────────────────┴──────┐  │
│  │           WebSocket Gateway (/ws/agent)                   │  │
│  │   协议: JSON-RPC 2.0 over WebSocket                       │  │
│  └───────────────────────┬──────────────────────────────────┘  │
└──────────────────────────┼─────────────────────────────────────┘
                           │ wss://hub/ws/agent
         ┌─────────────────┼─────────────────┐
         │                 │                  │
    ┌────┴────┐     ┌─────┴─────┐     ┌──────┴──────┐
    │  Hermes │     │   Codex   │     │   Cursor    │
    │  Agent  │     │    CLI    │     │  (MCP/Ext)  │
    │         │     │           │     │             │
    │ bridge  │     │  bridge   │     │  MCP Server │
    │ process │     │  process  │     │  (Node.js)  │
    └─────────┘     └───────────┘     └─────────────┘
   机器 A           机器 B             机器 C
```

---

## 消息协议 (JSON-RPC 2.0 over WebSocket)

### Agent → Hub

| 方法 | 说明 | Payload |
|------|------|---------|
| `agent.register` | 注册上线 | `{name, type, token, platform, host, capabilities}` |
| `agent.heartbeat` | 心跳保活 | - |
| `task.progress` | 进度上报 | `{task_id, type: stdout/stderr/system, content}` |
| `task.complete` | 任务完成 | `{task_id, output}` |
| `task.error` | 任务失败 | `{task_id, error}` |
| `approval.request` | 请求审批 | `{task_id, question, context, suggested}` |

### Hub → Agent

| 方法 | 说明 | Payload |
|------|------|---------|
| `task.start` | 下发任务 | `{task_id, title, prompt}` |
| `task.cancel` | 取消任务 | `{task_id}` |
| `approval.response` | 审批结果 | `{task_id, approved, reply}` |
| `system.ping` | 心跳探测 | - |

---

## 各 Agent 接入方式

### 1. Hermes Agent (推荐)

**部署方式**: 在 Hermes 运行机器上启动 bridge 进程

```bash
# 编译 bridge (需要 Go 环境)
cd /root/aispace/hermesshare/agent-bridge
go build -o agent-bridge .

# 启动 (连接到 Hub)
./agent-bridge \
  --hub-url wss://47.107.172.201:8089/ws/agent \
  --name hermes-main \
  --type hermes \
  --token your-secret-token
```

**Hermes 集成**: 在 Hermes Agent 的 SKILL 中，当遇到需要审批的操作时：
- 检测关键词 `/approve`、`需要确认`、危险操作（`DROP`、`rm -rf`、`sudo`）
- 通过 bridge 发送 `approval.request`
- 等待 `approval.response` 后继续

### 2. Codex CLI

Codex 本身不支持 WebSocket，需要通过 bridge 包装：

```bash
./agent-bridge \
  --hub-url wss://47.107.172.201:8089/ws/agent \
  --name codex-worker-1 \
  --type codex \
  --token your-secret-token
```

Bridge 收到任务后会调用 `codex run --prompt "..."`，并将输出实时上报。

**审批集成**: Codex 的确认操作（如 `y/N`）可通过 bridge 拦截并转发为 `approval.request`。

### 3. Cursor (MCP Server 方式)

Cursor 通过 **MCP Server** 与 Hub 交互，是最灵活的方式：

```
Hub (WebSocket)
  ←→ MCP Server (Node.js)
       ←→ Cursor Editor (via stdio/SSE)
```

**MCP Server 实现要点**:
```typescript
// cursor-mcp-bridge/index.ts
import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import WebSocket from 'ws';

const ws = new WebSocket('wss://your-hub/ws/agent');

// Hub 下发任务 → 写入 Cursor 可读取的文件
ws.on('message', (msg) => {
  const { method, payload } = JSON.parse(msg);
  if (method === 'task.start') {
    // 写入 Cursor 任务文件
    fs.writeFileSync(`/tmp/cursor-tasks/${payload.task_id}.md`, payload.prompt);
    // 通知 Cursor 有新任务 (通过 MCP tool)
  }
});

// MCP Tool: 上报进度
server.tool('reportProgress', async ({ taskId, content }) => {
  ws.send(JSON.stringify({
    method: 'task.progress',
    payload: { task_id: taskId, content }
  }));
  return { content: [{ type: 'text', text: 'OK' }] };
});
```

**Cursor 用户操作**:
1. 在 Cursor 中收到任务文件
2. 执行任务
3. 通过 MCP Tool `reportProgress` 上报进度
4. 遇到审批时调用 `requestApproval` MCP Tool

---

## 审批流完整示例

```
1. Hub 用户创建任务 → POST /api/agent-tasks
   {agent_id: 1, title: "重构登录模块", prompt: "..."}

2. Hub → Agent: task.start

3. Agent 执行任务，逐步上报:
   task.progress → "Step 1/5: 读取现有代码..."
   task.progress → "Step 2/5: 修改认证逻辑..."

4. Agent 发现危险操作:
   → approval.request {
       task_id: "task_xxx",
       question: "即将执行 DROP TABLE users_backup，确认？",
       context: "当前代码包含旧表迁移逻辑...",
       suggested: "建议先导出备份再删除"
     }

5. Hub 前端弹出审批通知 → 用户选择 "批准" / "拒绝" / "回复"

6. Hub → Agent: approval.response {approved: true, reply: "确认删除"}

7. Agent 继续执行 → task.complete
```

---

## API 列表

| 端点 | 方法 | 说明 |
|------|------|------|
| `/ws/agent` | WebSocket | Agent Bridge 连接端点 |
| `/api/agent-instances` | GET | 获取所有 Agent 实例 |
| `/api/agent-tasks` | POST | 下发任务给 Agent |
| `/api/agent-tasks` | GET | 任务列表 |
| `/api/agent-tasks/:id/logs` | GET | 任务日志 |
| `/api/agent-tasks/:id/cancel` | POST | 取消任务 |
| `/api/approvals` | GET | 待审批列表 |
| `/api/approvals/:id/reply` | POST | 回复审批 |

---

## 安全

1. **Token 认证**: Agent 注册时需提供 token，防止未授权接入
2. **路径安全**: 下载资源接口已做路径穿越防护
3. **审批拦截**: 敏感操作（数据库删除、sudo、rm -rf）自动触发审批
