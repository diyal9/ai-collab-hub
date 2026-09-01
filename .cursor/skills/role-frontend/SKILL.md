---
name: role-frontend
description: Use when implementing or designing Vue UI, Agent Orchestrator canvas, dashboard pages, or frontend UX for AI Collab Hub.
---

# 前端开发角色

完整角色定义见 `team/roles/frontend-dev.md`。

## 强制依赖 Skill

执行 UI 任务前 **必须** 加载：

1. **`frontend-design`** — 选定美学方向并实现
2. **Taste Skill**（若环境已安装）— 交互品味

## 工作流程

1. 读 Producer 验收标准
2. 应用 frontend-design 确定：字体、色板、布局、动效
3. 在 `apps/web/src/` 实现，遵循现有 Vue 3 + Element Plus 约定
4. 自测核心路径 → requesting-code-review
5. 架构变化更新 `wiki/concepts/frontend-architecture.md`

## 技术约束

- baseURL: `/ai-collab-hub/api`
- 路由前缀: `/ai-collab-hub/`
- 编排画布: `@vue-flow/core`

## 禁止

- 默认 AI 审美（Inter、紫色渐变、无个性卡片）
- 未经 Backend 确认擅自改 API 契约
- 跳过 QA 可访问性与错误状态检查
