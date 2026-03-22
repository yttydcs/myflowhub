# MyFlowHub3 Documentation

`docs/` 是 MyFlowHub3 meta workspace 的受治理文档入口。

## Reading Order
- 先看 [../plan.md](../plan.md)
  - 了解当前全局状态、活跃事项与控制面入口
- 再看 [../repos.md](../repos.md)
  - 了解仓库职责、依赖边界与交接关系
- 再按文档类别进入：
  - [requirements/README.md](requirements/README.md)
  - [specs/README.md](specs/README.md)
  - [plan/README.md](plan/README.md)
  - [change/README.md](change/README.md)
  - [lessons/README.md](lessons/README.md)

## Sections
- [requirements/README.md](requirements/README.md)
  - 长期需求、范围与验收口径
- [specs/README.md](specs/README.md)
  - 技术约束、协议映射、跨仓稳定入口
- [plan/README.md](plan/README.md)
  - 历史 workflow 的完整计划正文、Checklist 与约束
- [change/README.md](change/README.md)
  - 已完成变更的结果归档、验证方式、影响与回滚
- [lessons/README.md](lessons/README.md)
  - 可复用的复盘、陷阱与防错经验

## Quick Entry
- 想看当前主线与整体状态：
  - [../plan.md](../plan.md)
  - [../repos.md](../repos.md)
- 想查跨仓协议映射速查表：
  - [specs/protocol_map.md](specs/protocol_map.md)
- 想看当前 Server 的长期协议规范：
  - [../repo/MyFlowHub-Server/docs/specs/README.md](../repo/MyFlowHub-Server/docs/specs/README.md)
- 想看某次 workflow 的计划与执行证据：
  - [plan/README.md](plan/README.md)
  - [change/README.md](change/README.md)

## Historical Note
- `target.md`
  - 当前工作区根目录中不存在该文件。
  - 历史归档、旧计划与旧变更文档中仍可能引用它，应理解为历史上下文，而不是当前入口。

## Maintenance Rules
- 新增 `docs/plan/*.md` 时，同步更新 [plan/README.md](plan/README.md)
- 新增 `docs/change/*.md` 时，同步更新 [change/README.md](change/README.md)
- 调整根级入口或跨仓路由时，同步更新 [../plan.md](../plan.md)、[../repos.md](../repos.md) 和本文件
