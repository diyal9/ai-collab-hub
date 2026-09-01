# 🚀 AI Collab Hub - 生产级演进路线图 (Roadmap to Production)

> **当前状态**: MVP (Minimum Viable Product) - 核心 DAG 编排引擎已上线，支持并发、崩溃恢复与实时反馈。
> **目标**: 构建高可用、高安全、可观测的企业级 AI 协作平台。

---

## 🟢 Phase 1: 稳定性加固 (Resilience)
**优先级**: 🔥🔥🔥 (立即执行)
**目标**: 防止网络抖动和外部服务异常导致 Flow 崩溃。

### 1.1 节点级重试策略 (Retry Policies)
- **问题**: 当前 Webhook/Agent 失败即终止 Flow。
- **方案**:
  - 引入指数退避重试 (1s, 2s, 4s...)。
  - 针对 5xx/Timeout 错误自动重试 3 次。
  - 针对 4xx 错误直接失败 (业务逻辑错误不重试)。

### 1.2 优雅停机 (Graceful Shutdown)
- **问题**: `kill` 进程导致 Goroutine 丢失状态。
- **方案**:
  - 捕获 `SIGTERM/SIGINT`。
  - 停止接收新 Flow -> 等待活跃 Flow 到达安全点 -> 强制超时退出。
  - 确保 DB 状态与内存一致。

### 1.3 超时熔断 (Timeouts & Circuit Breakers)
- **方案**:
  - 每个节点支持独立 `timeout` 配置。
  - 连续 N 次失败触发熔断，暂停该节点一段时间。

---

## 🔵 Phase 2: 可观测性 (Observability)
**优先级**: 🔥🔥🔥
**目标**: 出了错能秒级定位，告别翻日志。

### 2.1 结构化日志 (Structured Logging)
- **现状**: `log.Printf` 纯文本，难以检索。
- **方案**:
  - 接入 `slog` 或 `zap`。
  - 输出 JSON: `{"level":"error", "trace_id":"...", "exec_id":123, "msg":"node failed"}`。
  - 预留对接 Loki/ELK。

### 2.2 核心指标监控 (Metrics)
- **方案**: 暴露 `/metrics` (Prometheus 格式)。
- **关键指标**:
  - `flow_total` (执行总数)
  - `flow_duration_seconds` (耗时分布)
  - `node_error_rate` (错误率)
  - `active_flows` (当前活跃数)

### 2.3 全链路追踪 (Tracing)
- **方案**: 
  - 生成 `Trace-ID` 并注入 Webhook Header。
  - 支持下游服务 (Bridge/Sandbox) 关联日志。

---

## 🟠 Phase 3: 安全性与权限 (Security)
**优先级**: 🔥🔥
**目标**: 防止恶意执行和数据泄露。

### 3.1 审批 Token 安全 (JWT)
- **现状**: 简单 ID+Timestamp，易伪造。
- **方案**: 
  - 改用 JWT (HS256/RS256) 签发审批链接。
  - 包含 `Exp` (过期时间) 和签名校验。

### 3.2 密钥管理 (Secrets Management)
- **现状**: API Key 明文在 `config.yaml`。
- **方案**:
  - 支持环境变量注入 (`${LLM_API_KEY}`)。
  - 生产环境接入 Vault 或 K8s Secrets。

### 3.3 沙箱隔离加强
- **方案**:
  - Code 节点限制高危命令 (如 `rm -rf /`)。
  - 网络层隔离 (禁止 Sandbox 访问内网敏感 IP)。

---

## 🟣 Phase 4: 扩展性架构 (Scalability)
**优先级**: 🔥
**目标**: 支持水平扩容，应对高并发。

### 4.1 去中心化调度 (Redis/MQ)
- **现状**: `FlowManager` 是纯内存 Map，单机瓶颈。
- **方案**:
  - 引入 Redis 存储 `FlowContext`。
  - 使用 NATS/RabbitMQ 作为任务队列，Hub 变为无状态 Worker。

### 4.2 数据库迁移 (DB Migrations)
- **现状**: `AutoMigrate` (生产环境大忌)。
- **方案**:
  - 引入 `goose` 或 `golang-migrate` 管理版本化 SQL。

---

## ⚫ Phase 5: DevOps & 交付
**优先级**: 🔥
**目标**: 标准化发布流程。

### 5.1 CI/CD Pipeline
- GitHub Actions: `go test`, `golangci-lint`, Docker Build & Push。

### 5.2 Docker Compose / K8s
- 提供一键拉起脚本 (Hub + DB + Redis + Nginx)。

---

## 📝 更新日志

| 日期 | 版本 | 说明 |
|------|------|------|
| 2026-06-06 | v1.0.0 | 初始路线图发布。MVP 核心引擎 (并发+崩溃恢复) 已上线。 |
