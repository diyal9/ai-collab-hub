# QA 报告：Agent Team 流水线首轮验收 — 2026-06-24

## 摘要

完成环境安装、Mock Hub 搭建、Dashboard/Team 页面交付及自动化门禁脚本。流水线 **通过**。

## 测试范围

| 项 | 结果 |
|----|------|
| `scripts/setup-dev.sh` | ✓ |
| `scripts/verify-pipeline.sh` | ✓ |
| mock-hub 单元测试 | ✓ 2/2 |
| agent-bridge 配置解析测试 | ✓ 1/1 |
| frontend `npm run build` | ✓ |
| mock-hub `/health` | ✓ |
| mock-hub `/ai-collab-hub/api/login` | ✓ |
| mock-hub `/ai-collab-hub/api/team/status` | ✓ |
| vite 代理 `/ai-collab-hub/api/*` | ✓ |

## 功能验收

| ID | 级别 | 描述 | 状态 |
|----|------|------|------|
| F001 | - | Dashboard 显示 Agent Team 入口卡片 | pass |
| F002 | - | `/team` 路由展示四角色 | pass |
| F003 | - | 侧栏 Agent Team 导航 | pass |
| F004 | P3 | 生产 Hub 未接入，部分页面仍为空数据 | known |

## Code Review 发现

| ID | 级别 | 发现 |
|----|------|------|
| R001 | P3 | `agent-bridge/config.go` init 中 `flag.Parse` 与 `go test` 冲突，已改为 lazy load |
| R002 | P3 | vite 原仅代理 `/api`，已补 `/ai-collab-hub/api` |
| R003 | P3 | 主 Hub 后端不在本仓，dev 依赖 mock-hub |

## 质量反思

- **可预防**：Go 包若在 `init()` 调 `flag.Parse()`，测试必挂 — 已记入 backend 问题库模式
- **沉淀建议**：`team-learned-backend-go-flag-init`（待第 3 次复发后升格）

## 建议沉淀

- [x] `wiki/concepts/agent-team.md` 已存在
- [x] `memory/sessions/2026-06-24-pipeline-run.md`
- [ ] 生产 Hub 接入后替换 mock-hub

## 结论

**批准合并**（开发环境流水线）。生产发布需真实 Hub 后端。
