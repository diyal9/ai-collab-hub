# Agent 桥接架构

Hub 通过 WebSocket Gateway 统一管理远端 Agent。

## 拓扑

```
Hub (/ws/agent)
  ├── agent-bridge (Go) → Hermes / Codex
  ├── cursor-mcp-bridge (Node) → Cursor IDE
  └── agent-sandbox → 远端 Shell/Cursor/Aider 执行
```

## 协议

JSON-RPC 2.0 over WebSocket。

### Agent → Hub

- `agent.register`, `agent.heartbeat`
- `task.progress`, `task.complete`, `task.error`
- `approval.request`

### Hub → Agent

- `task.start`, `task.cancel`
- `approval.response`, `system.ping`

## 审批触发词

`/approve`, `需要确认`, `DROP`, `rm -rf`, `sudo` 等。

## 详细文档

原始资料：`raw/sources/AGENT_BRIDGE_ARCHITECTURE.md`
