---
type: Decision
title: Adopt Superpowers Workflow
description: Virtual team uses obra/superpowers skills as mandatory development pipeline.
tags: [decision, superpowers, workflow]
timestamp: 2026-06-24T00:00:00Z
---

# Context

AI Collab Hub 需要可重复、可审计的虚拟团队协作流程。

# Decision

采用 [Superpowers](https://github.com/obra/superpowers) 作为标准工作流：

1. brainstorming
2. writing-plans
3. executing-plans / subagent-driven-development
4. test-driven-development
5. requesting-code-review / receiving-code-review

# Consequences

- 任务启动必须检查匹配 Skill
- 质量门禁与 Superpowers 清单对齐
- 新 Skill 遵循 Superpowers writing-skills 规范

# Related

- [Agent Team Charter](/concepts/agent-team-charter.md)
- [Team Skills Registry](/processes/team-skills.md)
