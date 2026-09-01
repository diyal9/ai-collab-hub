# 项目命名说明

## 当前标准

| 名称 | 用途 |
|------|------|
| **AI Collab Hub** | 产品名 |
| **ai-collab-hub** | 仓库名、URL 路径前缀、部署目录名 |
| **hermes** | Agent 类型名（`type: hermes`），与 URL 无关 |

## URL 前缀

所有 Web 与 API 使用 **`/ai-collab-hub`**：

| 位置 | 值 |
|------|-----|
| 前端 base | `/ai-collab-hub/` |
| API | `/ai-collab-hub/api` |
| `config.yaml` | `public_url: .../ai-collab-hub` |

Hub 后端通过中间件剥离 `/ai-collab-hub` 前缀，内部路由仍为 `/api/*`。

## 历史：`hermesshare`

旧版部署使用 **HermesShare** 作为产品名，URL 前缀为 `/hermesshare`，服务器目录为 `/root/aispace/hermesshare/`。

2026-06 起统一改为 **ai-collab-hub**。`brain/raw/` 中归档资料可能仍含旧路径，仅供参考。

## `hermes` 与 `ai-collab-hub`

- **hermes** — Agent 产品线名称；技能目录 `~/.hermes/skills`；npm 包 `@hermes/*`
- **ai-collab-hub** — 本协作平台的路径与仓库名

二者不要混用。

## 生产迁移（从 hermesshare）

若线上仍为旧前缀，需同步修改：

1. Nginx `location /ai-collab-hub/`
2. 前端构建（已内置新 base）
3. 服务器目录可保留或迁至 `/root/aispace/ai-collab-hub/`
