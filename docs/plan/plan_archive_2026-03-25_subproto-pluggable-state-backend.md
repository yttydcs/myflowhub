# Plan - pluggable-state-backend-subproto

## Workflow Information
- Repo: `MyFlowHub-SubProto`
- Branch: `refactor/pluggable-state-backend`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state`
- Current Stage: `4`

## Stage Records

### Initialization
- guide.md:
  - Workspace `guide.md` 已读取。
  - Server 仓规格文档作为长期技术入口已读取。
- base/worktree confirmation:
  - Cross-repo companion worktree:
    - `MyFlowHub-Server` -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state`
  - 本 repo 的实现与计划仅在当前 worktree 中进行。

### Stage 1 - Requirements Analysis
#### Goal
- 为 `flow` 和 `varstore` 提供可插拔 persistence 接口，同时显式化 runtime 共享依赖。

#### Scope
- 必须:
  - `flow` 从“直接文件读写”收敛到“interface + JSON backend”
  - `varstore` 从“records 仅内存”收敛到“interface + memory backend”
  - capability registry 改为显式共享对象
- 可选:
  - repo-local backend helper / stored types
- 不做:
  - PG adapter 实现在本 repo 不落地
  - `auth` 持久化本轮不改造

#### Use Cases
- `Server` 可以为 `SubProto` handler 注入不同 persistence backend。
- `SubProto` 业务逻辑不关心 backend 来自 JSON、memory 还是 PG。

#### Functional Requirements
- `flow` 和 `varstore` 需要 repo-local interface，供 `Server` 装配。
- handler 仍然保留当前协议路由与业务判断。
- `varstore` 必须把权威写入顺序固定为“owner persistence success -> cache/event/notify/resp”。

#### Non-functional Requirements
- 最大限度保留现有 handler 逻辑。
- 不把 backend 细节泄露到协议动作层。

#### Inputs / Outputs
- Inputs:
  - explicit deps
  - persistence interfaces
- Outputs:
  - backward-compatible handler constructors
  - testable memory/JSON backends

#### Edge Cases
- nil persistence 时必须有安全默认值。
- old constructor compatibility 需要明确保留或集中迁移。

#### Acceptance Criteria
- repo-local tests 能在无 PG 条件下完整通过。
- no-PG 默认行为与当前一致。
- capability provider 不再依赖 `cfg` 指针身份共享。

#### Risks
- constructor 兼容层不足会影响上游 `Server` 编译。

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 在 `flow` / `varstore` 包内定义 persistence interface 和 memory/JSON default backend；`Server` 只注入实现。

#### Alternatives Considered
- 继续直接用 `cfg` / file system:
  - 放弃原因：无法清晰支持 PG，也让 shared deps 继续隐式化。
- 由 `Server` 直接操作 `SubProto` 内部 map:
  - 放弃原因：会破坏 repo 边界。

#### Module Responsibilities
- `flow`
  - 定义 stored flow 文档结构与 persistence interface
  - 负责 JSON default backend
- `varstore`
  - 定义 stored var 结构与 persistence interface
  - 负责 memory default backend 和 cache coordination
- shared constructor/options
  - 提供显式 deps 入口，包括 `CapRegistry`

#### Data / Call Flow
- `flow`:
  - `Init()` -> `LoadAll` -> fill `flows map`
  - `set/delete` -> persistence -> in-memory update -> response
- `varstore`:
  - non-owner write -> route only
  - owner write -> persistence -> cache -> event / notify / up_*

#### Interface Drafts
- `flow.FlowPersistence`
- `varstore.VarPersistence`
- explicit deps / options carrying `CapRegistry`

#### Error Handling and Safety
- persistence failure stops success side effects.
- default backend must remain available without PG.

#### Performance and Testing Strategy
- keep in-memory hot maps
- test JSON/memory backends separately from handler protocol tests

#### Extensibility Design Points
- future `sqlite` / other backends can be added outside this repo if interfaces stay stable.

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 为 `SubProto` 提供稳定的 persistence/deps extension points。
- Current state:
  - `flow` 直接文件读写
  - `varstore` 直接内存 map
  - registry 依赖 `cfg` 指针共享

#### Docs Governance Routing Decision
- Requirements impact: `none`
- Specs impact: `clarify`
- Related requirements:
  - `none`
- Related specs:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
- Related lessons:
  - `none`
- Canonical destinations:
  - stable technical truth -> server specs
  - repo-local workflow execution -> this `plan.md`

#### Related Requirements / Specs / Lessons
- See docs governance routing decision above.

#### Executable Task List
- [x] `SUB1` 显式化 capability registry / runtime deps
- [x] `SUB2` 抽取 `flow` persistence + JSON backend
- [x] `SUB3` 抽取 `varstore` persistence + memory cache semantics
- [x] `SUB4` 补齐 repo-local tests and compatibility layer

