# 2026-08-27 vNext 全量迁移与最终切换

## Source

- 来源：用户在统一节点运行时第一阶段归档后，先要求继续规划重构，随后确认希望一次性完成全部迁移，并显式调用 `$m-plan`。
- 日期：2026-08-27。

## Request Text / Source-preserving Summary

- 不希望新核心长期停留在演示或只迁移一个节点的状态。
- 希望在没有外部兼容负担的前提下，一次性完成全部第一方代码与产品迁移。
- “一次性”指一个完整迁移 workflow 和最终一次切换，不要求把所有改动压缩成一个不可审查的提交。
- 迁移必须继续遵守已经接受的统一节点树、父控子权限、一级 Subscription/Command、可插拔链路和单一 canonical monorepo。
- 新系统不保留长期旧 wire、旧 module path 或 SubProto compatibility bridge。

## Context

第一阶段已经建立 `MFH3` wire、统一 Node runtime、Memory/TCP Transport、Variable/Stream/Command、Subscription、最小 Hub/SDK/CLI 和跨子树闭环，但尚未迁移生产级身份与准入、QUIC/RFCOMM、旧业务能力、Desktop、Android、MetricsNode、ClipboardNode 和 EmbeddedSDK。

当前工作区包含 10 个旧仓、22 个历史 Go module、多个 Flutter/Wails/Android/Embedded 构建入口，以及主工作区和 MetricsNode 的既有未提交内容。全量迁移必须使用固定 commit 作为只读输入，并为每项旧行为记录 `migrate / replace / drop` 处置，不能从脏工作树静默复制。

## Confirmed Requirements

- 使用一个专用长期迁移分支和 worktree 完成全部 canonical source migration。
- 使用多个有依赖、有测试门禁的变更边界，而不是一个巨大提交。
- 在最终验收之前，旧仓和旧产品入口保持冻结回退路径；不在中途维持双向兼容演进。
- 补齐持久身份、直接父子准入、信任撤销、权限策略、重连、订阅恢复、资源目录和可观察运行状态。
- 迁移当前真实存在的 TCP、QUIC、RFCOMM 能力；串口、USB 或 WebSocket 若没有既有实现和产品需求，不伪装成“迁移”。
- 用 Variable、Stream、Command 重写 VarStore、TopicBus/通知、Exec、Management、File 与 Flow，不复制 SubProto handler/action 分发骨架。
- 迁移 Hub、Go SDK、Desktop、Android、MetricsNode、ClipboardNode 与 Embedded C/MicroPython 实现及其构建入口。
- 所有旧第一方功能必须进入迁移处置矩阵；允许删除无意义旧 API，但必须记录理由和替代行为。
- canonical 默认开发入口、构建和测试在最终切换后只依赖新 monorepo。
- 远端 push、签名发布、应用商店发布、旧仓远端归档和本地旧仓删除仍需单独授权。

## Open Questions

以下问题不阻塞源代码迁移计划，但会影响最终外部发布验收：

- macOS/iOS 签名构建和商店发布需要对应主机、证书和单独发布授权。
- 真实 Android/RFCOMM/ESP32 硬件 smoke 取决于设备可用性；缺失证据不得伪报通过。
- canonical 远端、默认分支和版本发布策略在本地全量迁移通过后单独确认。

## Routed Docs

- [统一节点运行时需求](../requirements/unified-node-runtime.md)
- [节点树、链路与资源架构规范](../specs/node-tree-link-resource-architecture.md)
- [仓库与模块边界规范](../specs/repository-and-module-boundaries.md)
- [使用单一 Canonical Monorepo](../decisions/2026-08-27_single-canonical-monorepo.md)
- [单迁移分支、分门禁实施与最终一次切换](../decisions/2026-08-27_gated-full-migration-cutover.md)
- [全量迁移执行计划](../../plan.md)

## Related Changes

- [Canonical Monorepo 与统一节点运行时第一阶段](../change/2026-08-27_canonical-monorepo-unified-node-runtime.md)
- [vNext 全量迁移与旧仓退役](../change/2026-08-27_vnext-full-migration.md)
