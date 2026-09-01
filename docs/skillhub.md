# SkillHub 技能管理系统

## 架构概览

```
┌─────────────────────────────────────────────────────────┐
│                    前端界面 (Vue 3)                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐  │
│  │ npx 安装  │  │ 上传技能  │  │ 手动创建/编辑         │  │
│  └────┬─────┘  └────┬─────┘  └──────────┬───────────┘  │
└───────┼──────────────┼───────────────────┼──────────────┘
        │              │                   │
        ▼              ▼                   ▼
┌─────────────────────────────────────────────────────────┐
│              后端 API (Go/Gin)                           │
│  /api/skills        /api/skills/upload                   │
│  /api/skills/:id    /api/skills/:id/toggle               │
└───────────────────────┬─────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────────┐
│ skillhub-cli │ │ 技能注册表    │ │ 本地技能目录      │
│ (npx 工具)   │ │ (registry)   │ │ ~/.hermes/skills/ │
└──────────────┘ └──────────────┘ └──────────────────┘
```

## 快速开始

### 1. 使用 CLI 工具

```bash
# 进入 CLI 目录
cd ~/aispace/ai-collab-hub/skillhub-cli

# 搜索技能包
node bin/skillhub.js search cocos

# 安装技能包
node bin/skillhub.js install cocos-ui-workflow

# 列出已安装技能
node bin/skillhub.js list

# 卸载技能包
node bin/skillhub.js rm cocos-ui-workflow
```

### 2. 前端界面操作

1. 登录 AI 协作平台
2. 进入"技能管理"页面
3. 点击"📦 npx 安装"按钮
4. 输入技能包名称（如: `cocos-ui-workflow`）
5. 点击安装

### 3. 添加自定义技能源

```bash
# 创建本地注册表
mkdir -p ~/my-skills

# 复制技能包到注册表
cp -r /path/to/my-skill ~/my-skills/

# 从自定义源安装
node bin/skillhub.js install my-skill -s ~/my-skills
```

## 技能包结构

```
skill-name/
├── SKILL.md              # 技能描述文件（必需）
├── references/           # 参考文档（可选）
│   └── api-guide.md
├── templates/            # 模板文件（可选）
│   └── config.yaml
└── scripts/              # 脚本文件（可选）
    └── validate.py
```

### SKILL.md 格式

```yaml
---
name: skill-name
category: gaming
description: 简短描述技能功能
trigger: 触发关键词
version: 1.0.0
---

# 技能名称

## 使用场景
- 场景 1
- 场景 2

## 使用步骤
1. 步骤 1
2. 步骤 2

## 配置要求
- 要求 1
- 要求 2
```

## 可用技能包

| 技能包 | 分类 | 描述 |
|:---|:---|:---|
| `cocos-ui-workflow` | gaming | Cocos Creator UI 工作流 |
| `ai-gateway-selection` | mlops | AI 网关选择指南 |
| `digital-human-experience` | web-3d | 3D 数字人体验优化 |

## 扩展技能源

### 方式 1: 本地目录

```bash
# 任何包含 SKILL.md 的目录都可以作为技能源
node bin/skillhub.js install skill-name -s /path/to/skills
```

### 方式 2: Git 仓库

```bash
# 克隆仓库作为技能源
git clone https://github.com/your-org/skillhub-registry.git ~/skillhub-registry
node bin/skillhub.js install skill-name -s ~/skillhub-registry
```

### 方式 3: npm 包（未来支持）

```bash
# 未来支持从 npm 安装
npx @hermes/skillhub install skill-name
```

## 最佳实践

1. **版本管理**: 在 SKILL.md 中明确标注版本号
2. **分类清晰**: 使用标准分类（gaming, mlops, devops 等）
3. **触发词准确**: 设置合理的 trigger 关键词
4. **文档完整**: 包含使用场景、步骤、配置要求
5. **定期更新**: 使用 `update-all` 命令更新所有技能

## 故障排除

### 技能包未找到
- 检查技能源路径是否正确
- 使用 `search` 命令确认技能包存在

### 安装失败
- 检查 `~/.hermes/skills/` 目录权限
- 确认技能包结构正确（必须有 SKILL.md）

### 技能未生效
- 重启 Hermes Agent
- 检查技能是否启用（前端界面开关）
