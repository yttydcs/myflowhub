# 2026-08-27 使用单一 Canonical Monorepo

## Status

Accepted

## Context

当前 MyFlowHub 工作区包含 10 个第一方 Git 仓库和 22 个 Go module。运行时主链被拆分为 Proto、Core、SDK、SubProto 和 Server，其中 SubProto 又按功能维护多套独立 module 与版本标签。一次协议或运行时语义调整往往需要跨仓修改、连续发布和逐级升级依赖。

项目即将重新定义节点树、资源、订阅、指令和链路模型。当前没有项目外部用户依赖既有 Go module path、旧 wire 协议或仓库发布节奏，因此无需为外部兼容维持现有边界。

## Options Considered

### 1. 保留现有多仓并原地重构

- 优点：无需迁移 Git 仓库和 CI。
- 缺点：核心语义仍需跨 10 个仓库和多套 module 版本协调；无法原子修改和验证；旧边界会继续影响新架构。
- 结论：拒绝。

### 2. 仅合并 Proto、Core、SDK、SubProto 和 Server

- 优点：运行时可以原子修改，应用继续独立发布。
- 缺点：Desktop、Android、节点应用和 EmbeddedSDK 仍需跨仓同步协议与客户端接口；大规模重构期间仍会产生临时版本链。
- 结论：可行但不采用。它适合已有外部消费者或独立团队的项目，不符合当前情况。

### 3. 所有第一方代码进入单一 canonical monorepo

- 优点：协议、运行时、传输、应用、EmbeddedSDK、测试和文档可以原子演进；组件仍可独立构建和发布；不再通过 Git 仓库表达内部架构边界。
- 缺点：需要重建目录结构、CI 路径过滤和发布入口；迁移期间必须保护现有未提交工作及历史资料。
- 结论：采用。

## Decision

1. 建立一个新的 canonical monorepo，容纳全部第一方实现、应用、SDK、协议、测试和长期文档。
2. Git 仓库不再作为 Core、Proto、SDK、SubProto、Server 或各内部能力之间的架构隔离手段；隔离由 package、接口、依赖方向和测试负责。
3. 默认使用一个根 Go module。只有 Android/gomobile、独立工具链或其他已验证的构建约束确实要求时，才能增加嵌套 module，并必须记录原因。
4. 不使用 Git submodule 或嵌套 Git 仓库重新制造多仓协作问题。
5. Desktop、Android、Hub、节点应用和 EmbeddedSDK 保持独立构建产物与发布入口，但共享同一源码仓和协议真相。
6. 不承诺兼容旧 Go module path、旧 SubProto API 或旧 wire 行为。迁移期如需要兼容适配，只能作为有明确删除条件的内部桥接层。
7. 现有仓库在切换完成后转为只读历史来源。保留 commit、tag、plan/change 和协议文档，不通过删除或改写历史来隐藏架构演进。
8. 不要求把所有旧 Git 历史拼接进新仓。新仓文档必须记录旧仓最后基线、来源 commit/tag 和新目录映射，从而保持可追溯性。
9. 新内核应并行建立并通过纵向闭环验证；旧 SubProto 架构不作为新核心骨架原样迁入。

## Consequences

### Positive

- 节点树、权限、资源、订阅、指令和 Transport 可以在一个变更中完成实现与验证。
- 不再需要为内部协同连续发布 Proto、Core、SubProto、Server 和 SDK 标签。
- 跨语言协议夹具、EmbeddedSDK 和第一方应用可以在同一提交中同步。
- 独立构建产物仍可使用目录级 CI 和组件版本发布。

### Costs and Risks

- 初次迁移范围大，必须使用独立工作区并保护当前根仓和 MetricsNode 的未提交改动。
- 新仓若缺少清晰依赖规则，可能从“过度拆分”滑向“无边界大包”；因此必须保留 package 级依赖约束。
- 旧仓归档、远端只读设置和发布切换属于后续显式操作，不能在讨论阶段直接执行。

## Confidence

High。项目没有外部兼容负担，且当前核心模型仍在快速演进，单一源码真相带来的收益明显高于独立仓库发布隔离。

## Supersedes / Superseded By

- 取代 [repos.md](../../repos.md) 中“Proto/Core/SDK/SubProto/Server 作为长期独立仓库边界”的未来方向；在迁移完成前，`repos.md` 仍描述当前实际工作区，不应提前改写为已完成状态。
- 不删除既有仓库职责和发布历史；它们在切换前仍是旧系统的有效事实。

## Related Features

- 暂无独立 feature 文档。

## Related Specs

- [仓库与模块边界规范](../specs/repository-and-module-boundaries.md)
- [节点树、链路与资源架构规范](../specs/node-tree-link-resource-architecture.md)

## Related Changes

- [Canonical Monorepo 与统一节点运行时第一阶段](../change/2026-08-27_canonical-monorepo-unified-node-runtime.md)
