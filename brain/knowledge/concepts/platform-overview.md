---
type: Concept
title: AI Collab Hub Platform Overview
description: Enterprise AI collaboration and multi-agent orchestration platform.
tags: [platform, ai-collab-hub, architecture]
timestamp: 2026-06-24T00:00:00Z
resource: /ai-collab-hub
---

# Overview

AI Collab Hub 统一管理 Hermes、Codex、Cursor 等 Agent，提供 DAG 编排、审批流、知识中枢与 SkillHub。

# Core Modules

| Module | Path | Role |
|--------|------|------|
| Frontend | `apps/web/` | Web UI |
| Agent Bridge | `services/agent-bridge/` | WS gateway client |
| Sandbox | `services/agent-sandbox/` | Remote execution |
| MCP | `integrations/hub-mcp-go/`, `integrations/hub-mcp-py/` | IDE integration |

# Related

- Wiki synthesis: [/wiki/synthesis/mvp-status.md](/wiki/synthesis/mvp-status.md)
- [Agent Team Charter](/concepts/agent-team-charter.md)

# Citations

[1] [Production Roadmap](../../docs/production-roadmap.md)
