# AI Collab Hub

企业级 AI 协作与多 Agent 编排平台。

## 目录结构

```
ai-collab-hub/
├── AGENTS.md                 # Agent 团队总纲（会话入口）
├── README.md                 # 本文件
├── config.yaml               # 部署参考配置（见 apps/hub/config.yaml）
│
├── apps/                     # 主应用
│   ├── hub/                  # Go 后端（API + WebSocket + DAG 引擎）
│   └── web/                  # Vue 3 前端
│
├── services/                 # 运行时边车服务
│   ├── agent-bridge/         # Agent ↔ Hub WebSocket 桥接
│   ├── agent-sandbox/        # 远端沙箱执行
│   └── mock-hub/             # 本地开发 Mock API
│
├── integrations/             # IDE / 外部工具接入
│   ├── cursor-mcp-bridge/    # Cursor 双向 MCP 桥
│   ├── cursor-cli/           # Cursor Headless CLI 封装
│   ├── hub-mcp-go/           # Hub MCP Server (Go)
│   └── hub-mcp-py/           # Hub MCP Server (Python)
│
├── packages/                 # 可分发工具包
│   ├── skillhub-cli/         # 技能安装 CLI
│   └── skillhub-registry/    # 内置技能注册表
│
├── team/                     # 虚拟 Agent 团队定义
├── brain/                    # 知识基础设施
│   ├── wiki/                 # LLM Wiki（Karpathy 模式）
│   ├── knowledge/            # OKF v0.1 长期知识库
│   ├── memory/               # 工作/会话记忆
│   └── raw/                  # 只读原始资料
│
├── docs/                     # 项目文档
├── scripts/                  # 开发与部署脚本
├── releases/                 # 预编译发布物
└── .cursor/skills/           # Cursor Agent Skills
```

## 快速开始

```bash
# 安装依赖
bash scripts/setup-dev.sh

# 流水线验收
bash scripts/verify-pipeline.sh

# 启动 mock-hub + 前端
bash scripts/start-dev.sh
```

生产 Hub：

```bash
cd apps/hub && go run .
cd apps/web && npm run dev
```

## 命名说明

URL 前缀 `/ai-collab-hub` 说明见 [docs/NAMING.md](docs/NAMING.md)。

## 文档

- [架构：Agent Bridge](docs/architecture/agent-bridge.md)
- [SkillHub](docs/skillhub.md)
- [生产路线图](docs/production-roadmap.md)
- [Agent Team](team/README.md)
