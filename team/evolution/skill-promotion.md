# Skill 升格流程

当团队反复遇到同类问题时，将经验沉淀为可复用 Skill。

## 触发条件（满足任一）

1. `memory/issues/<category>/` 中同类问题 ≥3 条
2. QA 报告中「建议沉淀」项被 Producer 批准
3. Code Review 中同一类意见出现 ≥2 次

## RED-GREEN-REFACTOR（来自 Superpowers writing-skills）

### RED — 基线

1. 记录当前 Agent 在无 Skill 时的错误行为（压力场景）
2. 保存到 `team/evolution/cases/<topic>-baseline.md`

### GREEN — 写 Skill

1. 创建 `.cursor/skills/team-learned-<topic>/SKILL.md`
2. Frontmatter `description` 以 "Use when..." 开头，只写触发条件
3. 正文：Overview → When to Use → When NOT to Use → Workflow（清单）

### REFACTOR — 验证

1. 用相同压力场景重跑，确认 Agent 遵循 Skill
2. 关闭漏洞（补充 When NOT to Use）
3. 在 `knowledge/processes/team-skills.md` 登记

## 命名规范

```
team-learned-<category>-<short-topic>

示例：
team-learned-frontend-vue-flow-save
team-learned-backend-ws-reconnect
team-learned-qa-agent-lifecycle
```

## 审批

| 步骤 | 审批人 |
|------|--------|
| 提议 | QA 或任一开发者 |
| 批准 | Producer |
| 合并 | 纳入 PR 与 `knowledge/log.md` |

## 退役

Skill 若 90 天未触发 → 移至 `team/evolution/retired/`
