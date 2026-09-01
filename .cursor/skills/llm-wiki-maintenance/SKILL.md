---
name: llm-wiki-maintenance
description: Use when ingesting sources into wiki, answering questions from wiki, linting knowledge contradictions, or updating OKF knowledge bundle for AI Collab Hub.
---

# LLM Wiki 维护

## 三层（位于 `brain/`）

| 层 | 路径 | 规则 |
|----|------|------|
| 原始资料 | `brain/raw/` | 只读 |
| 领域百科 | `brain/wiki/` | LLM 维护 |
| 长期知识 | `brain/knowledge/` | OKF v0.1 |
| 工作记忆 | `brain/memory/` | 会话/迭代 |

## 操作

- **Ingest** — 读 `brain/raw/` → 更新 `brain/wiki/` + `brain/knowledge/`
- **Query** — 先读 `brain/wiki/index.md` 或 `brain/knowledge/index.md`
- **Lint** — 矛盾、孤儿页、断链、过期声明

## OKF Frontmatter

必填 `type`；推荐 `title`, `description`, `tags`, `timestamp`
