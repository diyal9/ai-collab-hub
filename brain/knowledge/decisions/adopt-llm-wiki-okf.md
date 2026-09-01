---
type: Decision
title: Adopt LLM Wiki + OKF Knowledge Architecture
description: Two-layer knowledge — Karpathy LLM Wiki for synthesis, OKF v0.1 for durable exchangeable facts.
tags: [decision, llm-wiki, okf, knowledge]
timestamp: 2026-06-24T00:00:00Z
---

# Context

团队需要可复利、可版本控制、Agent 可消费的知识基础设施。

# Decision

1. **LLM Wiki**（`wiki/`）— 遵循 [Karpathy LLM Wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f)：`index.md` + `log.md` + 概念交叉引用
2. **OKF Bundle**（`knowledge/`）— 遵循 [OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf)：frontmatter 必填 `type`
3. **Raw**（`raw/`）— 只读原始资料
4. **Memory**（`memory/`）— 会话与工作记忆，经 Lint 后升格

# Operations

- Ingest / Query / Lint — Skill: `llm-wiki-maintenance`

# Related

- [OKF Specification](/references/okf-spec.md)
- [Karpathy LLM Wiki](/references/karpathy-llm-wiki.md)