#### Task Details
##### SUB1 - Explicit Runtime Dependencies
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state\plan.md`
- Goal: 去掉 `cfg` 指针共享 registry 的隐式路径。
- Files / Modules:
  - `exec/capability`
  - `flow`
  - `varstore`
  - `topicbus`
  - `file`
  - `management`
- Write Set:
  - constructor / deps wiring
- Acceptance:
  - same runtime can显式共享一个 registry
- Test Points:
  - capability provider tests
- Rollback:
  - 恢复 `SharedRegistry(cfg)` 调用点

##### SUB2 - Flow Persistence Interface
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state\plan.md`
- Goal: 让 `flow` 不再绑定文件系统作为唯一真相源。
- Files / Modules:
  - `flow/*`
- Write Set:
  - `flow/*`
- Acceptance:
  - JSON backend 可完全承接当前默认行为
- Test Points:
  - startup load / save / delete tests
- Rollback:
  - 恢复当前文件路径实现

##### SUB3 - VarStore Persistence Interface
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state\plan.md`
- Goal: 把 `records` 与运行期 pending/subscription 状态拆开。
- Files / Modules:
  - `varstore/*`
- Write Set:
  - `varstore/*`
- Acceptance:
  - memory backend = current behavior
  - owner persistence order fixed
- Test Points:
  - owner/non-owner set/revoke
  - notify/up_set cache refresh
- Rollback:
  - 恢复 `records map` 直接主存储

##### SUB4 - Tests And Compatibility
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state\plan.md`
- Goal: 让上游 `Server` 接入前 repo-local API 可稳定编译和验证。
- Files / Modules:
  - affected tests
  - constructor compatibility shims if needed
- Write Set:
  - tests and compatibility helpers
- Acceptance:
  - repo-local `go test ./...` passes
- Test Points:
  - handler tests
  - capability tests
- Rollback:
  - 回退 interface / constructor split

#### Dependencies
- `SUB1` precedes `SUB2` / `SUB3`.
- `SUB4` depends on final interface shapes.
- Companion server worktree will consume the final constructor / interface shape.

#### Risks and Notes
- 若 constructor 改动过大，Server 仓接线会出现级联改动。
- `varstore` cache / notify / up_* 语义是高风险区，不能只看持久化 happy path。

#### Parallelism Assessment
- Current decision: `不并行实现`
- Reason:
  - `SUB1` interface 形状未固定前，并行收益低且冲突高

#### Issue List
- none

### Stage 3.2 - Implementation
#### Execution Record
- `SUB1`
  - 新增 `exec/runtimedeps` 作为显式共享 runtime deps 入口。
  - `exec/file/flow/topicbus/varstore/management` 均新增 `WithDeps` 或 `WithOptions` 构造入口。
  - 默认构造函数保持兼容，测试覆盖显式共享对象注入。
- `SUB2`
  - `flow` 新增 `Persistence` 接口与默认 `JSON` backend。
  - `flow.Init()/set/delete` 改为通过 persistence 读写定义。
  - `flow` 新增 injected persistence 初始化与失败语义测试。
- `SUB3`
  - `varstore` 新增 `Persistence` 接口与默认 `Memory` backend。
  - owner 本地 `set/revoke` 改为先持久化，再更新 cache / trigger / notify / up_* / resp。
  - `Init()` 增加 persistence preload；远端 `notify/up_*/*_resp` 仍只刷新 cache。
- `SUB4`
  - 补齐 constructor compatibility / persistence / failure-order tests。

#### Verification
- 临时 `go.work` 下通过：
  - `broker`
  - `exec`
  - `file`
  - `flow`
  - `management`
  - `topicbus`
  - `varstore`

### Stage 3.3 - Code Review
#### Checklist
- 需求覆盖：通过
  - 显式 deps、`flow` JSON backend、`varstore` memory backend、owner 写序均已落地。
- 架构合理性：通过
  - `SubProto` 仅定义业务接口和默认 backend；PG 适配保持在 `Server` 仓。
- 稳定性与一致性：通过
  - 默认无库行为保持不变；旧 constructor 仍可用。
- 测试覆盖：通过
  - 增加显式 deps、preload 与 persistence failure 顺序测试。
- 残余风险：已知
  - `flow/varstore` 的 PG 可用性语义最终由 `Server` 装配层决定；长期真相已在 server specs 记录。

### Stage 4 - Docs And Archive
#### Outputs
- 长期 specs 更新位置：
  - `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state\docs\specs\varstore.md`
- 变更归档：
  - `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state\docs\change\2026-03-25_pluggable-state-backend.md`

#### Workflow Status
- 当前 iteration 已完成，可继续进入下一轮需求，或等待用户决定是否结束 workflow。
