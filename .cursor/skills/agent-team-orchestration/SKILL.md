---
name: agent-team-orchestration
description: Use when starting multi-role work on AI Collab Hub, coordinating Producer/Frontend/Backend/QA virtual team, or when user mentions Agent Team, virtual team development, or Superpowers workflow.
---

# Agent Team 编排

## 核心原则

虚拟团队按 Superpowers 流水线协作。编排者（主 Agent）负责角色切换、上下文传递与质量门禁。

## 启动清单

1. 读取 `AGENTS.md` 与 `team/charter.md`
2. 检查 Superpowers `using-superpowers` — 匹配则必须遵循
3. 根据任务类型加载角色 Skill：`role-producer` | `role-frontend` | `role-backend` | `role-qa`
4. 检查 `memory/working/` 是否有进行中任务

## 任务分类与路由

| 任务类型 | 主责 Skill | 流程 |
|----------|------------|------|
| 新功能/改版 | role-producer | brainstorming → writing-plans → 分派 |
| UI/页面 | role-frontend | frontend-design + Taste → 实现 → QA |
| API/服务/部署 | role-backend | 契约 → TDD → 集成测试 |
| 测试/Review | role-qa | 测试计划 → 执行 → 报告 |
| 知识沉淀 | llm-wiki-maintenance | ingest / query / lint |
| 经验升格 | team-evolution | skill-promotion 流程 |

## 多角色执行模式

### 模式 A：串行（小任务）

Producer 定标准 → 开发角色实现 → QA 验收 → 沉淀记忆

### 模式 B：子 Agent 并行（大任务）

参照 Superpowers `subagent-driven-development`：

1. `writing-plans` 拆分为独立任务项
2. 每项派发子 Agent，附带：角色 Skill 路径、验收标准、相关 `wiki/` 页面
3. 汇总后 QA 整体验收

## 会话结束（强制）

1. 写 `memory/sessions/YYYY-MM-DD-<topic>.md`
2. 更新 `memory/working/index.md` 任务状态
3. 非平凡决策写入 `wiki/` 或 `knowledge/`
4. 追加 `wiki/log.md` 条目

## 质量门禁

合并或宣告完成前确认：

- [ ] Producer 验收标准已对照
- [ ] QA 报告存在或明确记录豁免
- [ ] 无未解决的 P0/P1
- [ ] 知识库已更新（如适用）
