# AI Collab Hub — Agent Team 总纲

本文件是虚拟开发团队的 **Schema**。会话启动时应读取本文件。

## 仓库布局

见根目录 [README.md](README.md)。核心路径：

| 路径 | 内容 |
|------|------|
| `apps/hub/` | Go 主后端 |
| `apps/web/` | Vue 前端 |
| `services/` | agent-bridge、sandbox、mock-hub |
| `integrations/` | MCP 与 Cursor 桥接 |
| `packages/` | skillhub-cli、registry |
| `team/` | 虚拟团队角色与宪章 |
| `brain/` | wiki、knowledge、memory、raw |

命名说明：[docs/NAMING.md](docs/NAMING.md)

## 团队使命

以虚拟 Agent Team 持续演进 **AI Collab Hub**。

## 角色矩阵

| 角色 | Skill | 职责 |
|------|-------|------|
| 制作人 | `role-producer` | 产品、UX、Multi-Agent 设计 |
| 前端 | `role-frontend` | UI/交互（frontend-design + Taste） |
| 服务端 | `role-backend` | Go 服务、运维 |
| 测试 | `role-qa` | 验收、Review、自动化 |

编排：`agent-team-orchestration`

## 工作流 — Superpowers

遵循 [Superpowers](https://github.com/obra/superpowers)：`brainstorming` → `writing-plans` → `executing` → `TDD` → `code-review`

## 知识三层

```
brain/raw/       → 只读原始资料
brain/wiki/      → LLM Wiki
brain/knowledge/ → OKF v0.1
brain/memory/    → 工作/会话记忆
```

维护 Skill：`llm-wiki-maintenance`

## 自我进化

见 `team/evolution/skill-promotion.md`，Skill：`team-evolution`

## 导航

- 宪章：`team/charter.md`
- 角色：`team/roles/`
- Wiki：`brain/wiki/index.md`
- 知识库：`brain/knowledge/index.md`
- 记忆：`brain/memory/index.md`

## 配置速览

- Hub API：`:8087`，公开路径 `/ai-collab-hub`
- 主配置：`apps/hub/config.yaml`
