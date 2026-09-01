---
type: Playbook
title: QA Standards
description: Testing, acceptance, and code review standards for AI Collab Hub.
tags: [qa, testing, quality]
timestamp: 2026-06-24T00:00:00Z
---

# Severity Levels

| Level | Definition | Release |
|-------|------------|---------|
| P0 | Data loss, security breach, core flow broken | Block |
| P1 | Major feature unusable, no workaround | Block |
| P2 | Feature degraded, workaround exists | Risk accept by Producer |
| P3 | Cosmetic, minor UX | Track |

# Required Per Iteration

1. Test plan in `memory/working/test-plan-*.md`
2. QA report in `memory/working/qa-reports/`
3. Code review checklist (see `role-qa` skill)
4. Quality reflection section in report

# AI-Specific Test Areas

- Agent lifecycle: register → heartbeat → task → complete
- Approval flow end-to-end
- Flow DAG: execute, cancel, failure recovery
- MCP tools ↔ Hub API parity

# Automation Goals

- Go: `*_test.go` for bridge, sandbox, hub-mcp
- Frontend: component tests for critical paths (orchestrator save/execute)
- E2E: optional Playwright for login → create flow → execute

# Issue Knowledge Base

Recurring issues → `memory/issues/<category>/` → promote to skill or OKF.
