# Session: Agent Team Bootstrap — 2026-06-24

## Participants (roles)

- Producer — 定义团队结构与知识架构
- Orchestrator — 创建文件与 Skill 体系

## Decisions

1. 采用 Superpowers 作为标准开发流水线
2. 知识双层：`wiki/`（LLM 百科）+ `knowledge/`（OKF 长期）
3. 四角色：Producer / Frontend / Backend / QA
4. 自我进化：重复问题 → `team-learned-*` Skill

## Deliverables

- `AGENTS.md` — 团队 Schema
- `agent-team/` — 宪章与角色定义
- `.cursor/skills/` — 7 个团队 Skill
- `wiki/`, `knowledge/`, `memory/` — 知识基础设施

## Open questions

- 主 Hub 后端源码仓库位置（待 Backend 确认）
- Taste Skill 安装路径（待 Frontend 配置）

## Next actions

- [ ] 安装 Superpowers 插件或复制 skills 到本地
- [ ] 用真实功能迭代跑通一次完整流水线
- [ ] QA 建立首批自动化测试

## Promote to wiki/knowledge?

- Done: `wiki/concepts/agent-team.md`, `knowledge/concepts/agent-team-charter.md`
