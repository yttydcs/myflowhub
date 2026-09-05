# Plan — 按深度拓扑发现归档入口

- Current Stage：$m-archive；文档归档完成，本地合入和清理待执行。
- Canonical Repo：D:/project/MyFlowHub3/repo/MyFlowHub。
- Docs Root：canonical docs/，不另设文档仓库。
- 实现：a3757f8de753dcbb67a615ed3c7ae58b3a4ac2f8。
- 执行范围：DOC01、PROTO01、TREE01、MGMT01、SDK01、DESK01、QA01，均验收通过。
- 归档授权：用户显式调用 $m-archive，包含本地提交、合入和本轮工作树清理；未授权推送或发布。

## 当前入口

- [变更与收口状态](docs/change/2026-09-06_depth-scoped-topology-discovery.md)
- [完整计划快照](docs/plan/plan_archive_2026-09-06_depth-scoped-topology.md)
- [完整验收清单快照](docs/plan/todo_archive_2026-09-06_depth-scoped-topology.md)
- [当前查询与缓存合同](docs/specs/topology-discovery.md)
- [Desktop当前行为](docs/features/desktop.md)
- [待办入口](todo.md)

## 后续范围

LIVE02、SEARCH02、AUTHZ02、HOST02 仍为独立后续设计。Android、Embedded、Flow重新设计继续保留，不恢复退役实现。没有新的实施任务自动获得批准。
