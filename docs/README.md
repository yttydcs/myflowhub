# MyFlowHub3 文档入口

`docs/` 是 MyFlowHub3 meta workspace 的文档汇总目录。

## 建议阅读顺序
- 先看 [../plan.md](../plan.md)
  - 了解当前全局状态、文档分工、主要入口
- 再看 [../repos.md](../repos.md)
  - 了解仓库职责、依赖边界、长期路线图
- 再按需求进入下列专题

## 当前可用入口
- [change/README.md](change/README.md)
  - 已完成变更的结果归档、验证方式、影响与回滚
- [plan_archive/README.md](plan_archive/README.md)
  - 历史 workflow 的完整计划正文、Checklist、约束与审查线索
- [protocol_map.md](protocol_map.md)
  - 协议映射速查表
- [../guide.md](../guide.md)
  - 当前工作区规范
- [../repos.md](../repos.md)
  - 仓库职责、依赖边界与接手说明
- [../plan.md](../plan.md)
  - 当前全局索引

## 按问题进入
- 想知道“现在项目整体怎么看”：
  - 看 [../plan.md](../plan.md) 和 [../repos.md](../repos.md)
- 想知道“某个功能什么时候改过、结果是什么”：
  - 看 [change/README.md](change/README.md)
- 想知道“当时怎么拆任务、有哪些边界和验收条件”：
  - 看 [plan_archive/README.md](plan_archive/README.md)
- 想查协议 action / payload：
  - 看 [protocol_map.md](protocol_map.md)
- 想看当前子协议规范文档：
  - 看 `repo/MyFlowHub-Server/docs`

## 历史引用说明
- `target.md`
  - 当前工作区根目录中不存在该文件。
  - 历史归档、旧计划与旧变更文档中仍可能引用它，应理解为历史上下文，而不是当前入口。

## 维护规则
- 新增 `docs/change/*.md` 时，同步更新 [change/README.md](change/README.md)
- 新增 `docs/plan_archive/*.md` 时，同步更新 [plan_archive/README.md](plan_archive/README.md)
- 若根级文档入口发生变化，同步更新 [../plan.md](../plan.md)、[../repos.md](../repos.md) 和本文件
