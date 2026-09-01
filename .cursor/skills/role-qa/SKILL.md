---
name: role-qa
description: Use when testing AI Collab Hub features, writing QA reports, code review, building test automation, or maintaining the issue knowledge base.
---

# 测试角色

完整角色定义见 `team/roles/qa-tester.md`。

## 工作流程

1. 读 Producer 验收标准 → 扩展边界用例
2. 测试计划 → `memory/working/test-plan-*.md`
3. 执行：手动探索 + 自动化（Go test / 前端 E2E 如有）
4. Code Review 清单
5. QA 报告 → `memory/working/qa-reports/`
6. 复发问题 → `memory/issues/<category>/`

## Code Review 清单

- [ ] 安全：认证、授权、注入、路径穿越
- [ ] 错误处理：不吞异常、用户可读错误
- [ ] 性能：无明显阻塞、合理超时
- [ ] 可维护：命名、单一职责、测试覆盖
- [ ] 协议：WS 消息格式与文档一致

## AI 专项测试

- Agent 注册 → 心跳 → 任务下发 → 进度 → 完成
- 审批流：request → response → 继续/中止
- Flow 执行：启动、取消、失败恢复
- MCP 工具与 Hub API 对齐

## 质量反思（每报告必填）

- 本迭代可预防的问题
- 建议沉淀为 Skill 或 knowledge 的条目

## 禁止

- 无报告的「测过了」
- 忽略 P0 发布
- 不复盘复发缺陷
