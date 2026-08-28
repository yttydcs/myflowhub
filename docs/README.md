# MyFlowHub Documentation

`docs/` 是 canonical `repo/MyFlowHub` checkout 的受治理文档入口。

## Reading Order
- 先看 [../plan.md](../plan.md)
  - 了解当前全局状态、活跃事项与控制面入口
- 再看 [../repos.md](../repos.md)
  - 了解仓库职责、依赖边界与交接关系
- 需要稳定事实时按类别进入：
  - [intake/README.md](intake/README.md)
  - [features/README.md](features/README.md)
  - [requirements/README.md](requirements/README.md)
  - [specs/README.md](specs/README.md)
  - [decisions/README.md](decisions/README.md)
- 需要 workflow 历史或排障经验时进入：
  - [plan/README.md](plan/README.md)
  - [change/README.md](change/README.md)
  - [lessons/README.md](lessons/README.md)

## Sections
- [intake/README.md](intake/README.md)
  - 原始请求证据、来源上下文与待澄清问题
- [features/README.md](features/README.md)
  - 当前用户可见行为、权限、状态与验收场景
- [requirements/README.md](requirements/README.md)
  - 长期需求、范围与验收口径
- [specs/README.md](specs/README.md)
  - 技术约束、协议映射、跨仓稳定入口
- [decisions/README.md](decisions/README.md)
  - 重要架构选择、替代方案、后果与取代关系
- [plan/README.md](plan/README.md)
  - 历史 workflow 的完整计划正文、Checklist 与约束
- [change/README.md](change/README.md)
  - 已完成变更的结果归档、验证方式、影响与回滚
- [lessons/README.md](lessons/README.md)
  - 可复用的复盘、陷阱与防错经验

## Troubleshooting Order
- 先从 [lessons/README.md](lessons/README.md) 按症状、关键词和触发条件查找。
- lesson 不足时再查 [change/README.md](change/README.md) 的实施历史。
- 准备修改行为前，回到 `features/`、`requirements/` 和 `specs/` 确认当前稳定事实。

## Private Docs Boundary
- `docs/` 是当前 canonical repository 的 governed docs root。
- 工作区级本机 SDK、附件和 sibling worktree 不属于本目录的治理范围。
- remote、push、publication 与 backup 目标由用户单独决定；整理文档不隐含发布授权。

## Quick Entry
- 想看当前主线与整体状态：
  - [../plan.md](../plan.md)
  - [../repos.md](../repos.md)
- 想查 vNext 协议与资源映射速查表：
  - [specs/protocol_map.md](specs/protocol_map.md)
- 想追溯已退役旧仓的来源、提交与迁移结果：
  - [../migration/README.md](../migration/README.md)
  - [../migration/sources.yaml](../migration/sources.yaml)
- 想看某次 workflow 的计划与执行证据：
  - [plan/README.md](plan/README.md)
  - [change/README.md](change/README.md)

## Historical Note
- `target.md`
  - 当前工作区根目录中不存在该文件。
  - 历史归档、旧计划与旧变更文档中仍可能引用它，应理解为历史上下文，而不是当前入口。

## Maintenance Rules
- 新增 `docs/intake/*.md`、`docs/features/*.md` 或 `docs/decisions/*.md` 时，同步更新对应分类索引
- 新增 `docs/plan/*.md` 时，同步更新 [plan/README.md](plan/README.md)
- 新增 `docs/change/*.md` 时，同步更新 [change/README.md](change/README.md)
- 新增可复用排障经验时同步更新 [lessons/README.md](lessons/README.md)，不要只留在 `change/`
- 更新 [specs/protocol_map.md](specs/protocol_map.md) 时，以当前 monorepo 的 `protocol/` schema 和生成门禁为准，不再回退到旧 Proto 仓库
- 调整根级入口或跨仓路由时，同步更新 [../plan.md](../plan.md)、[../repos.md](../repos.md) 和本文件
