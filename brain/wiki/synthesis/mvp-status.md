# MVP 状态综合

> 截至 2026-06，基于 `docs/production-roadmap.md` 与代码库分析。

## 已具备

- DAG 编排引擎（并发、崩溃恢复、实时反馈）
- 多 Agent WebSocket 注册与任务下发
- 审批流设计
- 知识中枢 UI + 记忆系统 UI
- MCP 多语言接入（Go/Python/Node）
- SkillHub CLI

## 主要缺口（路线图）

| 阶段 | 缺口 |
|------|------|
| Phase 1 | 节点重试、优雅停机、熔断 |
| Phase 2 | 结构化日志、Prometheus、Tracing |
| Phase 3 | JWT 审批、密钥管理、沙箱加固 |
| Phase 4 | Redis/MQ 调度、DB Migration |
| Phase 5 | CI/CD、Docker/K8s |

## 架构注意

- 主 Hub 后端源码可能不在本仓库
- FlowManager 当前为内存 Map，单机瓶颈

## 引用

- [平台总览](../concepts/platform-overview.md)
- OKF: `/knowledge/synthesis/production-readiness.md`
