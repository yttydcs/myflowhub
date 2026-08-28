# Plan - flow run archive backend

## Workflow Information
- Repo:
  - `MyFlowHub-SubProto`
- Branch:
  - `feat/run-archive-backend`
- Base:
  - `main`
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\subproto-run-archive-backend`
- Current Stage:
  - `3.1`
- Related Worktrees:
  - `MyFlowHub-Server`
    - Branch: `feat/run-archive-backend`
    - Worktree: `D:\project\MyFlowHub3\worktrees\server-run-archive-backend`
    - Control Doc: `D:\project\MyFlowHub3\worktrees\server-run-archive-backend\todo.md`

## Stage Records

### Initialization
- `guide.md`:
  - 已阅读 `D:\project\MyFlowHub3\guide.md`
  - 约束确认：
    - commit 信息使用中文
    - 所有 worktree 必须位于 `D:\project\MyFlowHub3\worktrees\`
    - 子协议稳定文档以 `repo/MyFlowHub-Server/docs` 为准
- base/worktree confirmation:
  - `MyFlowHub-SubProto` 主线存在用户既有脏改动：
    - `auth/actions_login.go`
    - `auth/display_name_test.go`
  - 本轮实现不得在主仓直接改动，且不得触碰上述无关文件
  - 已创建专用 worktree：
    - `D:\project\MyFlowHub3\worktrees\subproto-run-archive-backend`
    - `D:\project\MyFlowHub3\worktrees\server-run-archive-backend`

### Stage 1 - Requirements Analysis
#### Goal
- 把 retained run archive 从“固定本地 sidecar”升级为“独立可插拔 backend”，优先支持 `pg`，同时保持未配置 PG 时继续正常运行。

#### Scope
- 必须:
  - 为 retained run archive 抽取独立 store 接口，不污染现有 flow definition `Persistence`
  - 保持 `flow` 在未配置 PG 时继续可用：
    - `run archive backend=off` 时保持当前默认无 archive 行为
    - `run archive backend=file` 时继续使用本地 JSON sidecar
  - 支持 `Server` 在显式配置 `pg` 时注入 PG archive backend
  - 保持现有查询面不变：
    - `list_runs`
    - `status`
    - `detail`
  - 保持 retained window 仍由 `flow.max_retained_runs` 控制
  - 兼容当前 `flow.run_archive_enabled`：
    - `true` 等价于 `file`
    - 未设置时不改变默认关闭行为
  - backend 显式配置为 `pg` 但 PG 不可用时，返回明确错误，不静默降级
- 可选:
  - 为 PG archive 增加最小表结构与表名配置
  - 为 file backend 增加更清晰的配置键，替代单纯 bool 开关
- 不做:
  - 不新增 Proto action 或改 wire
  - 不把 archive 并入 `flow.Persistence`
  - 不恢复活动 run、scheduler、trigger dedup 记忆
  - 不做 file/pg 自动迁移工具
  - 不把 retained window 扩展为无限历史

#### Use Cases
- 单节点轻量部署未配置 PG，仍能继续使用当前 flow 与可选 file archive。
- Server 节点显式配置 PG 后，retained run 在重启后可从 PG 恢复查询，不依赖本地文件。
- 删除 flow 定义后，retained window 内的 archived run 仍可通过既有查询面读取。

#### Functional Requirements
- `flow` 必须定义独立的 run archive store 接口，至少支持：
  - `LoadAll(ctx) ([]ArchivedRunRecord, error)`
  - `Save(ctx, record) error`
  - `Delete(ctx, flowID, runID) error`
- `flow` 在 backend=off 时不得引入额外 archive I/O。
- `flow` 在 backend=file 时必须继续把 retained run 写入 `flow.base_dir/_runs/<flow_id>/<run_id>.json`。
- `flow` 在 backend=pg 时必须能从 injected store 预热 retained runs，并在 run 终态 / prune 时更新 PG。
- `Init()` 必须在 definition load 后再预热 archived retained runs。
- `finalizeRun` 必须继续维持 `archive -> prune` 顺序。
- `delete` 后 retained archived runs 仍可查询，直到超出 retained window 被回收。
- `flow.run_archive_enabled=true` 时，不要求同时配置新 backend key，也必须工作。

#### Non-functional Requirements
- 无 PG 默认路径必须保持稳定，不能因为新增 PG 依赖而要求额外配置。
- PG 必须是显式 opt-in，且 failure 语义清晰可审计。
- 查询路径应继续优先命中内存索引，避免每次 `status/detail/list_runs` 都直接打后端。
- backend 边界要清晰，便于未来新增 `sqlite` 等实现。

#### Inputs / Outputs
- 输入:
  - `Server/docs/requirements/flow_data_dag.md`
  - `Server/docs/specs/flow.md`
  - `SubProto/flow` 当前 file archive 与 persistence 实现
  - `Server/modules/defaultset/state_backends.go` 的 PG wiring 模式
- 输出:
  - `SubProto/flow` 独立 archive store 抽象 + file backend
  - `Server` 可选 PG archive backend 装配
  - 对应测试与 docs 变更

#### Edge Cases
- backend 显式配置非法值
- `backend=pg` 但缺 `state.pg.dsn`
- `backend=pg` 但 PG 连接失败
- legacy bool 与新 backend key 同时出现时的优先级
- file backend 与 definition JSON backend 共用 `flow.base_dir` 时的路径边界
- prune 发生后 archive backend 删除失败
- archived run 记录损坏或字段不合法

#### Acceptance Criteria
- 不配置 PG 时：
  - `flow` 默认行为不变
  - `flow.run_archive_enabled=true` 仍走 file archive
- 配置 `flow.run_archive.backend=file` 时：
  - retained archive 行为与当前 sidecar 语义一致
- 配置 `flow.run_archive.backend=pg` 时：
  - retained archive 能从 PG 预热并支持既有查询面
- 显式配置 `pg` 但缺 DSN 或连接失败时：
  - `Server` 初始化返回明确错误
- `list_runs/status/detail` 对 retained archived run 的查询语义保持一致

#### Risks
- archive 抽象若误复用 definition persistence，会把职责重新耦合
- config 兼容策略若定义不清，容易破坏现有 `run_archive_enabled` 部署
- PG adapter 若直接进入查询路径，可能引入额外性能回退

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 采用“`SubProto` 定义 archive store 接口 + repo-local file backend + `Server` 显式注入 PG backend”的方案。
- `Handler` 继续以内存 `runs/runOrderByFlow` 作为查询索引真相；archive backend 只负责：
  - 启动预热
  - 终态保存
  - 超限删除
- 默认不注入 archive store 时：
  - 根据 config 解析结果选择 `off` 或 repo-local file store
- `Server` 仅在 `backend=pg` 时注入 PG archive store。

#### Alternatives Considered
- 方案 A：继续把 file archive 逻辑写死在 `run_archive.go`
  - 不采用原因：
    - 无法干净支持 PG
    - 无法显式表达 `off/file/pg` 三态
- 方案 B：把 archive 合并到现有 `flow.Persistence`
  - 不采用原因：
    - 会把 definition store 与 retained run archive 混成一类职责
- 方案 C：查询时直接命中 backend，不做内存预热
  - 不采用原因：
    - 会让 `status/detail/list_runs` 退化为高频 I/O 路径

#### Module Responsibilities
- `MyFlowHub-SubProto/flow`
  - 定义 archive config、record 结构、archive store 接口
  - 提供 default file archive store
  - 保持 handler 查询、run 状态、prune、delete 语义
- `MyFlowHub-Server/modules/defaultset`
  - 解析 `flow.run_archive.backend`
  - 在 `pg` 模式下构造 PG archive store
  - 在错误配置时返回明确错误
- `MyFlowHub-Server/docs`
  - 收敛 run archive backend 配置与默认行为说明

#### Data / Call Flow
- startup:
  - `Server` 读取 archive backend 配置
  - `backend=pg` -> 注入 PG store
  - `backend=file/off` -> 不注入，由 `SubProto` 本地选择
  - `Handler.Init()` -> load definitions -> load archived runs -> 填充内存 retained index
- run terminal:
  - `finalizeRun()` -> `archiveStore.Save(record)` -> `pruneRuns()` -> 对超限 run 调 `archiveStore.Delete(...)`
- delete flow:
  - 删除 definition 与活动 run
  - archived retained runs 保留在 archive store，不立即清空
- query:
  - `status/detail/list_runs` 继续读内存 retained index，不直接打 backend

#### Interface Drafts
- config:
  - `flow.run_archive.backend = off | file | pg`
  - `flow.run_archive_enabled = true|false` 作为 legacy 兼容入口
  - `state.pg.flow_run_archive_table` 作为 PG archive 表名
- `flow.RunArchiveStore`
  - `LoadAll(ctx context.Context) ([]ArchivedRunRecord, error)`
  - `Save(ctx context.Context, rec ArchivedRunRecord) error`
  - `Delete(ctx context.Context, flowID, runID string) error`
- `flow.HandlerOptions`
  - 新增 `RunArchiveStore RunArchiveStore`

#### Error Handling and Safety
- legacy bool 与新 backend key 并存时：
  - 新 backend key 优先
- `backend=pg` 且配置缺失/连接失败：
  - 由 `Server` 初始化显式失败
- backend 读写失败：
  - `SubProto` 记录明确 warning
  - 查询面继续依赖已在内存中的 retained runs
- file backend 路径仍继续受 `flow_id/run_id` 校验

#### Performance and Testing Strategy
- 继续保持查询命中内存索引，backend I/O 只发生在：
  - startup preload
  - terminal save
  - prune delete
- 测试重点：
  - legacy bool -> file backend 兼容
  - `off/file/pg` backend 选择
  - file backend reload / prune / delete-after-flow-delete
  - PG backend 构造与连接失败路径
  - injected fake archive store 的 save/load/delete 顺序

#### Extensibility Design Points
- backend 抽象允许未来增加 `sqlite`
- archived record 与 store 接口独立于 definition persistence，避免跨语义污染
- `Server` 继续只负责 backend 选择和外部依赖装配

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 为 retained run archive 增加独立 backend 抽象与可选 PG 支持，同时保持无 PG 默认路径稳定。
- Current state:
  - archive 由 `flow.run_archive_enabled` 控制，固定写本地 `_runs/*.json`
  - `flow.Persistence` 只负责 definition
  - `Server` 已有 flow/varstore 的 PG backend 装配模式，可复用

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact:
  - `clarify`
- Specs impact:
  - `clarify`
- Lessons impact:
  - `none`（当前先不预设 lessons，待 Stage 4 再判断）

#### Related Requirements / Specs / Lessons
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\server-run-archive-backend\docs\requirements\flow_data_dag.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\server-run-archive-backend\docs\specs\flow.md`
- Related lessons:
  - `none`

#### Executable Task List
- `RA-SUB-1`：抽取 run archive config 与 store 接口，保留 legacy bool 兼容
- `RA-SUB-2`：把当前 local JSON sidecar 改造成 file archive store，并把 handler 改为走 store
- `RA-SRV-1`：在 `Server/defaultset` 增加 archive backend 选择与 PG archive store
- `RA-SRV-2`：更新 `requirements/specs`，明确 `off/file/pg` 与“无 PG 默认工作”语义
- `RA-VER-1`：补齐 repo-local 与 cross-repo 验证

#### Task Details
##### RA-SUB-1 - Archive Config And Interface
- Owner:
  - Main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\subproto-run-archive-backend`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\subproto-run-archive-backend\todo.md`
- Goal:
  - 为 archive 增加 `off/file/pg` 配置模型与独立 store 接口
- Files / Modules:
  - `flow/config.go`
  - `flow/handler.go`
  - new archive store file(s)
- Write Set:
  - `flow/*`
- Acceptance:
  - legacy `flow.run_archive_enabled=true` 仍等价于 file archive
  - 无显式 backend 时默认继续关闭 archive
- Test Points:
  - config parsing tests
  - fake store injection tests
- Rollback:
  - 回退到当前 bool + fixed file archive 方案

##### RA-SUB-2 - File Archive Store Migration
- Owner:
  - Main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\subproto-run-archive-backend`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\subproto-run-archive-backend\todo.md`
- Goal:
  - 把当前 `_runs/*.json` 逻辑收敛为 file backend，不改查询语义
- Files / Modules:
  - `flow/run_archive.go`
  - `flow/runtime_fix_test.go`
  - related tests
- Write Set:
  - `flow/*`
- Acceptance:
  - file backend reload / prune / delete 后查询行为与当前一致
- Test Points:
  - existing archive tests
  - new file backend unit tests
- Rollback:
  - 恢复当前直接文件 helper 方案

##### RA-SRV-1 - Archive Backend Selection And PG Wiring
- Owner:
  - Main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\server-run-archive-backend`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\server-run-archive-backend\todo.md`
- Goal:
  - 为 run archive 增加 `pg` backend 装配，默认不影响无 PG 场景
- Files / Modules:
  - `modules/defaultset/state_backends.go`
  - `modules/defaultset/flow_enabled.go`
  - `modules/defaultset/state_backends_test.go`
- Write Set:
  - `modules/defaultset/*`
- Acceptance:
  - `backend=pg` 时能构造 PG archive store
  - 缺 DSN / 连接错误明确失败
  - `backend=off|file` 时不要求 PG
- Test Points:
  - backend selection tests
  - PG error-path tests
- Rollback:
  - 移除 archive backend wiring，退回 repo-local file/off

##### RA-SRV-2 - Requirements And Specs Clarification
- Owner:
  - Main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\server-run-archive-backend`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\server-run-archive-backend\todo.md`
- Goal:
  - 把 archive backend 的默认行为、PG 可选语义、legacy 兼容写进稳定文档
- Files / Modules:
  - `docs/requirements/flow_data_dag.md`
  - `docs/specs/flow.md`
- Write Set:
  - `docs/requirements/*`
  - `docs/specs/*`
- Acceptance:
  - 文档明确 `PG` 是 opt-in，不配置 PG 时系统仍正常运行
- Test Points:
  - 文档与实现交叉核对
- Rollback:
  - 回退文档到 file-only archive 描述

##### RA-VER-1 - Validation
- Owner:
  - Main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\subproto-run-archive-backend`
  - `D:\project\MyFlowHub3\worktrees\server-run-archive-backend`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\subproto-run-archive-backend\todo.md`
- Goal:
  - 完成 repo-local 和 cross-repo 验证
- Files / Modules:
  - tests only
- Write Set:
  - none
- Acceptance:
  - `SubProto flow` tests 通过
  - `Server defaultset/hubruntime` tests 通过
  - 主仓联调 `hubruntime` 与 `flow` archive 相关测试通过
- Test Points:
  - `go test github.com/yttydcs/myflowhub-subproto/flow/...`
  - `go test ./modules/defaultset/... ./hubruntime/...`
- Rollback:
  - 定位失败任务并回退对应改动

#### Dependencies
- `RA-SRV-1` 依赖 `RA-SUB-1/2` 暴露稳定接口
- `RA-SRV-2` 依赖接口和配置命名最终定稿
- `RA-VER-1` 依赖前面所有任务完成

#### Risks and Notes
- 需要避免误碰 `MyFlowHub-SubProto` 主仓已有 `auth/*` 脏改动
- 若新配置命名与现有 `flow.backend` 风格不一致，后续维护成本会升高
- 需要明确 legacy bool 优先级，避免部署歧义

#### Parallelism Assessment
- 本轮不派发子 Agent。
- 原因：
  - `SubProto` archive 接口与 `Server` wiring 紧耦合，下一步实现顺序依赖明显
  - 先在主执行链上把接口定稳，再做 `Server` 注入更安全

#### Issue List
- none

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Execution Record
- `RA-SUB-1`
  - `flow/config.go`
    - 新增 `flow.run_archive.backend = off | file | pg`
    - 保留 `flow.run_archive_enabled` 兼容，并映射到 `file`
    - 显式拒绝不支持的 backend 值
  - `flow/handler.go`
    - `HandlerOptions` 新增 `RunArchiveStore`
    - 新增 injected archive store / backend 解析 / config fail-fast
    - `currentRunArchiveStoreLocked()` 保持“手工打开 archive 时默认 file backend”的兼容路径
- `RA-SUB-2`
  - `flow/run_archive.go`
    - 抽出 `RunArchiveStore`
    - 抽出 `FileRunArchiveStore`
    - `persist/load/prune` 改为走统一 store
    - 归档记录新增内部排序键 `archived_at_ns`，保证 reload 后 retained run 顺序稳定
  - `flow/config_test.go`
    - 覆盖默认值、legacy bool、backend override、`pg`、非法值
  - `flow/run_archive_store_test.go`
    - 覆盖 injected archive store 的 load/save/delete 和 pg-without-store fail-fast
  - `flow/runtime_fix_test.go`
    - `loadArchivedRunsFromDisk()` 更新为 backend-agnostic `loadArchivedRuns()`

### Stage 3.3 - Review
- 需求覆盖：通过
  - `off/file/pg`、legacy bool 兼容、无 PG 默认可用、retained reload/prune/delete 语义均已覆盖
- 架构合理性：通过
  - archive 仍独立于 definition `Persistence`，`Server` 只负责可选 PG 装配
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 查询面继续走内存 retained index；backend I/O 仅发生在 preload/save/delete
- 可读性与一致性：通过
  - `RunArchiveStore` 与 `FileRunArchiveStore` 边界清晰，legacy 配置优先级显式
- 可扩展性与配置化：通过
  - backend 抽象已留出 `sqlite` 等后续扩展位
- 稳定性与安全：通过
  - `backend=pg` 且未注入 store 会显式初始化失败
  - file backend 仍受 `flow_id/run_id` 校验保护
- 测试覆盖情况：通过
  - config parsing、injected store、reload/prune/delete、cross-repo wiring 均已验证
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未使用子Agent

### Stage 4 - Change Archive
- 使用 `$m-docs` 校验归档路由、requirements/specs 影响和索引更新。
- Change:
  - `docs/change/2026-04-02_flow-run-archive-backend-runtime.md`
  - `docs/change/2026-04-02_server-flow-run-archive-backend-contract.md`
- Requirements impact:
  - `updated`
  - `repo/MyFlowHub-Server/docs/requirements/flow_data_dag.md`
- Specs impact:
  - `updated`
  - `repo/MyFlowHub-Server/docs/specs/flow.md`
- Lessons impact:
  - `none`
- Related requirements:
  - `repo/MyFlowHub-Server/docs/requirements/flow_data_dag.md`
- Related specs:
  - `repo/MyFlowHub-Server/docs/specs/flow.md`
- Related lessons:
  - `none`
- Validation:
  - `GOWORK=D:\project\MyFlowHub3\.tmp\verify-run-archive-backend\go.work go test github.com/yttydcs/myflowhub-subproto/flow -count=1 -p 1`
  - `GOWORK=D:\project\MyFlowHub3\.tmp\verify-run-archive-backend\go.work go test github.com/yttydcs/myflowhub-server/modules/defaultset/... github.com/yttydcs/myflowhub-server/hubruntime/... -count=1 -p 1`
