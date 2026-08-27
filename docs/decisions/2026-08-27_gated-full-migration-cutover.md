# 2026-08-27 单迁移分支、分门禁实施与最终一次切换

## Status

Accepted

## Context

统一节点运行时第一阶段已经证明新核心闭环，但全部第一方产品仍运行在旧多仓/SubProto 架构上。项目没有外部兼容用户，可以 clean break；同时迁移范围覆盖协议、运行时、Hub、多个桌面与移动应用、节点应用、EmbeddedSDK 和真实 Transport，若压缩为一个无中间门禁的提交，将难以定位回归、验证平台差异或安全地回滚。

## Options Considered

### 1. 每迁移一个产品就独立发布并长期兼容两套架构

- 优点：单次变更较小。
- 缺点：需要长期 wire/SDK/数据兼容层，重新产生跨版本协调成本。
- 结论：拒绝。

### 2. 所有源码一次写完并压缩成一个提交

- 优点：表面上切换简单。
- 缺点：不可分段审查和回滚，跨平台失败难定位，长时间无法获得可信验证反馈。
- 结论：拒绝。

### 3. 一个迁移分支、多个门禁变更、最终一次切换

- 优点：终态只有新架构，同时保留阶段验证、问题隔离和回滚能力。
- 缺点：分支存续时间较长，必须严格维护依赖顺序、迁移矩阵和测试门禁。
- 结论：采用。

## Decision

1. 全量迁移在 `refactor/vnext-full-migration` 专用分支/worktree 中进行。
2. 所有第一方 canonical source migration 属于同一个 workflow，但按协议与运行能力、Transport、业务能力、产品、Embedded、构建验收等门禁分段实施。
3. 每个门禁必须保持仓库可构建、可测试，并具备明确验收和回滚边界；不得把“最终一次切换”误解为“一个巨大提交”。
4. 旧仓在最终验收前保持只读回退来源。新分支不向旧仓回写，也不建立长期 legacy bridge。
5. 每项旧功能和构建入口必须在迁移矩阵中标记为 `migrated`、`replaced` 或 `dropped-with-reason`，禁止静默遗漏。
6. 最终本地切换条件是 canonical 默认入口不再依赖旧 module、SubProto、旧仓 `replace` 或嵌套 Git；完整产品矩阵和可用环境中的验证门禁通过。
7. 远端推送、签名发布、应用商店发布、旧仓远端归档和本地旧仓删除不由本决策自动授权，必须在本地迁移完成后单独执行。

## Consequences

- 新架构不承担临时兼容层的长期维护成本。
- 执行计划会很大，但任务依赖、写集、验收和回滚必须逐项明确。
- 某个平台或硬件证据不可用时，相关发布验收保持未验证，不影响已经完成的源码迁移事实，但不能宣称对应发布已完成。
- 迁移分支只有在最终门禁通过后才适合合并；中途产品入口仍指向旧系统。

## Confidence

High。项目无外部兼容负担，且用户明确选择全量迁移；分门禁执行是同时满足 clean break 与工程可控性的最低复杂度方案。

## Supersedes / Superseded By

- 不取代 [使用单一 Canonical Monorepo](2026-08-27_single-canonical-monorepo.md)，而是定义其剩余迁移和切换策略。
- 不取代 [统一权威节点树与可插拔链路](2026-08-27_authoritative-node-tree-and-pluggable-links.md)。

## Related Features

- 产品 feature 文档将在全量迁移的首个执行门禁中建立。

## Related Specs

- [仓库与模块边界规范](../specs/repository-and-module-boundaries.md)
- [节点树、链路与资源架构规范](../specs/node-tree-link-resource-architecture.md)

## Related Changes

- [Canonical Monorepo 与统一节点运行时第一阶段](../change/2026-08-27_canonical-monorepo-unified-node-runtime.md)
