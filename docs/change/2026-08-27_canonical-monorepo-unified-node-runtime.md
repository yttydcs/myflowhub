# 2026-08-27 Canonical Monorepo 与统一节点运行时第一阶段

## 变更背景 / 目标

旧 MyFlowHub 由多个 Git 仓库、多个 Go module 和 SubProto 分发机制共同表达。该结构允许替换局部实现，但同时把协议、权限、路由、订阅和业务能力拆散，导致第一方开发、联调、版本发布与跨仓演进成本过高。

本阶段采用无外部兼容负担的 clean break：建立单一 canonical monorepo 和统一 Node runtime，把权威父子树、可插拔链路、资源、Subscription 与 Command 作为核心概念，并删除新架构中的 SubProto 分发层。

## 具体变更内容

- 建立根 Go module `github.com/yttydcs/myflowhub`，移除 canonical target 中的 `go.work` / `go.work.sum` 与嵌套 module 边界。
- 新增固定头 `MFH3` wire envelope、结构化错误、显式 phase/operation、长度校验与 payload 上限。
- 新增统一 `Driver` / `Pipe` / `Listener` / `LinkSession` 契约，并实现 memory、TCP 两个适配器。
- 建立单父权威节点树、后代路由、拓扑 epoch、Ed25519 join/ack、显式信任与来源证明。
- 将 Variable、Stream、Command 收敛为 Node 拥有的资源，由一个 Registry 管理。
- 将 Subscription 提升为一级运行时关系，提供租约、独立授权边界、快照顺序、Variable 合并、Stream gap 与链路清理。
- 将 Command 提升为一级运行时能力，提供授权来源、并发上限、deadline、去重、panic 隔离和迟到结果观察。
- 组合 Node、Hub、Go SDK 与最小 CLI，证明跨子树订阅、指令、权限允许/拒绝、超时、断连与 reparent。
- 固化同 peer 重连语义：新边提交已签名 epoch，旧会话只能按 peer + epoch 清理自身状态，不得撤销替换后的边。
- 记录 10 个旧仓精确 commit 与 16 个 MetricsNode 既有脏生成文件，旧仓只作为只读迁移输入。

## Docs root

- `D:/project/MyFlowHub3/docs`
- 文档与代码位于同一根 Git 仓库的不同 worktree；本次归档只进行本地 Git 收口，不配置远端、不推送、不发布。

## Intake impact

- `updated`
- 原始请求已经路由到稳定需求、架构决策、技术规范与本归档。

## Feature impact

- `none`
- 第一阶段未切换现有 Desktop、Android、节点应用或其他生产 UI。

## Requirements impact

- `updated`
- 新增并实现统一节点运行时第一阶段需求与验收口径。

## Specs impact

- `updated`
- 新增或明确节点树/链路/资源、仓库边界、wire、资源模型、订阅与指令技术契约。

## Decision impact

- `updated`
- 实施单一 canonical monorepo 与统一权威节点树/可插拔链路两项 Accepted 决策。

## Lessons impact

- `updated`
- 新增会话替换与异步清理必须按 generation/epoch 绑定的复用经验。

## Related intake

- [节点树、订阅与指令重构诉求](../intake/2026-08-27_node-tree-subscription-command-redesign.md)

## Related features

- 无；现有产品迁移属于后续 D02-D05。

## Related requirements

- [统一节点运行时需求](../requirements/unified-node-runtime.md)

## Related specs

- [节点树、链路与资源架构规范](../specs/node-tree-link-resource-architecture.md)
- [仓库与模块边界规范](../specs/repository-and-module-boundaries.md)
- [Wire Protocol vNext](../specs/wire-protocol-vnext.md)
- [Resource Model vNext](../specs/resource-model-vnext.md)
- [Subscription vNext](../specs/subscription-vnext.md)
- [Command vNext](../specs/command-vnext.md)

## Related decisions

- [统一权威节点树与可插拔链路](../decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)
- [使用单一 Canonical Monorepo](../decisions/2026-08-27_single-canonical-monorepo.md)

## Related lessons

- [会话替换与异步清理必须绑定 generation](../lessons/session-replacement-generation-cleanup.md)
- [跨仓 SemVer 发布链](../lessons/cross-repo-semver-release.md)

## 对应 plan.md 任务映射

- `M00`：迁移清单、旧仓 commit、MetricsNode 脏文件保护和稳定文档交接。
- `M01`：根 module、依赖方向与文档/迁移守卫。
- `M02`：`MFH3` envelope、codec、操作、错误和 fuzz。
- `M03`：LinkSession 与 memory/TCP Transport。
- `M04`：身份、信任、节点树、epoch、路由与父控子裁决。
- `M05`：Registry、Variable、Stream、Command 资源定义。
- `M06`：Subscription 租约、聚合、背压、合并与 gap。
- `M07`：Command 调用、超时、去重、迟到结果和 panic 隔离。
- `M08`：Node、Hub、Go SDK 和最小 CLI。
- `M09`：跨子树 memory/TCP 垂直切片与独立验证。
- `D01-D07`：真实 QUIC/RFCOMM/serial、剩余业务与应用迁移、EmbeddedSDK、兼容桥、远端发布等均未纳入本阶段。

## 经验 / 教训摘要

