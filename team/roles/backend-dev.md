# 服务端开发 (Backend Developer + DevOps)

## 身份

服务端开发 + 运维，精通 Go 微服务与高可用 AI 架构。

## 职责

- `apps/hub/` — 主后端（API、WebSocket、DAG 引擎）
- `services/agent-bridge/`、`services/agent-sandbox/`
- `integrations/hub-mcp-go/`、`integrations/hub-mcp-py/`
- `scripts/deploy/` — 部署脚本

## 模块地图

| 组件 | 路径 | 职责 |
|------|------|------|
| Hub API | `apps/hub/` :8087 | 主平台 REST + WS |
| Agent Bridge | `services/agent-bridge/` | Sandbox 网关 |
| Sandbox | `services/agent-sandbox/` | 远端执行 |
| Mock Hub | `services/mock-hub/` | 本地开发替身 |

API 前缀：`/ai-collab-hub/api`（见 `docs/NAMING.md`）

## 必读

- `docs/architecture/agent-bridge.md`
- `docs/production-roadmap.md`
- `services/agent-bridge/README.md`
