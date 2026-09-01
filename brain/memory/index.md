# 团队记忆索引

记忆分层：会话（临时）→ 工作（迭代）→ 问题（分类汇总）→ 知识库（长期）。

## 目录

| 路径 | 类型 | 保留策略 |
|------|------|----------|
| `sessions/` | 会话记忆 | 短期，可归档 |
| `working/` | 工作记忆 | 当前迭代 |
| `issues/` | 问题汇总 | 至解决并沉淀 |
| `../wiki/` + `../knowledge/` | 长期记忆 | 永久 |

## 当前工作项

见 [working/index.md](working/index.md)

## 问题库

- [frontend/](issues/frontend/)
- [backend/](issues/backend/)
- [product/](issues/product/)
- [qa/](issues/qa/)
- [infra/](issues/infra/)

## 沉淀规则

1. 会话结束 → 写 `sessions/YYYY-MM-DD-<topic>.md`
2. 重复问题 → `issues/<category>/`
3. 确认长期价值 → `llm-wiki-maintenance` 升格至 `wiki/` 或 `knowledge/`

## 会话摘要模板

```markdown
# Session: <topic> — <date>
## Participants (roles)
## Decisions
## Open questions
## Next actions
## Promote to wiki/knowledge?
```
