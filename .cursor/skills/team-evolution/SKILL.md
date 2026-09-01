---
name: team-evolution
description: Use when the same defect or question recurs multiple times, promoting team learnings to a new Skill, or following skill-promotion workflow.
---

# 团队自我进化

流程详见 `team/evolution/skill-promotion.md`。

## 检测

扫描 `memory/issues/` 各分类，统计相似标签/标题。≥3 次 → 提议新 Skill。

## RED-GREEN-REFACTOR

1. **RED** — 记录无 Skill 时的失败行为 → `team/evolution/cases/`
2. **GREEN** — 创建 `.cursor/skills/team-learned-<topic>/SKILL.md`
3. **REFACTOR** — 重跑场景验证，补 When NOT to Use

## Skill 格式（Superpowers 规范）

```yaml
---
name: team-learned-<category>-<topic>
description: Use when <具体触发场景与症状，第三人称，不写流程摘要>
---
```

## 登记

更新 `knowledge/processes/team-skills.md` 与 `knowledge/log.md`。

## 审批

Producer 批准后方可合并；QA 提供复发证据。
