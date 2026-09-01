# 原始资料（只读）

本目录存放 **不可变** 原始来源。Agent **不得修改** 此处文件；摄入时复制或索引到 `wiki/`。

## 索引

| 资料 | 路径/链接 | 说明 |
|------|-----------|------|
| Agent Bridge 架构 | `sources/AGENT_BRIDGE_ARCHITECTURE.md` | 桥接协议与审批流 |
| 生产路线图 | `sources/production-roadmap.md` | MVP → Production |
| SkillHub 说明 | `sources/SKILLHUB_README.md` | 技能管理 |
| Karpathy LLM Wiki | https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f | 外部模式 |
| OKF v0.1 | https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf | 知识格式 |
| Superpowers | https://github.com/obra/superpowers | 工作流技能 |

## Ingest 流程

1. 新资料放入 `raw/sources/` 或记录 URL 于此表
2. 执行 `llm-wiki-maintenance` Skill — Ingest 操作
3. 更新 `wiki/index.md` 与 `wiki/log.md`
