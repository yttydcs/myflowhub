# Plan - pluggable-state-backend

## Workflow Information
- Repo: `MyFlowHub-Server`
- Branch: `refactor/pluggable-state-backend`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state`
- Current Stage: `4`

## Stage Records

### Initialization
- guide.md:
  - Root workspace `guide.md` 已读取。
  - 子协议长期文档入口按要求从 `repo/MyFlowHub-Server/docs` 进入。
- base/worktree confirmation:
  - Main control repo: `D:\project\MyFlowHub3` on `master`
  - Participating repos:
    - `MyFlowHub-Server` -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state` -> `refactor/pluggable-state-backend`
    - `MyFlowHub-SubProto` -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state` -> `refactor/pluggable-state-backend`
  - Main repo path remains control-plane only; implementation and repo-local planning stay inside the worktrees above.

### Stage 1 - Requirements Analysis
#### Goal
- 在不破坏现有协议边界的前提下，为 `flow` 和 `varstore` 引入可插拔持久化能力。
- 保持 `config` 独立可用，不因数据库不可用而失效。

#### Scope
- 必须:
  - 为 `flow` 抽取持久化抽象，保留“未配置 PG 时继续本地 JSON”行为。
  - 为 `varstore` 抽取持久化抽象，支持“未配置 PG 时继续纯内存”行为。
  - 明确 owner / cache / route 语义，保证 `varstore` 的权威写入仍由 owner 节点完成。
  - 显式化 runtime 共享依赖，避免继续用 `cfg` 充当 service locator。
- 可选:
  - 提供 PG backend 的最小表结构与初始化逻辑。
  - 为 `flow` / `varstore` 增加 backend 选择配置项。
- 不做:
  - 不持久化 `flow` 的 run 状态、scheduler、runtime context。
  - 不持久化 `varstore` 的订阅、pending、writing、路由缓存等运行期状态。
  - 不把 `flow` / `varstore` 的业务语义搬回 `Server` 仓库。
  - 不做 PG<->JSON 自动迁移工具或历史数据搬运脚本。

#### Use Cases
- 设备节点只需要少量 `varstore` 数据，未配置数据库时依然可正常运行。
- Hub / server 需要重启后保留 `flow` 定义和 `varstore` 业务值时，可配置 PG 做持久化。
- `management config_get/config_set` 仍然通过现有配置层工作，不与业务状态 backend 耦合。

#### Functional Requirements
- `flow` persistence 必须支持 `LoadAll/Save/Delete` 语义。
- `varstore` persistence 必须支持 `LoadAll/Upsert/Delete` 语义。
- `flow` 在 PG 模式下直接存定义本体，而不是“本地文件路径引用”。
- `varstore` 在 PG 模式下仅持久化业务记录 `(owner, name, value, type, visibility)`。
- `varstore set/revoke` 在非 owner 节点只负责转发；owner 节点必须在持久化成功后再广播与回包。
- backend 未配置时：
  - `flow` 默认使用 JSON 文件 backend。
  - `varstore` 默认使用 memory backend。
- backend 已配置但不可用时，不允许静默降级到其他 backend。

#### Non-functional Requirements
- 维持当前 repo 分层：`SubProto` 管业务语义，`Server` 管装配与 backend 生命周期。
- 改动面尽量小，优先复用现有内存 map 和现有 JSON `flow` 语义。
- 配置层保持可用性优先，不依赖 PG 可用性。
- 依赖显式化，避免继续依赖“相同 cfg 指针”共享 registry。

#### Inputs / Outputs
- Inputs:
  - 用户配置的 backend 选择
  - 可选 PG 连接信息
  - 现有 `flow` JSON 文件目录和运行时 `config`
- Outputs:
  - `flow` definitions persistence
  - `varstore` records persistence
  - 明确的 runtime dependency wiring
  - 不变的 `config_get/config_set` 行为

#### Edge Cases
- 配置了 PG 但连接失败：`flow` / `varstore` backend 返回明确错误，不静默回退。
- `varstore` 非 owner 节点收到写请求：不得先写本地持久层。
- `varstore` owner 节点持久化失败：不得发送成功事件、notify、`up_set` 或成功响应。
- `flow` 删除失败：不得只改内存状态后再丢失持久层一致性。
- JSON backend 与 PG backend 切换时，不自动迁移既有数据。

#### Acceptance Criteria
- 不配置 PG 时：
  - `flow` 仍按现有 JSON 文件读写。
  - `varstore` 仍按现有纯内存语义运行。
- 配置 PG 时：
  - `flow` 能从 PG 恢复定义并在 `set/delete` 时更新 PG。
  - `varstore` 能从 PG 预热 records，并在 owner 节点写入时更新 PG。
- `config_get/config_set` 在无 PG 和 PG 不可用场景下仍可用。
- capability registry 和持久化依赖通过显式对象共享，不再依赖 `cfg` 指针身份。

#### Risks
- `flow` / `varstore` constructor 变更会波及多个 handler 与测试。
- PG backend 的失败语义若处理不一致，容易出现“回包成功但未持久化”的分叉。
- `varstore` cache 与持久层的写序若处理不当，会破坏逐跳缓存语义。

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 采用“业务 handler 保留内存态 + 可插拔 persistence adapter + server 显式 wiring”的方案。
- `SubProto` 定义 persistence 接口和 runtime deps 入口；`Server` 提供 JSON/memory/PG backend 实现与选择逻辑。
- `config` 继续作为独立配置层存在，不承载 `flow` / `varstore` 业务状态。

#### Alternatives Considered
- 方案 A: `flow` 继续本地 JSON，PG 只存文件路径。
  - 放弃原因：真相源仍在本地文件，多实例/迁移/备份都不稳定。
- 方案 B: 由 `Server` 直接实现 `flow` / `varstore` 业务读写。
  - 放弃原因：破坏 `SubProto` 负责业务语义、`Server` 只装配的边界。
- 方案 C: 所有运行期状态都搬进 PG。
  - 放弃原因：改动面过大，且 `pending/subscriptions/scheduler` 不需要跨重启恢复。

#### Module Responsibilities
- `MyFlowHub-SubProto/flow`
  - 定义 `FlowPersistence` 接口。
  - 保留 DAG 校验、run/scheduler、事件订阅和协议动作处理。
  - 通过 persistence 读写定义，不再直接假设文件系统是唯一真相源。
- `MyFlowHub-SubProto/varstore`
  - 定义 `VarPersistence` 接口。
  - 保留 owner 路由、权限、订阅、逐跳缓存和 notify/up_* 语义。
  - 把 records map 定位为 memory cache / default memory store。
- `MyFlowHub-SubProto/*`
  - 构造时改为接收显式 `RuntimeDeps` 或 repo-local options，显式拿到 `CapRegistry` / permission / persistence。
- `MyFlowHub-Server`
  - 负责 backend 选择、PG 连接池、JSON/memory/PG adapter 构造与注入。
  - 保持 `hubruntime` config layering 独立。

#### Data / Call Flow
- `flow` startup:
  - `Server` 选择 `JSONFlowPersistence` 或 `PGFlowPersistence`
  - handler `Init()` -> `LoadAll()` -> 填充内存 `flows map`
- `flow set/delete`:
  - 校验请求 -> persistence 写入/删除 -> 更新内存 map -> 调整 scheduler -> 响应
- `varstore set/revoke`:
  - 非 owner 路由节点 -> `assist_*` 上送 -> 不写 persistence
  - owner 节点 -> permission check -> persistence upsert/delete -> 更新本地 records cache -> `changed/deleted` event + notify/up_* -> 响应
- `varstore up_set/notify_set`:
  - 只刷新本地 cache，不写 persistence

#### Interface Drafts
- `RuntimeDeps`
  - `Config core.IConfig`
  - `Logger *slog.Logger`
  - `CapRegistry *execcap.Registry`
  - `PermConfig *permission.Config`
  - `FlowPersistence flow.FlowPersistence`
  - `VarPersistence varstore.VarPersistence`
- `flow.FlowPersistence`
  - `LoadAll(ctx) ([]StoredFlow, error)`
  - `Save(ctx, flowID, doc) error`
  - `Delete(ctx, flowID) error`
- `varstore.VarPersistence`
  - `LoadAll(ctx) ([]StoredVar, error)`
  - `Upsert(ctx, rec) error`
  - `Delete(ctx, owner, name) error`
- Backend config
  - `flow.backend = json | pg`
  - `varstore.backend = memory | pg`
  - PG DSN / connection config in `Server` runtime config

#### Error Handling and Safety
- backend 明确配置但初始化失败 -> 启动失败或 backend 返回明确错误；不 silent fallback。
- owner 节点 persistence 写失败 -> 不更新本地 cache，不发送 success resp / notify / up_*。
- JSON backend 保留 atomic write 语义；PG backend 使用单次 upsert/delete 保证权威写路径。

#### Performance and Testing Strategy
- `flow` / `varstore` 在 PG 模式下启动时全量预热到内存，读取仍优先走内存 cache。
- `varstore` 只在 owner 写路径命中 PG，`up_*` / `notify_*` 不重复写库。
- 测试覆盖:
  - no PG 默认行为不回归
  - PG backend load/save/delete/upsert
  - `varstore` owner / non-owner 写路径顺序
  - backend configured-but-unavailable error path
  - capability registry explicit sharing

#### Extensibility Design Points
- 后续可在 `Server` 增加 `sqlite` backend，而不改 `SubProto` 业务逻辑。
- `RuntimeDeps` 还能承载未来共享对象，避免继续把 `cfg` 当隐式依赖容器。
- `flow` 若将来需要 run archive，可新增独立 persistence，而不污染 definition store。

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 为 `flow` 和 `varstore` 增加可插拔持久化后端，优先支持 PG，同时保持默认无库行为稳定。
- Current state:
  - `flow` 直接文件持久化到 `./flows/*.json`
  - `varstore` 仅内存 `records map`
  - capability registry 依赖 `cfg` 指针共享

#### Docs Governance Routing Decision
- Requirements impact: `none`
- Specs impact: `clarify`
- Related requirements:
  - `none`
- Related specs:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
  - `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`
- Related lessons:
  - `none`
- Canonical destinations:
  - long-lived technical truth -> `repo/MyFlowHub-Server/docs/specs/*.md`
  - workflow execution -> this worktree `plan.md`
  - completed results -> `docs/change/YYYY-MM-DD_topic.md`

#### Related Requirements / Specs / Lessons
- See docs governance routing decision above.

#### Executable Task List
- [x] `SUB1` 显式化 runtime deps 与 capability registry 共享
- [x] `SUB2` 为 `flow` 抽取 persistence 接口并保留 JSON backend
- [x] `SUB3` 为 `varstore` 抽取 persistence 接口并固定 owner 写序
- [x] `SRV1` 在 `Server` 引入 backend 选择、PG wiring 与 adapter
- [x] `SRV2` 更新 `hubruntime/modules/defaultset` 构造路径与测试
- [x] `DOC1` 更新相关 specs 与 change 归档

#### Task Details
##### SUB1 - Explicit Runtime Dependencies
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state\plan.md`
- Goal: 去掉基于 `cfg` 指针的 registry 共享，提供显式 deps / options 入口。
- Files / Modules:
  - `flow`
  - `varstore`
  - `topicbus`
  - `file`
  - `exec`
  - `management`
- Write Set:
  - constructor / options / capability registry wiring
- Acceptance:
  - handler 不再强依赖 `SharedRegistry(cfg)` 作为唯一共享路径
- Test Points:
  - capability provider tests
  - constructor compatibility tests
- Rollback:
  - 恢复旧 constructor 与 `SharedRegistry(cfg)` 路径

##### SUB2 - Flow Persistence Abstraction
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state\plan.md`
- Goal: 把 `flow` 的文件持久化抽成接口，保留 JSON backend 默认行为。
- Files / Modules:
  - `flow/config.go`
  - `flow/handler.go`
  - new `flow` persistence files
- Write Set:
  - `flow/*`
- Acceptance:
  - no PG 时行为不变；`flow` 启动仍能从 JSON 恢复
- Test Points:
  - existing flow tests
  - JSON backend load/save/delete tests
- Rollback:
  - 恢复直接文件读写逻辑

##### SUB3 - VarStore Persistence Abstraction
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state\plan.md`
- Goal: 把 `varstore` 的业务记录抽成可插拔持久化接口，保留 memory cache 语义。
- Files / Modules:
  - `varstore/types.go`
  - `varstore/varstore.go`
  - new `varstore` persistence files
- Write Set:
  - `varstore/*`
- Acceptance:
  - owner 节点 persistence 成功后才广播和回包；非 owner 不写 persistence
- Test Points:
  - owner/non-owner set/revoke path tests
  - memory backend tests
- Rollback:
  - 恢复纯内存 `records map` 主存储逻辑

##### SRV1 - Server Backend Selection And PG Wiring
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state\plan.md`
- Goal: 提供 `flow.backend` / `varstore.backend` 和 PG backend 构造。
- Files / Modules:
  - `hubruntime`
  - `modules/defaultset`
  - new PG adapter package(s)
- Write Set:
  - `hubruntime/*`
  - `modules/defaultset/*`
  - new backend wiring files
- Acceptance:
  - 可按配置创建 JSON/memory/PG backend；PG 配置异常时返回明确错误
- Test Points:
  - backend selection tests
  - PG init error tests
- Rollback:
  - 恢复 `cfg + log` 直连 constructor 路径

##### SRV2 - Runtime Wiring And Regression Coverage
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state\plan.md`
- Goal: 让 `hubruntime/modules` 使用显式 deps，并补足不回归测试。
- Files / Modules:
  - `hubruntime/runtime.go`
  - `modules/defaultset/*`
  - `tests/*`
- Write Set:
  - runtime wiring and tests
- Acceptance:
  - 默认 no-PG 行为通过现有/新增测试
- Test Points:
  - `go test ./...`
  - targeted regression tests for flow and varstore
- Rollback:
  - 恢复旧 runtime wiring

##### DOC1 - Specs Update And Archive
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state\plan.md`
- Goal: 更新长期 specs，并在完成后做 change archive。
- Files / Modules:
  - `repo/MyFlowHub-Server/docs/specs/flow.md`
  - `repo/MyFlowHub-Server/docs/specs/varstore.md`
  - related `docs/change` / indexes
- Write Set:
  - specs and change docs
- Acceptance:
  - specs 反映 backend 语义、默认行为与 config 独立边界
- Test Points:
  - docs impact check
  - manual cross-link verification
- Rollback:
  - 回退 docs changes and archive entries

#### Dependencies
- `SUB1` is a prerequisite for `SRV1` and `SRV2`.
- `SUB2` and `SUB3` must land before final `Server` PG wiring can compile.
- `DOC1` depends on the final implemented semantics, not on preliminary chat decisions alone.

#### Risks and Notes
- `varstore` 现有 tests 可能假设纯内存主存储，改造时要避免破坏 notify/up_* 语义。
- `flow` 默认 JSON backend 必须保持当前 atomic write / startup load 行为。
- PG backend 首轮不做自动迁移工具；需要在文档里明确这一点。

#### Parallelism Assessment
- Current decision: `不并行实现`
- Reason:
  - `SUB1` 的 constructor / deps 形状会直接影响 `SUB2/SUB3/SRV1`
  - 当前阶段先单线程收敛接口，避免跨 repo 冲突
- Additional worktrees already prepared for cross-repo execution; no sub-agent dispatch planned in `3.1`.

#### Issue List
- none

### Stage 3.2 - Implementation
#### Execution Record
- `SUB1`
  - 由 companion `SubProto` worktree 完成显式 `runtimedeps` 与 constructor 兼容层。
- `SUB2`
  - `flow` 已具备 `Persistence` 接口与默认 JSON backend。
- `SUB3`
  - `varstore` 已具备 `Persistence` 接口与默认 memory backend，并固定 owner 写序。
- `SRV1`
  - `modules/defaultset` 新增 backend 选择：
    - `flow.backend=json|pg`
    - `varstore.backend=memory|pg`
    - `state.pg.dsn`
    - `state.pg.flow_table`
    - `state.pg.varstore_table`
  - 新增 PG flow / varstore persistence adapter。
- `SRV2`
  - defaultset 在装配阶段显式构造并共享 `runtimedeps.Deps`。
  - `management/exec/file/topicbus/flow/varstore` 改为显式接收共享 deps。
  - 补齐 backend 选择与 PG 错误路径测试。

#### Verification
- 临时 `go.work` 下 `go test ./... -count=1 -p 1` 通过。

### Stage 3.3 - Code Review
#### Checklist
- 需求覆盖：通过
  - 默认 JSON/memory 行为保持；PG backend 可配置注入；runtime deps 显式共享。
- 架构合理性：通过
  - `Server` 只做装配与 backend 选择，业务语义仍在 `SubProto`。
- 错误路径：通过
  - backend 非法值、缺少 DSN、PG 连接失败均有明确错误，不静默回退。
- 测试覆盖：通过
  - `modules/defaultset`、`modules`、`hubruntime`、`tests` 全量回归通过。
- 残余风险：已知
  - PG backend 目前采用按操作连接，后续若写频率上升可再评估连接池。

### Stage 4 - Docs And Archive
#### Outputs
- specs 更新：
  - `docs/specs/flow.md`
  - `docs/specs/varstore.md`
- 变更归档：
  - `docs/change/2026-03-25_pluggable-state-backend.md`
- 索引更新：
  - `docs/change/README.md`

#### Workflow Status
- 当前 iteration 已完成，可继续下一轮需求或等待用户决定是否结束 workflow。
