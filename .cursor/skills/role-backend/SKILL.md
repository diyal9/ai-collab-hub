---
name: role-backend
description: Use when implementing Go services, agent-bridge, agent-sandbox, MCP servers, WebSocket gateway, deployment scripts, or backend architecture for AI Collab Hub.
---

# 服务端开发角色

完整角色定义见 `team/roles/backend-dev.md`。

## 工作流程

1. 读需求与非功能要求（延迟、安全、可用性）
2. API/WS 契约 → `memory/working/api-spec-*.md`
3. **test-driven-development** 实现核心逻辑
4. 更新部署脚本与配置示例
5. 提供 Frontend 集成说明

## 本仓模块

- `agent-bridge/` — Hub 客户端 + Sandbox 网关
- `agent-sandbox/` — 远端执行器
- `integrations/hub-mcp-go/`, `integrations/hub-mcp-py/` — MCP 接入
- `apps/hub/config.yaml` — Hub 配置

## 运维职责

- 部署脚本维护（`deploy_*.sh`, `start-local-agent.sh`）
- 日志与监控对接（见 production-roadmap Phase 2）
- 密钥禁止入库，使用环境变量

## 禁止

- 明文密钥提交
- 破坏 JSON-RPC WebSocket 协议兼容性而不更新文档
- 跳过 QA 集成测试场景
