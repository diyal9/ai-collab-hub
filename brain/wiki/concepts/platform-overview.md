# AI Collab Hub 平台总览

## 模块（重组后）

| 模块 | 路径 |
|------|------|
| 主后端 | `apps/hub/` |
| 前端 | `apps/web/` |
| Agent Bridge | `services/agent-bridge/` |
| Sandbox | `services/agent-sandbox/` |
| MCP | `integrations/hub-mcp-go/`, `integrations/hub-mcp-py/` |
| 技能 CLI | `packages/skillhub-cli/` |

## 部署

- API: `:8087`，URL 前缀 `/ai-collab-hub`（见 `docs/NAMING.md`）
- 本地开发：`bash scripts/start-dev.sh`

## 相关

- [Agent 桥接](agent-bridge.md)
- [Agent Team](agent-team.md)
