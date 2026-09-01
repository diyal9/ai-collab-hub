---
type: Metric
title: Production Readiness Synthesis
description: Gap analysis between MVP and production-grade platform.
tags: [synthesis, roadmap, production]
timestamp: 2026-06-24T00:00:00Z
---

# Current State

MVP: DAG engine, agent WS, approval design, knowledge UI, MCP bridges.

# Gaps (from roadmap)

| Phase | Focus |
|-------|-------|
| 1 Resilience | Retry, graceful shutdown, circuit breaker |
| 2 Observability | slog/zap, Prometheus, tracing |
| 3 Security | JWT approval, secrets, sandbox hardening |
| 4 Scalability | Redis/MQ, DB migrations |
| 5 DevOps | CI/CD, Docker/K8s |

# Architectural Risks

- Hub backend may live outside this repo
- In-memory FlowManager = single-node limit

# Related

- Wiki: [/wiki/synthesis/mvp-status.md](/wiki/synthesis/mvp-status.md)
- Raw: `docs/production-roadmap.md`
