# Todo — 按深度拓扑发现

## Current Status

- Stage：$m-archive complete；本地master已合入，本轮工作树/分支已清理，未推送。
- [变更与收口状态](docs/change/2026-09-06_depth-scoped-topology-discovery.md)
- [计划快照](docs/plan/plan_archive_2026-09-06_depth-scoped-topology.md) / [验收清单快照](docs/plan/todo_archive_2026-09-06_depth-scoped-topology.md)

## 已验收

- [x] DOC01 — 稳定文档与模型边界。
- [x] PROTO01 — 数值depth与严格协议校验。
- [x] TREE01 — 原子拓扑快照。
- [x] MGMT01 — children/subtree与有界缓存、订阅兼容。
- [x] SDK01 — attached查询接口与生成契约。
- [x] DESK01 — 按需树/目录与View异步保护。
- [x] QA01 — Go/race、224项前端测试、真实Wails/TCP与当前Windows产品验证。
- [x] ARC01 — 文档归档、本地合入、本轮工作树和分支清理完成；用户草稿及其他工作树保留。

## 未授权的后续事项

- [ ] LIVE02 — 实时拓扑推送与一致性/恢复契约。
- [ ] SEARCH02 — 服务端搜索、分页和超大树。
- [ ] AUTHZ02 — payload/member权限与effective discovery。
- [ ] HOST02 — 全NodeHost拓扑provider装配与Hub宿主迁移。
- [ ] ANDROID-REDESIGN — [Android重新设计](docs/requirements/mobile-embedded-redesign.md)。
- [ ] EMBEDDED-REDESIGN — [嵌入式重新设计](docs/requirements/mobile-embedded-redesign.md)。
- [ ] FLOW-REDESIGN — [通用自动化模型重做](docs/requirements/flow-redesign.md)。
- [ ] PUB01 — 推送、发布、签名与部署，需独立授权。