- 物理承载可以扩展，但 LinkSession、树权限、订阅和指令不能根据 Transport 类型分叉。
- 同一 NodeID 的新会话不能只靠“覆盖 map entry”完成重连；拓扑边和清理动作必须携带 epoch。
- 被替换的会话仍必须清理自己的订阅等 link-local 状态，但不得清理新会话拥有的 topology state。
- `WaitGroup.Add` 必须与 `closed` 状态在同一锁下完成，否则 `Close()` 可能先关闭诊断通道，后台协程随后 panic。
- Variable 合并只能有一个权威缓冲层；额外的 buffered output channel 会提前固化中间 revision。
- deadline 与结果竞争应使用无缓冲 handoff，使 timeout 获胜后结果必然进入 late-result 路径。
- Go fuzz 的 wire 等价判断应把 nil 与零长度 byte slice 视为同一 payload。

## 可复用排查线索

- 症状：
  - 同父重连返回 `join topology epoch mismatch`。
  - 重连后旧订阅仍保留到租约过期。
  - 旧 LinkSession 关闭后新树边突然消失。
  - Node 关闭期间出现 `panic: send on closed channel`。
  - Variable 慢消费者偶尔收到中间 revision，而不是最新合并值。
- 触发条件：
  - 新旧会话同时存在，旧会话清理与新会话激活并发。
  - 后台协程在 `Wait()` 开始后才登记。
  - 合并队列之后又增加第二个 buffered channel。
- 关键词：
  - `TopologyEpoch`、`ReattachChild`、`DetachParentEpoch`、`WithdrawChildEpoch`
  - `CleanupLink`、`WaitGroup.Add`、`send on closed channel`
  - `late result`、`Variable coalescing`、`join topology epoch mismatch`
- 快速检查：
  - 确认树边撤销同时校验 peer 和 epoch。
  - 确认 superseded session 仍执行 link-local cleanup。
  - 用 `go test -race` 和同父重连 100 轮检查生命周期竞态。
  - 用空 payload fuzz seed 检查 codec round trip。

## 关键设计决策与权衡

- 权限树与逻辑 Node 树保持同一棵权威树：父节点控制子节点，子节点信任当前已认证父节点；资源挂在 Node 下，而不是单独形成第二棵权限树。
- Transport 只提供双向 Pipe/Listener 能力；蓝牙、TCP、QUIC、串口等实现不得渗透到资源与权限层。
- Subscription 和 Command 是核心运行时 API；Command 只弥补无法由 Variable/Stream 订阅表达的主动行为。
- 请求先向可裁决点上送，授权后转为只向下传播的 control，避免子节点自行伪造父控制。
- 不保留 SubProto compatibility bridge；没有外部用户时优先降低架构复杂度。
- 第一阶段只实现 memory/TCP 基础和核心垂直切片，真实平台迁移延期，以免在核心契约稳定前扩大迁移面。

## 测试与验证方式 / 结果

- `GOWORK=off go test ./runtime/tree ./runtime/subscription ./runtime/command ./runtime/node -count=100 -timeout=5m`
  - 通过。
- `GOWORK=off go test -race ./runtime/... ./tests/integration/... -count=10 -timeout=8m`
  - 通过。
- `GOWORK=off go test ./... -shuffle=on -count=20 -timeout=8m`
  - 通过。
- `GOWORK=off go test ./tests/integration -run TestCrossSubtreeVerticalSlice -count=100 -timeout=8m`
  - memory/TCP 均通过，合计 12.824 秒。
- `GOWORK=off go test ./protocol -run '^$' -fuzz FuzzCodecRoundTrip -fuzztime=15s`
  - 通过 3,527,777 次执行。
- `GOWORK=off go vet ./...`、架构/文档/迁移守卫、`gofmt` 扫描、`git diff --check`
  - 全部通过；仅有 Windows LF-to-CRLF checkout 提示。

## 潜在影响

- 新 module path、wire envelope 与 API 对旧系统不兼容，这是已接受的 clean break。
- D01-D07 尚未完成，现有产品仍运行在旧仓实现上；本阶段不会自动切换生产入口。
- 正式 QUIC、RFCOMM、串口、平台 provider、File/Flow、Desktop/Android、节点应用及 EmbeddedSDK 仍需后续 workflow。
- 当前专用 worktree 位于旧 meta-workspace 下，运行 Go 命令需显式 `GOWORK=off`；canonical repository 自身不跟踪 `go.work`。
- 没有正式 latency/throughput SLO；本阶段只验证有界队列、竞态安全和重复压力基线。

## 回滚方案

- 本阶段是单一 feature commit/merge 边界，可通过 Git revert 回滚 canonical monorepo 变更。
- 若尚未集成，可直接保留或删除 `codex/refactor-monorepo` 专用 worktree/branch，不影响旧仓。
- 旧仓、旧产品入口和 MetricsNode 既有脏生成文件均未被修改，可继续作为回退运行路径。
- 不通过删除或覆盖主检出目录既有脏文件进行回滚；其状态由独立保护快照保留。

## 子Agent执行轨迹

- 本轮未使用子Agent。
- 原因：当前宿主策略要求显式用户授权后才能委派；用户未请求子Agent，所有实现、复核、修复、测试与归档均由主Agent完成。
