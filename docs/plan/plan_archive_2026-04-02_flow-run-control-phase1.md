# Plan - subproto-run-control-phase1

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-SubProto`
- Branch: `feat/run-control-phase1`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1`
- Current Stage: `4 Archived / RC-P1-4 Completed`
- Additional Worktrees:
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1`
  - `D:\project\MyFlowHub3\worktrees\core-run-control-phase1`

## Stage Records

### Initialization
- `guide.md`: 已阅读，确认提交信息使用中文，且所有 worktree 必须位于 `D:\project\MyFlowHub3\worktrees\`
- participating repos/modules/base:
  - `MyFlowHub-Proto/protocol/flow`
  - `MyFlowHub-SubProto/flow`
  - `MyFlowHub-Core/config`
  - `MyFlowHub-Server/docs/requirements/flow_data_dag.md`
  - `MyFlowHub-Server/docs/specs/flow.md`
  - `MyFlowHub-Server/hubruntime`
  - base branch:
    - `Proto/SubProto/Server`: `main`
    - `Core`: `master`
- dedicated branches/worktrees:
  - `Proto`: `feat/run-control-phase1` @ `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1`
  - `SubProto`: `feat/run-control-phase1` @ `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1`
  - `Server`: `feat/run-control-phase1` @ `D:\project\MyFlowHub3\worktrees\server-run-control-phase1`
  - `Core`: `feat/run-control-phase1` @ `D:\project\MyFlowHub3\worktrees\core-run-control-phase1`
- active execution worktree:
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1`

### Stage 1 - Requirements Analysis
#### Goal
- 将 `flow` 从“仅支持启动和查询”补到“具备基础 run control 与更清晰治理边界”的下一阶段。
- 先形成按优先级排序的可执行补强清单，并从第一项 `cancel_run` 开始落地。

#### Scope
- 必须：
  - 明确本阶段 `P0/P1/P2` 可执行清单
  - 第一顺序项补齐显式 `cancel_run` 控制能力
  - 为 `cancel_run` 补齐稳定 requirements/specs、Proto 契约、SubProto 实现与测试
  - 同步 Proto canonical 协议映射文档及根 workspace 副本，避免 Flow 契约继续漂移
- 可选：
  - 若 `cancel_run` 落地后改动面很小，可顺手收敛明显的文档遗漏
- 不做：
  - 本轮不实现 `list_runs/history`
  - 本轮不实现 `flow.run` / `flow.read` 细粒度权限
  - 本轮不实现 `branch` / `foreach` / `script` / `cron`

#### Use Cases
- 调用方可以按 `flow_id + run_id` 显式取消正在执行的 run，而不是只能删除整个 flow。
- 取消后，`status` 与 `detail` 能返回 `cancelled` 状态和取消原因。
- 后续迭代可在同一 checklist 上继续实现 run history、权限细化和协议映射治理。

#### Functional Requirements
- 需要新增显式 `cancel_run` 控制动作，支持 `flow_id` + `run_id`
- `cancel_run` 仅允许命中目标 flow 的活动 run；不存在或已结束时返回明确错误
- 取消后，run 状态必须转为 `cancelled`
- `status/detail` 查询该 run 时必须可见取消状态与原因
- 协议和文档需要明确 `cancel_run` 与 `delete` 的边界：
  - `delete` 仍表示删除定义并中断关联 run
  - `cancel_run` 仅中断指定 run，不删除 flow 定义
- 需要形成后续顺序执行 checklist：
  - `RC-P0-1`：显式 `cancel_run`
  - `RC-P0-2`：`list_runs/history`
  - `RC-P0-3`：`flow.run` / `flow.read` 权限细化
  - `RC-P0-4`：协议映射与索引同步
  - `RC-P1-*`：重试策略、并发/重入控制、触发去重、运行归档
  - `RC-P2-*`：`branch` / `foreach` / `subflow` / `cron`

#### Non-functional Requirements
- 保持跨仓改动面最小，不顺手扩展 history/permissions
- 取消语义必须明确、可审计，不能静默吞掉“已结束 run”
- 文档真相必须先回到 `MyFlowHub-Server/docs`
- 不破坏现有 `delete`、`run`、`status`、`detail` 语义

#### Inputs / Outputs
- 输入：
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\protocol\flow\types.go`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\handler.go`
  - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
- 输出：
  - 可执行清单
  - `cancel_run` 相关 docs / protocol / runtime / tests

#### Edge Cases
- `run_id` 不存在
- `run_id` 属于其他 `flow_id`
- run 已 `succeeded/failed/cancelled`
- run 正在等待远程/本地节点执行，取消需尽快反映到状态查询
- `cancel_run` 与 `delete` 并发发生时，取消原因需要保持一致口径

#### Acceptance Criteria
1. 可执行清单具备明确优先级与任务边界
2. `cancel_run` 可取消活动 run，且不删除 flow 定义
3. `status/detail` 对被取消 run 返回 `cancelled`
4. requirements/specs/Proto/SubProto 测试和文档均同步
5. `docs/specs/protocol_map.md` 反映当前 Flow 稳定动作集合

#### Risks
- 若直接在现有 `delete` 取消逻辑上拼接，可能导致取消原因和状态边界混乱
- 若要求 `run_id` 全局唯一但不校验 `flow_id` 归属，容易引入跨 flow 误取消
- 若先改 Proto/实现而未同步 Server docs，会继续扩大契约漂移

#### Issue List
- none

### Stage 3.2 - Implementation
#### Executed Work
- `RC-P0-1` 已完成：
  - `Server docs`：补齐 `cancel_run` requirements/specs
  - `Proto`：新增 `ActionCancelRun` / `CancelRunReq` / `CancelRunResp`
  - `SubProto`：新增 `handleCancelRun`、取消 helper、`status/detail` 取消可见性与测试
  - `workspace docs`：已同步 `D:\project\MyFlowHub3\docs\specs\protocol_map.md`

#### Implementation Notes
- `cancel_run` 维持最小契约：
  - 不删除 flow 定义
  - 已结束 run 返回 `409`
  - 不新增新的权限常量
- `detail` 不扩顶层 run 状态字段，改为通过 `msg` 与活动节点状态反映取消结果

#### Blockers
- none

### Stage 3.3 - Review
#### Checklist
- requirements/specs 已与实现同步
- Proto canonical `docs/protocol_map.md` 已重新生成
- SubProto `cancel_run` 成功 / 404 / 409 / invalid id / remote forward failure 已覆盖
- delete 基线在临时 `go.work` 联测下保持通过
- root workspace `docs/specs/protocol_map.md` 已与 Proto canonical 副本同步

#### Verification
- `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
- `D:\project\MyFlowHub3`
  - `GOWORK=D:\project\MyFlowHub3\.tmp\verify-run-control-phase1\go.work go test github.com/yttydcs/myflowhub-subproto/flow/... -count=1 -p 1`

#### Review Result
- 通过

### Stage 4 - Archive
#### `$m-docs` Archive Output
- `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\change\2026-04-02_server-flow-cancel-run-contract.md`
- `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\change\2026-04-02_proto-flow-cancel-run.md`
- `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\docs\change\2026-04-02_flow-cancel-run-runtime.md`

#### Lessons Routing
- lessons impact: none

#### Next Iteration Entry Point
- 若继续顺序执行，下一项为 `RC-P0-2: list_runs/history`

### Next Iteration - RC-P0-2 list_runs/history
#### Stage 1 - Requirements Analysis
- Goal:
  - 为保留窗口内历史 run 提供独立查询入口，避免调用方只能通过 `status(latest)` 或 `list(last_run_id)` 看到最新一条。
- Scope:
  - 必须：
    - 新增 `list_runs` 动作，按 `flow_id` 返回当前保留窗口内的 run 摘要
    - 补齐 requirements/specs、Proto 契约、SubProto 实现与测试
    - 同步 Proto canonical `protocol_map` 及 root workspace 副本
  - 不做：
    - 不做全局 archive / 持久化 run history
    - 不做跨 flow 聚合历史查询
    - 不做新的权限常量
    - 不把完整节点结果并入 history 摘要
- Functional Requirements:
  - `list_runs` 必须要求 `flow_id`
  - `list_runs` 命中时返回该 flow 当前保留窗口内的 run 摘要，顺序为最新到最旧
  - 请求可带 `limit`；为空或非法时使用全部保留窗口
  - 每条 run 摘要至少包含：
    - `run_id`
    - `status`
    - `started_at`
    - `ended_at`
    - `msg`（取消原因等，best-effort）
  - `flow_id` 不存在时返回 `404`
  - `status/detail` 现有语义不能回退
- Non-functional Requirements:
  - 复用现有 `runOrderByFlow` 索引，避免新扫描路径
  - 不改变当前 run 回收策略
  - 改动面保持在最小稳定契约，不顺手扩成 archive 系统
- Acceptance:
  - 调用方可按 `flow_id` 看到最近 run 列表
  - 列表顺序与 retention 语义一致
  - `cancelled` run 在 history 中可见其取消说明
  - requirements/specs/Proto/SubProto/tests 全同步

#### Stage 2 - Architecture Design
- Overall Solution:
  - 新增独立 `list_runs` / `list_runs_resp`
  - 运行时直接复用 `runOrderByFlow[flowID]`，按逆序组装 `runSummary`
- Interface Draft:
  - request:
    - `req_id`
    - `origin_node`
    - `executor_node`
    - `flow_id`
    - `limit`（可选）
  - response:
    - `req_id`
    - `code`
    - `msg`
    - `executor_node`
    - `flow_id`
    - `runs`
  - `runSummary`:
    - `run_id`
    - `status`
    - `started_at`
    - `ended_at`
    - `msg`
- Error Handling:
  - `flow_id` 非法：`400`
  - `limit <= 0`：视为未设置
  - flow 不存在：`404`
  - 远端无路由/转发失败：沿用现有 `*_resp` 失败响应模式
- Testing Strategy:
  - 多 run 顺序与 limit 覆盖
  - `cancel_run` 后 history 可见取消原因
  - flow not found / invalid flow_id / remote forward failure

#### Stage 3.1 - Planning
- `$m-docs` impact:
  - Requirements impact: `update`
  - Specs impact: `update`
- Canonical destinations:
  - stable requirements -> `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
  - stable specs -> `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
  - Proto canonical map -> `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md`
  - workspace sync copy -> `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
- Task Mapping:
  - `RC-P0-2`
    - `Server docs`: requirements/specs for `list_runs`
    - `Proto`: action/types + protocol map regeneration
    - `SubProto`: action registration, handler, run summary helper, tests
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\protocol\flow\types.go`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\actions.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\types.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\handler.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\runtime_fix_test.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\flow_id_test.go`
  - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
- Parallelism Assessment:
  - 不使用子Agent
  - 原因：docs / Proto / runtime / tests 强耦合，共享一组 action/type 变更
- Blockers:
  - none

#### Stage 3.2 - Implementation
- `RC-P0-2` 已完成：
  - `Server docs`：补齐 `list_runs` requirements/specs
  - `Proto`：新增 `ActionListRuns` / `ListRunsReq` / `ListRunsResp` / `RunSummary`
  - `SubProto`：新增 `handleListRuns`、retained run summary helper 和测试
  - `workspace docs`：已同步 `D:\project\MyFlowHub3\docs\specs\protocol_map.md`

#### Stage 3.3 - Review
- Checklist:
  - requirements/specs 已与实现同步
  - Proto canonical `docs/protocol_map.md` 已重新生成
  - `list_runs` 顺序 / limit / invalid flow_id / not found / remote forward failure 已覆盖
  - 全量 `flow` 模块在临时 `go.work` 联测下通过
- Verification:
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1`
    - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - `D:\project\MyFlowHub3`
    - `GOWORK=D:\project\MyFlowHub3\.tmp\verify-run-control-phase2\go.work go test github.com/yttydcs/myflowhub-subproto/flow/... -count=1 -p 1`
- Review Result:
  - 通过

#### Stage 4 - Archive
- `$m-docs` Archive Output:
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\change\2026-04-02_server-flow-list-runs-contract.md`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\change\2026-04-02_proto-flow-list-runs.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\docs\change\2026-04-02_flow-list-runs-runtime.md`
- Lessons Routing:
  - lessons impact: none
- Next Iteration Entry Point:
  - 若继续顺序执行，下一项为 `RC-P0-3: flow.run / flow.read 权限细化`

### Stage 2 - Architecture Design
#### Overall Solution
- 采用“先补显式 run control，再补观测与权限”的分阶段方案：
  - 当前 iteration 只落地 `cancel_run`
  - 后续 iteration 再扩展 `list_runs/history`、细粒度权限与更高阶编排节点

#### Alternatives Considered
- 备选：先做 `list_runs/history`
- 不采用原因：
  - 没有显式取消能力时，run 生命周期仍缺关键控制面，history 先做价值有限
- 备选：直接补 `flow.run` / `flow.read` 权限
- 不采用原因：
  - 需要先明确 run control 动作集合，否则权限边界会二次改动
- 备选：把取消继续绑定在 `delete`
- 不采用原因：
  - `delete` 删除定义，`cancel_run` 只中断 run，语义不同，不能继续混用

#### Module Responsibilities
- `MyFlowHub-Server/docs`
  - 维护长期 requirements/specs 真相
- `MyFlowHub-Proto/protocol/flow`
  - 新增 `cancel_run` 契约与载荷类型
- `MyFlowHub-Proto/docs/protocol_map.md`
  - 作为 canonical 生成文档，承载 Flow 动作映射同步
- `MyFlowHub-SubProto/flow`
  - 实现 `cancel_run` handler、状态转换和测试
- `workspace docs/specs/protocol_map.md`
  - 同步最新稳定 Flow 动作/权限副本

#### Data / Call Flow
- caller -> `flow.cancel_run`
- executor 路由到目标执行者
- 执行者校验 `flow_id/run_id`
- 命中活动 run -> 触发 run context cancel -> 更新 `runState.status=cancelled`
- `status/detail` 读取同一 `runState`，返回取消状态与原因

#### Interface Drafts
- 新增 action:
  - `cancel_run`
  - `cancel_run_resp`
- 新增请求：
  - `req_id`
  - `origin_node`
  - `executor_node`
  - `flow_id`
  - `run_id`
- 新增响应：
  - `req_id`
  - `code`
  - `msg`
  - `flow_id`
  - `run_id`
  - `executor_node`
  - `status`

#### Error Handling and Safety
- `run_id` 缺失或非法：`400`
- run 未命中：`404`
- run 已结束：明确返回 `409` 或等价明确错误码/文案
- 取消只允许作用于目标 run，不批量影响其他 run
- 不改变 `delete` 的删除定义语义

#### Performance and Testing Strategy
- 复用现有 `runState.cancel`，避免引入额外轮询
- 测试重点：
  - 活动 run 成功取消
  - 已结束 run 拒绝取消
  - `flow_id/run_id` 归属校验
  - `status/detail` 读取已取消 run
  - 路由/远程转发场景最小覆盖

#### Extensibility Design Points
- `cancel_run` 契约与状态字段为后续 `list_runs/history` 复用
- 当前 action 命名保留与未来 `pause_run/resume_run` 并列扩展空间
- checklist 已预留后续权限与 history 项，不把一次性大改绑死在首轮

### Stage 3.1 - Planning
#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: `add`
- Specs impact: `add`
- Canonical destination:
  - 稳定行为边界 -> `repo/MyFlowHub-Server/docs/requirements/flow_data_dag.md`
  - 稳定技术契约 -> `repo/MyFlowHub-Server/docs/specs/flow.md`
  - 单次 workflow 结果 -> Stage 4 `docs/change`
  - protocol 副本同步 -> `docs/specs/protocol_map.md`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md`
  - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
- Related lessons:
  - 无

#### Project Goal and Current State
- 目标：把 Flow 补强建议收敛成顺序 checklist，并从 `cancel_run` 起步
- 当前状态：
  - `run/status/detail` 已存在，但无显式 `cancel_run`
  - run 取消只在 `delete flow` 时内部发生
  - `docs/specs/protocol_map.md` 仍未完全反映主线 Flow 动作集合

#### Ordered Execution Checklist
- `RC-P0-1`：显式 `cancel_run` 控制能力
- `RC-P0-2`：`list_runs/history` 查询能力
- `RC-P0-3`：`flow.run` / `flow.read` 权限细化
- `RC-P0-4`：协议映射与索引同步收口
- `RC-P1-1`：retry backoff / policy
- `RC-P1-2`：并发/重入控制
- `RC-P1-3`：trigger 去重与运行归档
- `RC-P2-1`：`branch`
- `RC-P2-2`：`foreach`
- `RC-P2-3`：`subflow`
- `RC-P2-4`：`cron`

#### Task Details
##### RC-P0-1 - explicit cancel_run
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\todo.md`
- Goal: 增加取消指定 run 的稳定控制面
- Files / Modules:
  - `Proto`: `protocol/flow/types.go`
  - `Proto`: `docs/protocol_map.md`
  - `SubProto`: `flow/actions.go`, `flow/types.go`, `flow/handler.go`, `flow/*test.go`
  - `Server docs`: `docs/requirements/flow_data_dag.md`, `docs/specs/flow.md`
  - `workspace docs`: `docs/specs/protocol_map.md`
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\protocol\flow\types.go`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\actions.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\types.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\handler.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\delete_test.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\runtime_fix_test.go`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
- Acceptance:
  - 可显式取消指定活动 run
  - 已结束 run 不被错误取消
  - `status/detail` 能反映 `cancelled`
- Test Points:
  - 新增 `cancel_run` 成功 / not found / terminal-state 用例
  - 相关 `status/detail` 用例
  - 目标 `go test github.com/yttydcs/myflowhub-subproto/flow/... -count=1 -p 1`
- Rollback:
  - 回退 `cancel_run` 契约、handler、测试与 docs 变更

##### RC-P0-2 - list_runs/history
- Owner: main agent
- Goal: 为保留窗口内历史 run 提供查询入口
- Status: pending after `RC-P0-1`

##### RC-P0-3 - permission refinement
- Owner: main agent
- Goal: 为 `run/status/detail/list/get` 建立明确权限边界
- Status: completed and archived

##### RC-P0-4 - protocol map / indexes
- Owner: main agent
- Goal: 收口 workspace 协议副本和相关索引
- Status: completed and archived

#### Dependencies
- `cancel_run` 契约必须先在 `Proto` 定义
- `SubProto` 实现依赖最新 Proto worktree
- `Server docs` 必须先或同时更新，不能让实现跑在文档前面

#### Risks and Notes
- 根仓当前存在其他未提交文档改动；后续归档时只能最小暂存本次内容
- `docs/specs/protocol_map.md` 位于 workspace 根，不属于执行 worktree；仅允许作为 docs 同步例外在控制面收敛时修改

#### Parallelism Assessment
- 不使用子Agent
- 原因：
  - Proto / docs / runtime 改动强耦合，需要主代理统一保持契约一致
  - 当前关键路径是先收敛第一项 `cancel_run`

#### Issue List
- none

阻塞：否
当前状态：`RC-P0-3` 已完成实现与归档，下一项为 `RC-P0-4`

### Next Iteration - RC-P0-3 flow.run / flow.read 权限细化
#### Stage 1 - Requirements Analysis
- Goal:
  - 为 `flow` 的运行控制面和只读观测面建立稳定、独立的权限边界，并避免新增校验后把当前默认 `admin/node` 行为回退成 `403`。
- Scope:
  - 必须：
    - 新增稳定权限常量：
      - `flow.run`
      - `flow.read`
    - 明确动作与权限映射：
      - `run` / `cancel_run` -> `flow.run`
      - `status` / `detail` / `list_runs` / `list` / `get` -> `flow.read`
      - `flow::run` capability descriptor -> `flow.run`
    - 补齐 `Server docs`、`Proto`、`SubProto`、`Core`、`Server hubruntime` 默认值与测试
    - 同步 Proto canonical `docs/protocol_map.md` 与 root workspace 副本
  - 可选：
    - 若实现中发现文案重复明显，可顺手压缩说明，但不扩成新的权限模型
  - 不做：
    - 不把权限继续细拆到每个 action 一个常量
    - 不改变 `flow.set` / `flow.delete` 现有语义
    - 不引入新的角色继承、scope 或 capability 调用身份模型
- Use Cases:
  - 拥有 `flow.run` 的调用方可以手动触发 run、取消活动 run，并通过 `exec.call` 调用 `flow::run` capability。
  - 拥有 `flow.read` 的调用方可以读取 flow 定义、当前状态和 retained runs，而不必同时具备写权限。
  - 默认 `admin/node` 在未显式覆盖 `auth.role_perms` 的开箱配置下，继续保持当前可运行和可观测行为。
- Functional Requirements:
  - `flow.run` 必须成为稳定协议权限常量，并用于：
    - `run`
    - `cancel_run`
    - `flow::run`
  - `flow.read` 必须成为稳定协议权限常量，并用于：
    - `status`
    - `detail`
    - `list_runs`
    - `list`
    - `get`
  - `SubProto flow` 在 `executor==local` 与 `LCA` 裁决点都必须显式判权；来自父节点的已授权下行请求继续直接执行/转发。
  - 默认角色权限必须补齐，至少保证：
    - `admin` 包含 `flow.run` / `flow.read`
    - `node` 包含 `flow.run` / `flow.read`
  - 新增权限后，既有 `400/404/409/500` 语义不能被权限回退掩盖；无权限场景应返回明确 `403`。
- Non-functional Requirements:
  - 采用最小安全改动，优先复用现有逐级授权 / 转发路径
  - 不允许 `Core` 与 `Server hubruntime` 默认角色字符串继续漂移
  - capability descriptor 与 action handler 的权限口径必须一致
- Inputs / Outputs:
  - 输入：
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\auth.md`
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\hubruntime\options.go`
    - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\protocol\flow\types.go`
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\handler.go`
    - `D:\project\MyFlowHub3\worktrees\core-run-control-phase1\config\config.go`
  - 输出：
    - `flow.run` / `flow.read` 稳定 docs + protocol + runtime + defaults + tests
- Edge Cases:
  - `executor` 不可达时，权限细化不能改变现有 `404/forward failed` 返回路径
  - 来自父节点的下行帧不能重复判权，否则会破坏既有“父已裁决”假设
  - `flow::run` capability 必须暴露权限描述，否则 `exec.cap.query/snapshot` 无法反映真实授权要求
  - 若只修改 `Core` 而不修改 `Server hubruntime`，hub 默认值仍会漂移
- Acceptance Criteria:
  1. `run/cancel_run` 需要 `flow.run`
  2. `status/detail/list_runs/list/get` 需要 `flow.read`
  3. `flow::run` capability descriptor 暴露 `flow.run`
  4. 默认 `admin/node` 在 Core 与 Server runtime 中继续具备上述新权限
  5. requirements/specs/Proto/SubProto/Core/Server tests 全同步
- Risks:
  - 若只在 executor 本地判权，会绕过 LCA 统一裁决口径
  - 若只改 Proto / SubProto 而不改 Core / Server 默认值，当前默认部署会出现权限回退
  - 若引入三套不同 helper，后续权限扩展会继续复制粘贴

#### Stage 2 - Architecture Design
- Overall Solution:
  - 在 Proto 中新增 `PermFlowRun` / `PermFlowRead` 常量。
  - 在 SubProto 中把当前 `forwardToExecutorNoPerm(...)` 升级为带可选权限的统一 helper，复用 `set/delete` 现有逐级上报与父节点信任模型。
  - 在 Core 与 Server runtime 默认角色映射中同步补齐 `flow.run` / `flow.read`，维持开箱行为不回退。
- Alternatives Considered:
  - 备选：只在 executor 本地增加 `hasPermission(...)`
  - 不采用原因：
    - 会绕开既有 LCA 裁决模型
    - 与 `flow.set` / `flow.delete` 的治理口径不一致
  - 备选：把 `run/cancel_run` 继续视为无权限动作，仅给 `read` 加权限
  - 不采用原因：
    - 运行控制面本身就是敏感操作，不能继续裸奔
  - 备选：为 `run`、`cancel_run`、`status`、`detail` 等分别拆成多个常量
  - 不采用原因：
    - 当前 checklist 目标是最小稳定权限收口，不是做全量权限矩阵设计
- Module Responsibilities:
  - `MyFlowHub-Server/docs`
    - 维护 `flow.run` / `flow.read` 的长期 requirements/specs 真相
  - `MyFlowHub-Server/hubruntime`
    - 维护 Hub 默认 `auth.role_perms`，避免与 Core 默认值漂移
  - `MyFlowHub-Proto/protocol/flow`
    - 定义新权限常量并驱动 canonical protocol map
  - `MyFlowHub-SubProto/flow`
    - 为 run/read 动作接入显式判权与 capability descriptor 权限
  - `MyFlowHub-Core/config`
    - 维护基础默认角色权限字符串
- Data / Call Flow:
  - caller -> `flow.run/status/...`
  - 请求沿当前 `forward` 路径上送 / 下送
  - 到达本地 executor 或 LCA 时：
    - `flow.run` / `flow.read` 显式判权
    - 判权通过后向下转发或本地执行
  - `exec.call(flow::run)`:
    - capability descriptor 暴露 `flow.run`
    - `exec` 本地 capability 权限检查复用 descriptor
- Interface Drafts:
  - Proto constants:
    - `PermFlowRun = "flow.run"`
    - `PermFlowRead = "flow.read"`
  - SubProto helper:
    - 统一路由 helper 增加 `perm string` 参数；空串表示无需额外权限
  - 无新增 payload 字段
- Error Handling and Safety:
  - 无权限统一返回 `403 permission denied`
  - 继续保留：
    - invalid request -> `400`
    - missing executor / route -> `404/500`
    - terminal run -> `409`
  - 来自父节点的请求继续视为已授权，避免重复裁决
- Performance and Testing Strategy:
  - 复用现有转发 helper，避免新建多份 routing 分支
  - 测试重点：
    - `run` / `cancel_run` permission denied
    - `status/detail/list_runs/list/get` permission denied
    - remote forward failure 不回退
    - capability descriptor 含 `flow.run`
    - Core / Server 默认角色字符串锁定
- Extensibility Design Points:
  - 后续若需要继续细拆 `flow.read.detail` 等权限，可在统一 helper 上扩展，不必重写路由模型
  - `flow::run` 先跟随 `flow.run`，为后续更多 flow capability 暴露保留一致入口

#### Stage 3.1 - Planning
- Docs Governance Routing Decision:
  - 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
  - Requirements impact: `clarify`
  - Specs impact: `clarify`
  - Canonical destination:
    - 稳定需求边界 -> `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
    - 稳定技术契约 -> `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
    - 默认角色层级 -> `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\auth.md`
    - Proto canonical map -> `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md`
    - workspace sync copy -> `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
    - workflow results -> Stage 4 `docs/change`
  - Related requirements:
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
  - Related specs:
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\auth.md`
    - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md`
    - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
  - Related lessons:
    - `D:\project\MyFlowHub3\docs\change\2026-03-26_auth-default-role-hierarchy.md`
    - `D:\project\MyFlowHub3\docs\change\2026-04-02_flow-delete-permission-baseline.md`
- Task Mapping:
  - `RC-P0-3`
    - `Server docs`: requirements/specs/auth 更新
    - `Server runtime`: `hubruntime` 默认角色权限更新
    - `Proto`: new perms + protocol map regeneration
    - `SubProto`: route helper / handlers / capability descriptor / tests
    - `Core`: default role perms update
    - `workspace docs`: sync `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\auth.md`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\hubruntime\options.go`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\hubruntime\options_test.go`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\hubruntime\layered_config_test.go`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\protocol\flow\types.go`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\types.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\handler.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\capability_provider_test.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\runtime_fix_test.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\delete_test.go`
  - `D:\project\MyFlowHub3\worktrees\core-run-control-phase1\config\config.go`
  - `D:\project\MyFlowHub3\worktrees\core-run-control-phase1\config\config_test.go`
  - `D:\project\MyFlowHub3\worktrees\core-run-control-phase1\todo.md`
  - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
- Parallelism Assessment:
  - 不使用子Agent
  - 原因：
    - docs / Proto / SubProto / Core / Server runtime 共用同一组权限语义和默认值
    - `hubruntime` 默认值与 Core 常量存在耦合，拆开会增加回归风险
- Dependencies:
  - `Proto` 新权限常量必须先定义
  - `SubProto` handler 与 capability descriptor 依赖最新 Proto 常量
  - `Core` 与 `Server hubruntime` 默认值必须一起改，避免默认配置漂移
- Risks and Notes:
  - `Server hubruntime` 当前仍复制默认角色字符串，不能只改 Core
  - root workspace `docs/specs/protocol_map.md` 仍是 control-plane 例外路径，只允许同步当前稳定副本
- Issue List:
  - none

阻塞：否
进入 3.2

#### Stage 3.2 - Implementation
- `RC-P0-3` 已完成：
  - `Server docs`：
    - `docs/requirements/flow_data_dag.md`
    - `docs/specs/flow.md`
    - `docs/specs/auth.md`
    - 明确 `flow.run` / `flow.read` 稳定权限、动作映射和默认角色要求
  - `Proto`：
    - `protocol/flow/types.go`
    - 新增 `PermFlowRun` / `PermFlowRead`
    - 重新生成 `docs/protocol_map.md`
  - `SubProto`：
    - `flow/types.go`
    - `flow/handler.go`
    - `flow/capability_provider_test.go`
    - `flow/delete_test.go`
    - `run/cancel_run` 接入 `flow.run`
    - `status/detail/list_runs/list/get` 接入 `flow.read`
    - `flow::run` capability descriptor 暴露 `flow.run`
  - `Core`：
    - `config/config.go`
    - `config/config_test.go`
    - 默认 `admin/node` 补齐 `flow.run` / `flow.read`
  - `Server runtime`：
    - `hubruntime/options.go`
    - `hubruntime/options_test.go`
    - `defaultAuthRolePerms` 改为直接引用 `coreconfig.DefaultAuthRolePerms`
  - `workspace docs`：
    - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
    - 已同步 Proto canonical 副本

#### Stage 3.3 - Review
- Checklist:
  - 需求覆盖：通过
    - `flow.run` / `flow.read` 的使用边界、默认角色连续性与 capability 权限描述均已落实
  - 架构合理性：通过
    - 沿用现有逐级授权 / 父节点信任模型，仅在统一 helper 上增加可选权限参数
  - 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
    - 未新增新的持久化或全表扫描路径
  - 可读性与一致性：通过
    - run/read 动作统一走同一 permission-aware helper
  - 可扩展性与配置化：通过
    - Core 与 Server runtime 默认值已收敛到同一来源
  - 稳定性与安全：通过
    - 无权限场景显式返回 `403`
    - 父节点已授权下行请求继续直接执行 / 转发
  - 测试覆盖情况：通过
    - capability 权限描述、run/read permission denied、Core 默认值、Server runtime 默认值均有测试
  - 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
    - 本轮未使用子Agent
- Verification:
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1`
    - `$env:GOWORK='off'; go run ./cmd/protocolmapgen -write -out docs/protocol_map.md`
    - `$env:GOWORK='off'; go run ./cmd/protocolmapgen -check -out docs/protocol_map.md`
    - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - `D:\project\MyFlowHub3`
    - `GOWORK=D:\project\MyFlowHub3\.tmp\verify-run-control-phase3\go.work go test github.com/yttydcs/myflowhub-subproto/flow/... -count=1 -p 1`
    - `GOWORK=D:\project\MyFlowHub3\.tmp\verify-run-control-phase3\go.work go test github.com/yttydcs/myflowhub-core/... -count=1 -p 1`
    - `GOWORK=D:\project\MyFlowHub3\.tmp\verify-run-control-phase3\go.work go test github.com/yttydcs/myflowhub-server/hubruntime/... -count=1 -p 1`
- Review Result:
  - 通过

#### Stage 4 - Archive
- 使用 `$m-docs` 校验归档路由、requirements/specs 影响和索引更新。
- `$m-docs` Archive Output:
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\change\2026-04-02_server-flow-permission-refinement.md`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\change\2026-04-02_proto-flow-permission-refinement.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\docs\change\2026-04-02_flow-permission-refinement-runtime.md`
  - `D:\project\MyFlowHub3\worktrees\core-run-control-phase1\docs\change\2026-04-02_core-flow-permission-defaults.md`
- Index Updates:
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\change\README.md`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\change\README.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\docs\change\README.md`
  - `D:\project\MyFlowHub3\worktrees\core-run-control-phase1\docs\change\README.md`
- Lessons Routing:
  - Lessons impact: `none`
  - 原因：
    - 本轮没有新增超出既有经验库的排障模式
    - 默认角色漂移排查线索已由 `D:\project\MyFlowHub3\docs\change\2026-04-02_flow-delete-permission-baseline.md` 覆盖
- Next Iteration Entry Point:
  - 若继续顺序执行，下一项为 `RC-P0-4: 协议映射与索引同步收口`

### Next Iteration - RC-P0-4 protocol map / indexes convergence
#### Stage 1 - Requirements Analysis
- Goal:
  - 在不结束整个 workflow 的前提下，收口 workspace 根级 `protocol_map` 同步副本和相关入口索引，让当前 `flow` 新动作与权限补强可以从根文档体系被正确发现。
- Scope:
  - 必须：
    - 确认 `D:\project\MyFlowHub3\docs\specs\protocol_map.md` 与 Proto canonical `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md` 保持一致
    - 根级入口文档明确：
      - canonical source-of-truth 在 `MyFlowHub-Proto`
      - workspace `protocol_map` 只是同步副本
      - 不在 workspace 根直接运行 `protocolmapgen` 或手改 generated block
    - 更新受影响索引，使本轮 convergence 文档可从根级 `docs/` 被发现
    - 保持“worktree-local change 归档在 workflow 结束前不搬入全局目录”的边界
  - 可选：
    - 若根级索引中存在与当前 `protocol_map` 路由冲突的旧表述，可做最小澄清
  - 不做：
    - 不再次修改 Proto canonical `protocol_map` 内容
    - 不再次修改 `Server` 稳定 specs 真相
    - 不提前结束 workflow，也不把 worktree-local `docs/change/*` 批量迁回全局 `docs/change/`
- Use Cases:
  - 接手者从 workspace 根 `docs/README.md` / `docs/specs/README.md` 进入时，能知道 Flow 当前协议映射应看哪里、如何更新、哪里是同步副本。
  - 调整 `flow` 动作或权限后，控制面文档不会继续误导读者在错误目录本地生成或手改副本。
  - workspace 级 change index 能看到本轮 protocol/index convergence 的审计记录。
- Functional Requirements:
  - 根级 `docs/specs/README.md` 必须明确 `protocol_map.md` 的 canonical source 和 sync 策略
  - 根级 `docs/README.md` 必须明确 `protocol_map` 的更新顺序
  - 根级 `docs/change/README.md` 必须收录本轮 convergence 归档
  - 若 root sync copy 与 Proto canonical 已无 diff，必须显式记录“已验证一致，无需内容修改”
- Non-functional Requirements:
  - 仅做最小 docs convergence 改动
  - 不破坏 `protocol_map.md` generated block 保护边界
  - 保持 root docs 作为控制面入口，不把它变成第二真源
- Inputs / Outputs:
  - 输入：
    - `D:\project\MyFlowHub3\docs\README.md`
    - `D:\project\MyFlowHub3\docs\specs\README.md`
    - `D:\project\MyFlowHub3\docs\change\README.md`
    - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
    - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md`
  - 输出：
    - 根级 docs entry/index 收口
    - `RC-P0-4` convergence 归档
- Edge Cases:
  - root `protocol_map` 已与 canonical 一致，此时不能为了“做改动”而手改 generated block
  - 根仓仍是控制面目录，只允许 workflow convergence 例外编辑，不写业务实现
  - 当前 workflow 尚未结束，worktree-local archive 仍保留在各自 worktree 中
- Acceptance Criteria:
  1. root `protocol_map` 与 Proto canonical 一致，或已完成整文件同步
  2. 根级 `docs/README.md` 与 `docs/specs/README.md` 明确 sync-copy 规则
  3. 根级 `docs/change/README.md` 收录本轮 convergence 文档
  4. requirements/specs 真相未被重复复制到新的位置
- Risks:
  - 若在 root `protocol_map` 直接手改，会破坏 whole-file sync 约束
  - 若把 worktree-local change 提前搬进全局 `docs/change/`，会打乱 workflow 结束时的归档步骤
  - 若入口索引仍未说明 canonical source，后续接手者会继续在错误目录更新协议映射

#### Stage 2 - Architecture Design
- Overall Solution:
  - 把 `RC-P0-4` 收敛为一次控制面 docs convergence：
    1. 先比对 Proto canonical 与 root sync copy
    2. 若无 diff，只记录验证结果
    3. 在根级入口 README 中补 canonical / sync / maintenance 规则
    4. 新增一份 workspace 级 `docs/change` 归档并更新根级索引
- Alternatives Considered:
  - 备选：直接手改 root `docs/specs/protocol_map.md` 顶部说明
  - 不采用原因：
    - 会打破“整文件同步副本”规则
    - 后续再同步时仍会被 canonical 覆盖
  - 备选：等 workflow 结束再一起处理根级索引
  - 不采用原因：
    - 当前根级入口已经需要能正确指向本轮 Flow 协议补强后的文档路径
- Module Responsibilities:
  - `workspace docs root`
    - 维护入口、索引与 control-plane 同步规则
  - `Proto canonical protocol_map`
    - 继续作为协议映射单一真源
  - `subproto-run-control-phase1/todo.md`
    - 记录本轮 RC-P0-4 的 requirements/design/plan/review/archive
- Data / Call Flow:
  1. 读取 root `docs/README.md` / `docs/specs/README.md`
  2. 比对 root `docs/specs/protocol_map.md` 与 Proto canonical
  3. 若需要，执行整文件同步；若不需要，记录 no-diff
  4. 更新根级索引与 convergence 归档
- Interface Drafts:
  - 不新增运行时接口
  - docs rule:
    - canonical -> `MyFlowHub-Proto/docs/protocol_map.md`
    - sync copy -> `MyFlowHub3/docs/specs/protocol_map.md`
- Error Handling and Safety:
  - 不编辑 generated block
  - 不把 root docs 描述成第二真源
  - control-plane 例外编辑必须在归档中说明
- Performance and Testing Strategy:
  - 以文档 diff/self-review 为主
  - 验证：
    - `git diff --no-index` 比对 canonical 与 sync copy
    - root `git diff --check`
- Extensibility Design Points:
  - 后续新增协议动作时，继续遵循“先 Proto canonical，再 root sync copy，再更新入口索引”的固定顺序

#### Stage 3.1 - Planning
- Docs Governance Routing Decision:
  - 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
  - Requirements impact: `none`
  - Specs impact: `none`
  - Canonical destination:
    - stable protocol mapping truth -> `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md`
    - workspace sync copy -> `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
    - root entry / index maintenance -> `D:\project\MyFlowHub3\docs\README.md`, `D:\project\MyFlowHub3\docs\specs\README.md`, `D:\project\MyFlowHub3\docs\change\README.md`
    - workflow convergence archive -> `D:\project\MyFlowHub3\docs\change\2026-04-02_flow-protocol-map-and-index-sync.md`
  - Related requirements:
    - none
  - Related specs:
    - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
    - `D:\project\MyFlowHub3\docs\specs\README.md`
    - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md`
  - Related lessons:
    - `D:\project\MyFlowHub3\docs\change\2026-02-20_protocol-mapgen.md`
- Task Mapping:
  - `RC-P0-4`
    - root `protocol_map` sync-copy verification
    - root docs entry/index clarification
    - workspace-level convergence archive and index update
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\todo.md`
  - `D:\project\MyFlowHub3\docs\README.md`
  - `D:\project\MyFlowHub3\docs\specs\README.md`
  - `D:\project\MyFlowHub3\docs\change\README.md`
  - `D:\project\MyFlowHub3\docs\change\2026-04-02_flow-protocol-map-and-index-sync.md`
  - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
- Parallelism Assessment:
  - 不使用子Agent
  - 原因：
    - 改动都是控制面 docs，写集高度重叠
    - 需要主代理统一判断哪些 root docs 允许作为 convergence 例外修改
- Dependencies:
  - `RC-P0-1` ~ `RC-P0-3` 的 Proto canonical 和 root sync copy 已完成同步
  - root docs 索引更新不能与 workflow 结束时的归档搬运步骤混淆
- Risks and Notes:
  - root docs 编辑属于 control-plane workflow convergence 例外，需要在归档中说明
  - 若 root `protocol_map` 已无 diff，本轮不强行修改其正文
- Issue List:
  - none

阻塞：否
进入 3.2

#### Stage 3.2 - Implementation
- `RC-P0-4` 已完成：
  - `root docs entry`：
    - `D:\project\MyFlowHub3\docs\README.md`
    - 明确 `protocol_map` 是从 `MyFlowHub-Proto` 同步来的 workspace 副本，不能在 workspace 根直接生成
  - `root specs index`：
    - `D:\project\MyFlowHub3\docs\specs\README.md`
    - 明确 canonical / sync-copy 关系与正确更新顺序
  - `root change index`：
    - `D:\project\MyFlowHub3\docs\change\README.md`
    - 收录本轮 convergence 归档
  - `workspace change archive`：
    - `D:\project\MyFlowHub3\docs\change\2026-04-02_flow-protocol-map-and-index-sync.md`
  - `root protocol_map copy`：
    - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
    - 已与 Proto canonical 比对，本轮无需正文改动

#### Stage 3.3 - Review
- Checklist:
  - 需求覆盖：通过
    - root docs 已明确 canonical source / sync-copy / maintenance 边界
  - 架构合理性：通过
    - 保持 `protocol_map` whole-file sync copy，不引入第二真源
  - 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
    - 本轮仅 docs convergence，无运行时代码改动
  - 可读性与一致性：通过
    - root `docs/README.md`、`docs/specs/README.md` 与 `docs/change/README.md` 口径一致
  - 可扩展性与配置化：通过
    - 后续新增协议动作仍能沿用“Proto canonical -> root sync copy -> index update”的固定顺序
  - 稳定性与安全：通过
    - 未改 `protocol_map.md` generated block
    - 未提前搬运 worktree-local archive
  - 测试覆盖情况：通过
    - 已完成 root/proto `protocol_map` no-diff 验证和 root `git diff --check`
  - 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
    - 本轮未使用子Agent
- Verification:
  - `D:\project\MyFlowHub3`
    - `git diff --no-index --quiet D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md D:\project\MyFlowHub3\docs\specs\protocol_map.md`
    - `git diff --check`
- Review Result:
  - 通过

#### Stage 4 - Archive
- 使用 `$m-docs` 校验归档路由、requirements/specs 影响和索引更新。
- `$m-docs` Archive Output:
  - `D:\project\MyFlowHub3\docs\change\2026-04-02_flow-protocol-map-and-index-sync.md`
- Index Updates:
  - `D:\project\MyFlowHub3\docs\README.md`
  - `D:\project\MyFlowHub3\docs\specs\README.md`
  - `D:\project\MyFlowHub3\docs\change\README.md`
- Lessons Routing:
  - Lessons impact: `none`
  - 原因：
    - 本轮只补控制面 docs convergence，没有出现新的高成本排障路径
- Control-plane Exception:
  - 本轮改动位于 workspace 根 `docs/`，属于 workflow convergence 允许的 control-plane 例外
- Next Iteration Entry Point:
  - 若继续顺序执行，下一项为 `RC-P1-1: retry backoff / policy`

### Next Iteration - RC-P1-1 retry backoff / policy
#### Stage 1 - Requirements Analysis
- Goal:
  - 为 `flow` 节点执行补齐最小可控的 retry 间隔策略，避免当前失败后“立即重试”对下游执行面和远程 capability 造成突刺。
- Scope:
  - 必须：
    - 新增稳定节点字段 `retry_backoff_ms`
    - 语义限定为第一版固定间隔重试，不引入指数退避或 jitter
    - 更新 `Server docs`、`Proto`、`SubProto` 与测试
    - 保持当前 `retry` 次数、`timeout_ms` 与 `allow_fail` 语义不回退
  - 可选：
    - 若发现现有图校验缺少对新字段的最小边界检查，可一并补齐
  - 不做：
    - 不新增复杂策略对象或多字段 backoff profile
    - 不引入全局 flow 级 retry policy
    - 不修改 Win 编辑器或 UI schema
- Use Cases:
  - 某节点远程 `exec.call` 临时失败时，可在下一次尝试前等待固定毫秒数，而不是立刻重试。
  - 调用方可把 `retry=2 + retry_backoff_ms=500` 理解为“最多三次尝试，失败后每次等待 500ms 再重试”。
  - 未设置 `retry_backoff_ms` 的旧 flow 继续保持当前立即重试行为。
- Functional Requirements:
  - `graph.nodes[].retry_backoff_ms` 为可选 int，默认 `0`
  - 当节点执行失败且仍有剩余重试次数时：
    - 若 `retry_backoff_ms>0`，执行器在下一次尝试前等待该时长
    - 若 run 上下文在等待期间被取消，则立刻停止重试
  - 最后一轮失败后，不再额外等待
  - 新写入 graph 中，`retry_backoff_ms < 0` 必须被拒绝
  - 旧 graph 未设置该字段时，行为与当前主线保持一致
- Non-functional Requirements:
  - 采用最小协议扩展，只新增一个可选字段
  - 不引入 busy loop 或额外后台 goroutine
  - 等待期间必须尊重 run cancel / delete cancel
- Inputs / Outputs:
  - 输入：
    - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\protocol\flow\types.go`
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\handler.go`
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\graph_test.go`
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\runtime_fix_test.go`
  - 输出：
    - 固定间隔 retry backoff docs + protocol + runtime + tests
- Edge Cases:
  - `retry_backoff_ms=0`
  - `retry_backoff_ms<0`
  - 上一次失败后进入 backoff 等待时 run 被 `cancel_run` 或 `delete` 中断
  - 最后一次尝试失败后不应再多等一次 backoff
- Acceptance Criteria:
  1. 新 graph 可声明 `retry_backoff_ms`
  2. 节点在失败重试之间遵守固定等待
  3. backoff 等待期间的取消能尽快生效
  4. 负值 backoff 在保存前被拒绝
  5. requirements/specs/Proto/SubProto/tests 全同步
- Risks:
  - 若把 backoff 写成睡眠而不监听 `ctx.Done()`，取消延迟会变长
  - 若在最后一次失败后仍等待，会无意义拖长失败返回
  - 若直接上复杂 policy，会超出本轮最小改动边界

#### Stage 2 - Architecture Design
- Overall Solution:
  - 在 Proto `Node` 上新增可选字段 `RetryBackoffMs *int`。
  - 在 `executeFlow` 的重试循环中插入一个 `waitRetryBackoff(ctx, backoff)` helper：
    - 仅在失败且仍有后续尝试时等待
    - 通过 `select` 监听 `ctx.Done()`
  - 在 `validateSetNodeKindAndSpec` 路径中补 `retry_backoff_ms >= 0` 校验。
- Alternatives Considered:
  - 备选：直接引入 `retry_policy` 对象（fixed/exponential/jitter）
  - 不采用原因：
    - 现有需求只需要避免立即重试
    - 多字段 wire 扩展与测试面过大
  - 备选：把 backoff 做成全局 flow 配置
  - 不采用原因：
    - 节点间远程依赖和失败特征不同，固定到 flow 级过粗
- Module Responsibilities:
  - `MyFlowHub-Server/docs`
    - 写清 `retry_backoff_ms` 的需求和固定间隔语义
  - `MyFlowHub-Proto/protocol/flow`
    - 扩展 `Node` 字段
  - `MyFlowHub-SubProto/flow`
    - 在执行循环中落实 backoff 等待和取消响应
    - 增加 graph/runtime 测试
- Data / Call Flow:
  1. `set` 保存 graph
  2. `validateGraph` 校验 `retry_backoff_ms >= 0`
  3. `run` 执行节点
  4. 某次尝试失败且仍可重试时：
     - 进入固定 backoff 等待
     - 等待完成后开始下一次 attempt
     - 若等待中 `ctx.Done()`，run 立即终止
- Interface Drafts:
  - `graph.nodes[].retry_backoff_ms`
    - int
    - optional
    - default `0`
- Error Handling and Safety:
  - 新 graph 的负值 backoff 返回明确校验错误
  - backoff 等待不吞掉取消信号
  - 不改变已有 `408/500/404` 等节点执行返回口径
- Performance and Testing Strategy:
  - 不新增后台任务，只在现有重试路径按需等待
  - 测试重点：
    - 固定 backoff 间隔生效
    - backoff 期间取消生效
    - 负值 backoff 被拒绝
- Extensibility Design Points:
  - 后续若要扩展指数退避，可在 `waitRetryBackoff` 之上引入策略计算，不必重写主执行循环

#### Stage 3.1 - Planning
- Docs Governance Routing Decision:
  - 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
  - Requirements impact: `clarify`
  - Specs impact: `clarify`
  - Canonical destination:
    - stable requirements -> `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
    - stable specs -> `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
    - protocol field definition -> `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\protocol\flow\types.go`
    - workflow results -> Stage 4 `docs/change`
  - Related requirements:
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
  - Related specs:
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
  - Related lessons:
    - `D:\project\MyFlowHub3\docs\change\2026-03-22_flow-data-dag-runtime.md`
- Task Mapping:
  - `RC-P1-1`
    - `Server docs`: `retry_backoff_ms` 需求与契约
    - `Proto`: `Node` 新字段
    - `SubProto`: retry loop/backoff helper/validation/tests
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\todo.md`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\protocol\flow\types.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\handler.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\graph_test.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\runtime_fix_test.go`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md`
- Parallelism Assessment:
  - 不使用子Agent
  - 原因：
    - docs / Proto / runtime / tests 共用同一新字段定义
    - `executeFlow` 与 graph 校验共享写集
- Dependencies:
  - `Proto` 字段定义必须先落
  - `Server docs` 需要与固定间隔语义同步
- Risks and Notes:
  - 第一版固定间隔 backoff 是有意收敛，不代表最终策略上限
  - `docs/protocol_map.md` 很可能无正文变化，但 Proto 仍需回归 `go test ./...`
- Issue List:
  - none

阻塞：否
进入 3.2

#### Stage 3.2 - Implementation
- `RC-P1-1` 已完成：
  - `Server docs`：
    - `docs/requirements/flow_data_dag.md`
    - `docs/specs/flow.md`
    - 明确 `retry_backoff_ms` 的固定间隔语义、默认值、取消边界和负值非法规则
  - `Proto`：
    - `protocol/flow/types.go`
    - `Node` 新增 `RetryBackoffMs *int`
    - `docs/protocol_map.md` 无正文变化，符合“字段级扩展不进入 map 正文”的预期
  - `SubProto`：
    - `flow/handler.go`
    - 重试循环新增固定间隔 backoff 等待
    - `waitRetryBackoff(...)` 在等待期间监听 `ctx.Done()`
    - `validateSetNodeKindAndSpec(...)` 拒绝负值 `retry_backoff_ms`
  - `SubProto tests`：
    - `flow/graph_test.go`
    - `flow/runtime_fix_test.go`
    - 新增负值 backoff 校验、固定间隔生效、等待期间取消三类测试

#### Stage 3.3 - Review
- Checklist:
  - 需求覆盖：通过
    - `retry_backoff_ms` 的固定间隔、默认值、取消边界与旧 graph 兼容性已写入稳定 docs
  - 架构合理性：通过
    - 固定间隔等待只插入在“失败且仍有后续尝试”路径，不改变 `retry` / `timeout_ms` / `allow_fail` 原语义
  - 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
    - 未新增后台 goroutine 或额外扫描路径，只在现有重试环路按需等待
  - 可读性与一致性：通过
    - `waitRetryBackoff(...)` 将等待逻辑与主执行循环分离，后续策略扩展落点明确
  - 可扩展性与配置化：通过
    - 第一版只新增一个可选字段，后续若需指数退避可在 helper 上扩展
  - 稳定性与安全：通过
    - backoff 等待期间响应 `cancel_run` / `delete` 取消
    - 新 graph 的负值 backoff 显式返回校验错误
  - 测试覆盖情况：通过
    - 负值校验、固定间隔等待、等待期间取消均有测试锁定
  - 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
    - 本轮未使用子Agent
- Verification:
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1`
    - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - `D:\project\MyFlowHub3`
    - `GOWORK=D:\project\MyFlowHub3\.tmp\verify-run-control-phase3\go.work go test github.com/yttydcs/myflowhub-subproto/flow/... -count=1 -p 1`
- Review Result:
  - 通过

#### Stage 4 - Archive
- 使用 `$m-docs` 校验归档路由、requirements/specs 影响和索引更新。
- `$m-docs` Archive Output:
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\change\2026-04-02_server-flow-retry-backoff-contract.md`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\change\2026-04-02_proto-flow-retry-backoff.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\docs\change\2026-04-02_flow-retry-backoff-runtime.md`
- Index Updates:
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\change\README.md`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\change\README.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\docs\change\README.md`
- Lessons Routing:
  - Lessons impact: `none`
  - 原因：
    - backoff 等待监听 `ctx.Done()` 属于本轮内聚实现决策，暂未形成独立 lessons 条目
    - 相关排查线索已写入各自 `docs/change`，无需再上升到全局经验库
- Next Iteration Entry Point:
  - 若继续顺序执行，下一项为 `RC-P1-2: 并发/重入控制`

### Next Iteration - RC-P1-2 concurrent / reentrant run control
#### Stage 1 - Requirements Analysis
- Goal:
  - 把 `flow` 当前“手动 `run` 可并发、trigger 默认单飞”的隐式行为收敛成可声明、可审计的 active-run 控制，而不破坏旧 flow 的兼容性。
- Scope:
  - 必须：
    - 新增 flow 级可选字段 `max_active_runs`
    - 更新 `Server docs`、`Proto`、`SubProto` 与测试
    - `get` 返回中补齐该字段，避免定义读回时丢失配置
    - 保持旧 flow 未设置该字段时的行为兼容
  - 可选：
    - 若现有加载路径缺少对该字段的最小安全校验，可一并补齐
  - 不做：
    - 不引入复杂 queue / cancel_previous / replace-running policy
    - 不实现 trigger 级 debounce / dedup
    - 不修改 Win 编辑器或 UI schema
- Use Cases:
  - 某些 flow 需要“同一时刻最多只允许一个活动 run”，手动 `run` 命中时应返回明确冲突，而不是继续重入。
  - 某些 trigger flow 需要显式允许重叠运行，而不是沿用当前隐式单飞跳过策略。
  - 旧 flow 未声明并发字段时，现有部署行为不应被静默改变。
- Functional Requirements:
  - `set/get` 定义层新增 `max_active_runs`：可选 int
  - 当 `max_active_runs` 未设置时：
    - 手动 `run` 继续保持当前可并发行为
    - `interval/event/var_changed` 触发继续保持当前“有活动 run 就跳过”的单飞行为
  - 当 `max_active_runs=0` 时：
    - 所有启动来源都视为“不限制活动 run 数”
  - 当 `max_active_runs>0` 时：
    - 手动 `run` 与 trigger 启动都必须遵守统一 active-run 上限
    - 手动 `run` 超限时返回 `409`
    - trigger 超限时跳过本次启动，不生成新 run
  - `max_active_runs<0` 必须在 `set` 校验阶段被拒绝
  - `get` 命中 flow 时必须回显 `max_active_runs`
- Non-functional Requirements:
  - 采用最小协议扩展，不新增新的 action
  - 并发判断与 run 创建必须在同一临界区完成，避免双击 / 双 trigger 竞争穿透
  - 不改变 `cancel_run`、`list_runs`、retention 的既有语义
- Inputs / Outputs:
  - 输入：
    - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\protocol\flow\types.go`
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\handler.go`
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\runtime_fix_test.go`
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\trigger_test.go`
  - 输出：
    - active-run concurrency docs + protocol + runtime + tests
- Edge Cases:
  - 手动 `run` 双击并发进入，不能在“先检查后创建”的窗口里同时穿透上限
  - trigger 连续触发时，默认兼容行为仍需保持单飞
  - `max_active_runs=0` 与“字段未设置”语义不同，必须通过可选字段区分
  - 已删除 flow 的 retained run 不应影响新定义的 active-run 判定
- Acceptance Criteria:
  1. `set/get` 支持 `max_active_runs`
  2. `max_active_runs>0` 时，手动和 trigger 启动都遵守统一 active-run 上限
  3. 手动 `run` 超限返回 `409`
  4. trigger 超限时跳过，不创建新 run
  5. 未设置字段时保持当前兼容行为
  6. requirements/specs/Proto/SubProto/tests 全同步
- Risks:
  - 若并发检查不和 run 记录放在同一锁内，会留下竞争窗口
  - 若把未设置字段默认解释成 `1`，会破坏现有手动 `run` 行为
  - 若忘记更新 `get` 响应，调用方读回定义会丢失并发配置

#### Stage 2 - Architecture Design
- Overall Solution:
  - 在 Proto `SetReq` / `GetResp` 上新增可选字段 `MaxActiveRuns *int`。
  - 在 `SubProto flow` 中新增：
    - `validateFlowRunConfig(...)`
    - `activeRunCountLocked(flowID string) int`
    - 统一的“检查上限并登记 run” helper
  - 手动 `run` 路径在本地执行前判断 active-run 上限，超限返回 `409 active run limit reached`。
  - trigger 路径继续走 `tryStartRunWithTrigger(...)`，但改为根据显式字段决定是否允许重叠。
- Alternatives Considered:
  - 备选：新增 `allow_reentry` bool
  - 不采用原因：
    - 无法表达“显式无限制”与“兼容 legacy 默认值”的差异
    - 后续若要支持 `2/3/...` 的并发上限会再次改 wire
  - 备选：未设置字段时统一按 `1` 处理
  - 不采用原因：
    - 会回退当前手动 `run` 的并发行为
  - 备选：直接做 `cancel_previous`
  - 不采用原因：
    - 会引入新的终止语义和更大测试面
- Module Responsibilities:
  - `MyFlowHub-Server/docs`
    - 明确 `max_active_runs` 的兼容默认值、手动冲突和 trigger 跳过规则
  - `MyFlowHub-Proto/protocol/flow`
    - 扩展 `SetReq` / `GetResp`
  - `MyFlowHub-SubProto/flow`
    - 落实统一 active-run 上限判定和竞态安全的 run 创建
    - 补齐 set/get/runtime/trigger 测试
- Data / Call Flow:
  1. `set` 保存 flow 定义并校验 `max_active_runs`
  2. `get` 读回该字段
  3. `run` 或 trigger 到达执行器
  4. 执行器读取 flow 定义中的 effective active-run limit
  5. 若允许启动，则在同一临界区登记新 run；否则：
     - 手动 `run` 返回 `409`
     - trigger 跳过
- Interface Drafts:
  - `set/get.max_active_runs`
    - `nil` -> legacy compatibility
    - `0` -> unlimited for all start sources
    - `>0` -> unified active-run cap
- Error Handling and Safety:
  - `max_active_runs < 0` -> `400`
  - 手动 `run` 超限 -> `409 active run limit reached`
  - trigger 超限不生成假 run，也不改写已有 run 状态
- Performance and Testing Strategy:
  - 复用现有 `runOrderByFlow` 索引，不新增全表扫描
  - 重点验证：
    - set 校验负值
    - 手动 `run` 超限返回 `409`
    - trigger 默认兼容单飞
    - 显式 `max_active_runs=0` 或 `>1` 时允许重叠
    - `get` 回显该字段
- Extensibility Design Points:
  - 后续若要扩展 `cancel_previous` / `queue`，可以在同一“start gate” helper 上增加策略分支，而不必重写 run 生命周期

#### Stage 3.1 - Planning
- Docs Governance Routing Decision:
  - 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
  - Requirements impact: `clarify`
  - Specs impact: `clarify`
  - Canonical destination:
    - stable requirements -> `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
    - stable specs -> `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
    - protocol request/response fields -> `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\protocol\flow\types.go`
    - workflow results -> Stage 4 `docs/change`
  - Related requirements:
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
  - Related specs:
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
  - Related lessons:
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\docs\lessons\flow-trigger-run-missing-server-context.md`
- Task Mapping:
  - `RC-P1-2`
    - `Server docs`: `max_active_runs` requirements/specs
    - `Proto`: `SetReq` / `GetResp` 新字段
    - `SubProto`: set validation / get echo / manual-run conflict / trigger gate / tests
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\todo.md`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\protocol\flow\types.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\handler.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\runtime_fix_test.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\trigger_test.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\flow_id_test.go`
- Parallelism Assessment:
  - 不使用子Agent
  - 原因：
    - docs / Proto / runtime / tests 共享同一字段与兼容语义
    - 并发 gate 与 run 创建存在竞态边界，需要主代理统一修改
- Dependencies:
  - `Proto` 字段定义必须先落
  - `Server docs` 需要与兼容默认值口径保持一致
- Risks and Notes:
  - `nil` 与 `0` 的语义必须严格区分，不能把可选字段误写成普通 int
  - 若只改 trigger 路径不改 manual `run`，本轮“并发/重入控制”目标就不完整
- Issue List:
  - none

阻塞：否
进入 3.2

#### Stage 3.2 - Implementation
- `RC-P1-2` 已完成：
  - `Server docs`：
    - `docs/requirements/flow_data_dag.md`
    - `docs/specs/flow.md`
    - 补齐 `max_active_runs` 的兼容默认值、手动冲突和 trigger 跳过规则
  - `Proto`：
    - `protocol/flow/types.go`
    - `SetReq` / `GetResp` 新增 `MaxActiveRuns *int`
  - `SubProto`：
    - `flow/handler.go`
    - `flow/flow_id_test.go`
    - `flow/runtime_fix_test.go`
    - `flow/trigger_test.go`
    - set 校验负值 `max_active_runs`
    - `get` 回显 `max_active_runs`
    - 手动 `run` 超限返回 `409 active run limit reached`
    - trigger 启动在超限时跳过，`max_active_runs=0` 时允许重叠

#### Stage 3.3 - Review
- Checklist:
  - 需求覆盖：通过
    - `max_active_runs` 的 `nil` / `0` / `>0` 语义、手动冲突和 trigger 跳过规则均已落实
  - 架构合理性：通过
    - 统一采用单一 start gate，避免 manual/trigger 两条路径继续漂移
  - 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
    - 复用现有 `runOrderByFlow` 索引，未新增全表扫描或额外持久化
  - 可读性与一致性：通过
    - active-run 检查、run 创建与返回路径已收敛到同一组 helper
  - 可扩展性与配置化：通过
    - 后续若要扩展 queue / `cancel_previous`，可继续挂到同一 start gate
  - 稳定性与安全：通过
    - 负值 `max_active_runs` 在保存和磁盘加载两个入口都被拒绝
    - active-run 检查与 run 登记在同一临界区完成
  - 测试覆盖情况：通过
    - 负值校验、手动 `409`、legacy trigger 单飞、显式 unlimited trigger overlap、`get` 回显均已覆盖
  - 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
    - 本轮未使用子Agent
- Verification:
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1`
    - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - `D:\project\MyFlowHub3`
    - `GOWORK=D:\project\MyFlowHub3\.tmp\verify-run-control-phase3\go.work go test github.com/yttydcs/myflowhub-subproto/flow/... -count=1 -p 1`
- Review Result:
  - 通过

#### Stage 4 - Archive
- 使用 `$m-docs` 校验归档路由、requirements/specs 影响和索引更新。
- `$m-docs` Archive Output:
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\change\2026-04-02_server-flow-active-run-limit-contract.md`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\change\2026-04-02_proto-flow-active-run-limit.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\docs\change\2026-04-02_flow-active-run-limit-runtime.md`
- Index Updates:
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\change\README.md`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\change\README.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\docs\change\README.md`
- Lessons Routing:
  - Lessons impact: `none`
  - 原因：
    - 本轮主要是对已有 run-control 体系补齐显式字段和 start gate，没有形成独立的新型排障模式
    - 具体排查线索已写入各自 `docs/change`
- Next Iteration Entry Point:
  - 若继续顺序执行，下一项为 `RC-P1-3: 触发去重`

### Next Iteration - RC-P1-3 trigger dedup
#### Stage 1 - Requirements Analysis
- Goal:
  - 为 `event/var_changed` trigger 补齐最小可控的重复启动抑制能力，在不引入持久化复杂度的前提下减少重复通知导致的重复 run。
- Scope:
  - 必须：
    - 新增 trigger 级可选字段 `dedup_window_ms`
    - 更新 `Server docs`、`Proto`、`SubProto` 与测试
    - 明确 dedup 仅作用于 trigger 启动，不影响手动 `run`
    - 明确 dedup 默认关闭、内存态、重启后可清空
  - 可选：
    - 若现有 trigger 校验缺少对 dedup 的最小安全限制，可一并补齐
  - 不做：
    - 不实现 run archive / history retention 扩展
    - 不实现跨重启持久化 dedup
    - 不为 `interval` trigger 引入 dedup
- Use Cases:
  - 上游重复发布同一事件时，短窗口内不应为同一 flow 连续启动等价 run。
  - 同一变量在极短时间内重复发出相同 `changed/deleted` 通知时，trigger flow 应能选择抑制重复启动。
  - 未声明 dedup 的旧 flow 仍保持当前 trigger 行为，不被静默改变。
- Functional Requirements:
  - `trigger.dedup_window_ms`：可选 int
  - 当 `dedup_window_ms` 未设置或 `0` 时：
    - trigger 行为保持当前现状，不做额外 dedup
  - 当 `dedup_window_ms>0` 时：
    - `event` 与 `var_changed` trigger 必须按“同一 flow + 同一规范化 trigger 上下文”做窗口去重
    - 窗口内重复 trigger 不得生成新的 run
  - `dedup_window_ms<0` 必须在 `set` 校验阶段被拒绝
  - `interval` trigger 配置 `dedup_window_ms>0` 必须被拒绝
- Non-functional Requirements:
  - dedup 必须是显式 opt-in
  - dedup 状态仅存内存，不引入新的持久化结构
  - 去重判断与 run 登记必须在同一临界区完成，避免竞争窗口
- Inputs / Outputs:
  - 输入：
    - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\protocol\flow\types.go`
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\handler.go`
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\trigger_test.go`
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\flow_id_test.go`
  - 输出：
    - trigger dedup docs + protocol + runtime + tests
- Edge Cases:
  - 同一 `event` 在窗口内重复出现，但 payload 不同，不能误判为同一次 trigger
  - dedup 生效期间 run 已经结束，窗口内重复 trigger 仍需被抑制
  - `set/delete` 后旧 dedup 状态不能污染新定义
- Acceptance Criteria:
  1. `trigger` 支持 `dedup_window_ms`
  2. `event/var_changed` 在 dedup 窗口内重复 trigger 时不会生成新 run
  3. `interval` + `dedup_window_ms>0` 会被拒绝
  4. `dedup_window_ms<0` 会被拒绝
  5. 未设置字段时保持当前兼容行为
  6. requirements/specs/Proto/SubProto/tests 全同步
- Risks:
  - 若 dedup key 依赖不稳定字段，可能把同一 trigger 错误视为不同启动
  - 若 dedup 检查与 run 创建不在同一锁内，重复 trigger 仍可能竞争穿透
  - 若在 `set/delete` 后不清理 dedup 状态，新定义可能被旧窗口误伤

#### Stage 2 - Architecture Design
- Overall Solution:
  - 在 Proto `Trigger` 上新增可选字段 `DedupWindowMs *int`。
  - 在 `SubProto flow` 中新增：
    - `Handler.triggerDedup map[string]map[string]time.Time`
    - 对 trigger 的 dedup 校验与窗口记录逻辑
  - dedup key 由规范化 trigger 上下文生成：
    - `event` 使用 `buildTopicTriggerContext(...)`
    - `var_changed` 使用 `buildVarChangedTriggerContext(...)`
  - 在统一 start gate `prepareQueuedRunLocked(...)` 中完成 dedup 检查与记录，和 active-run gate 共用同一临界区。
- Alternatives Considered:
  - 备选：做 flow 级 dedup，不看 trigger 内容
  - 不采用原因：
    - 会把不同 payload / 不同变量变更错误地当成同一次 trigger
  - 备选：把 dedup 状态持久化到磁盘
  - 不采用原因：
    - 当前需求只需抑制短窗口重复启动，引入持久化会放大复杂度与恢复语义
  - 备选：为 `interval` 也开放 dedup
  - 不采用原因：
    - `interval` 本身已由调度频率决定，dedup 语义不清晰且容易与 active-run gate 混淆
- Module Responsibilities:
  - `MyFlowHub-Server/docs`
    - 明确 `dedup_window_ms` 的支持范围、默认值和 dedup 语义
  - `MyFlowHub-Proto/protocol/flow`
    - 扩展 `Trigger` 字段
  - `MyFlowHub-SubProto/flow`
    - 落实 trigger dedup runtime、校验与测试
- Data / Call Flow:
  1. `set` 保存 flow 定义并校验 `trigger.dedup_window_ms`
  2. `event/var_changed` 到达执行器并生成规范化 trigger 上下文
  3. start gate 在锁内先检查 dedup 窗口，再判断 active-run 规则
  4. 若通过 gate，则登记新 run 并更新 dedup 时间戳；否则跳过启动
- Interface Drafts:
  - `trigger.dedup_window_ms`
    - `nil` / `0` -> dedup disabled
    - `>0` -> event/var_changed trigger dedup enabled
- Error Handling and Safety:
  - `dedup_window_ms < 0` -> `400`
  - `interval` + `dedup_window_ms > 0` -> `400`
  - dedup 只抑制 trigger，不生成失败 run
- Performance and Testing Strategy:
  - 以 `flowID -> dedupKey -> lastSeen` 的内存 map 保存窗口状态
  - 重点验证：
    - 负值校验
    - `interval` 不支持 dedup
    - 重复 event 在窗口内被抑制
    - 不同 payload 不被误杀
    - 窗口过后可再次启动
- Extensibility Design Points:
  - 后续若要实现 run archive 或更复杂 trigger policy，可继续沿用同一 start gate 和规范化 trigger 上下文。

#### Stage 3.1 - Planning
- Docs Governance Routing Decision:
  - 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
  - Requirements impact: `add`
  - Specs impact: `add`
  - Canonical destination:
    - stable requirements -> `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
    - stable specs -> `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
    - protocol trigger field -> `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\protocol\flow\types.go`
    - workflow results -> Stage 4 `docs/change`
  - Related requirements:
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
  - Related specs:
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
  - Related lessons:
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\docs\lessons\flow-trigger-run-missing-server-context.md`
- Task Mapping:
  - `RC-P1-3`
    - `Server docs`: `dedup_window_ms` requirements/specs
    - `Proto`: `Trigger.DedupWindowMs` 新字段
    - `SubProto`: trigger dedup validation / gate / cleanup / tests
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\todo.md`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\protocol\flow\types.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\handler.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\trigger_test.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\flow_id_test.go`
- Parallelism Assessment:
  - 不使用子Agent
  - 原因：
    - dedup contract、wire 字段、runtime key 规范与测试共用同一语义
    - start gate 竞争边界需要主代理统一修改与复核
- Dependencies:
  - `Proto` 字段定义必须先落
  - `Server docs` 需要先明确支持范围和禁用场景
- Risks and Notes:
  - 本轮 checklist 细化：`RC-P1-3` 仅负责 trigger dedup，`run archive` 顺延到 `RC-P1-4`
  - dedup key 必须使用稳定的规范化 trigger 上下文，避免 JSON 字段顺序抖动
- Issue List:
  - none

阻塞：否
进入 3.2

#### Stage 3.2 - Implementation
- `RC-P1-3` 已完成：
  - `Server docs`：
    - `docs/requirements/flow_data_dag.md`
    - `docs/specs/flow.md`
    - 补齐 `trigger.dedup_window_ms` 的支持范围、默认值与窗口去重规则
  - `Proto`：
    - `protocol/flow/types.go`
    - `Trigger` 新增 `DedupWindowMs *int`
  - `SubProto`：
    - `flow/handler.go`
    - `flow/trigger_test.go`
    - `flow/flow_id_test.go`
    - 新增 `triggerDedup` 内存态窗口记录
    - set 校验拒绝负值 `dedup_window_ms`
    - 拒绝 `interval` + `dedup_window_ms>0`
    - 在 `prepareQueuedRunLocked(...)` 中对 `event/var_changed` 触发做窗口 dedup
    - `set/delete` 时清理旧 dedup 状态

#### Stage 3.3 - Review
- Checklist:
  - 需求覆盖：通过
    - `dedup_window_ms` 的默认关闭、正值启用、负值拒绝和 `interval` 禁用规则均已落实
  - 架构合理性：通过
    - dedup 与 active-run gate 复用同一 start gate，没有再分散到多条 trigger 路径
  - 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
    - 仅引入内存 map 和规范化 trigger JSON，未新增持久化或全量扫描
  - 可读性与一致性：通过
    - dedup key 统一来自既有 trigger context builder，避免另起一套字段拼装逻辑
  - 可扩展性与配置化：通过
    - 后续若补 run archive 或更复杂 trigger policy，可继续挂在同一 start gate
  - 稳定性与安全：通过
    - dedup 校验和窗口记录在同一锁内完成；`set/delete` 会清理旧窗口状态
  - 测试覆盖情况：通过
    - 负值校验、`interval` 禁用、窗口内重复 event 抑制、不同 payload 放行均有测试锁定
  - 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
    - 本轮未使用子Agent
- Verification:
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1`
    - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - `D:\project\MyFlowHub3`
    - `GOWORK=D:\project\MyFlowHub3\.tmp\verify-run-control-phase3\go.work go test github.com/yttydcs/myflowhub-subproto/flow/... -count=1 -p 1`
- Review Result:
  - 通过

#### Stage 4 - Archive
- 使用 `$m-docs` 校验归档路由、requirements/specs 影响和索引更新。
- `$m-docs` Archive Output:
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\change\2026-04-02_server-flow-trigger-dedup-contract.md`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\change\2026-04-02_proto-flow-trigger-dedup-window.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\docs\change\2026-04-02_flow-trigger-dedup-runtime.md`
- Index Updates:
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\change\README.md`
  - `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\change\README.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\docs\change\README.md`
- Lessons Routing:
  - Lessons impact: `none`
  - 原因：
    - 本轮新增的是最小 dedup 策略与校验边界，没有形成新的跨模块排障模式
    - 相关排查线索已写入各自 `docs/change`
- Next Iteration Entry Point:
  - 若继续顺序执行，下一项为 `RC-P1-4: run archive`

### Next Iteration - RC-P1-4 run archive
#### Stage 1 - Requirements Analysis
- Goal:
  - 为 retained window 内的终态 run 增加可重载的本地归档能力，让 `status/detail/list_runs` 在执行器重启后仍能查询最近保留的 run。
- Scope:
  - 必须：
    - 更新 `Server docs`
    - 在 `SubProto flow` 增加可选 run archive
    - 保持既有 wire 不变，不新增 action
    - 删除 flow 定义后，retained archive 仍可在窗口内查询
  - 可选：
    - 若归档上限可以直接复用现有 `flow.max_retained_runs`，则不新增新的 count config
  - 不做：
    - 不改 Proto wire
    - 不持久化活动 run
    - 不承诺窗口外长期历史
- Use Cases:
  - 某个 flow 的最近一次运行已经结束，执行器重启后仍能通过 `status/detail/list_runs` 读取该 retained run。
  - flow 定义被删除后，调用方仍可在 retained window 内查看刚结束的历史 run。
  - retained window 超限时，最老 archive 会被清理，避免无限增长。
- Functional Requirements:
  - 新增配置 `flow.run_archive_enabled`
  - 当 `flow.run_archive_enabled=true` 时：
    - retained window 内的终态 run 摘要与节点结果必须落到本地 archive
    - 执行器启动时必须加载 archive，供 `status/detail/list_runs` 查询
  - archive 仍复用 `flow.max_retained_runs` 作为 retained window 上限
  - 超出 retained window 的最老 archive 必须被清理
  - 删除 flow 定义不得立即删除 retained archive
- Non-functional Requirements:
  - 保持既有 `run/status/detail/list_runs` wire 与权限模型不变
  - 默认关闭，避免无意新增本地 I/O
  - archive 失败不能静默吞掉；需要显式日志
- Inputs / Outputs:
  - 输入：
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\config.go`
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\handler.go`
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\runtime_fix_test.go`
  - 输出：
    - run archive docs + runtime + tests
- Edge Cases:
  - retained run 在 archive 打开前已存在，不能假定自动补归档
  - 执行器重启后没有 flow 定义，但 archive 仍在 retained window 内
  - archive 超限时，需要同时裁剪内存 retained run 与磁盘 archive
- Acceptance Criteria:
  1. `flow.run_archive_enabled=true` 时，终态 run 会进入 retained archive
  2. 执行器重启后，`status/detail/list_runs` 仍可读取 retained archive
  3. 删除 flow 定义后，retained archive 仍可在窗口内查询
  4. 超出 `flow.max_retained_runs` 的旧 archive 会被清理
  5. 不新增 Proto / wire 变更
- Risks:
  - 若 archive 写入和 retained pruning 没有统一边界，可能出现“内存删了、磁盘没删”或反过来的漂移
  - 若 archive 默认开启，会给测试和未显式配置的部署引入额外 I/O
  - 若 archive 仅保 summary 不保 runtime，`detail` 重启后会退化

#### Stage 2 - Architecture Design
- Overall Solution:
  - 采用本地 JSON sidecar archive：
    - `flow.base_dir/_runs/<flow_id>/<run_id>.json`
  - archive 仅保存终态 run 的摘要、取消原因和完整 `runContext`
  - `Init()` 在 flow 定义加载后，再加载 retained archive 到 `runs/runOrderByFlow`
  - 继续复用 `flow.max_retained_runs` 做 retained window 上限；超限时同时裁掉内存与 archive 文件
- Alternatives Considered:
  - 备选：新增 Proto action 读取 archive
  - 不采用原因：
    - 现有 `status/detail/list_runs` 已足够承载 retained archive，不需要扩 wire
  - 备选：将 run archive 并入现有 definition persistence backend
  - 不采用原因：
    - 现有 `Persistence` 只负责 flow 定义；把 run archive 混进去会污染职责边界
  - 备选：默认开启 archive
  - 不采用原因：
    - 会给未显式配置的环境和测试引入额外本地 I/O
- Module Responsibilities:
  - `MyFlowHub-Server/docs`
    - 固化 `flow.run_archive_enabled` 契约与 retained archive 查询语义
  - `MyFlowHub-SubProto/flow`
    - 解析 archive config
    - 落地 archive save/load/prune
    - 补齐重启后查询与删除后保留的回归
- Data / Call Flow:
  1. run 进入终态
  2. 若 archive 开启，则把 run 摘要 + runtime 写入 local archive JSON
  3. prune 根据 `flow.max_retained_runs` 裁剪最老 retained run，并同步删除超限 archive
  4. `Init()` 扫描 archive 并恢复 retained runs 到查询索引
  5. `status/detail/list_runs` 继续走现有查询面命中这些 archived retained runs
- Interface Drafts:
  - config:
    - `flow.run_archive_enabled=false` -> legacy in-memory retained window
    - `flow.run_archive_enabled=true` -> retained window 持久化为 local archive
- Error Handling and Safety:
  - archive 写入/读取/删除失败 -> 记录明确 warning
  - archive 不参与活动 run 恢复
  - archive 路径必须继续受 `flow_id/run_id` 校验约束
- Performance and Testing Strategy:
  - 仅在终态保存和初始化预热时做文件 I/O
  - 重点验证：
    - retained archive reload
    - prune 删除旧 archive
    - delete 后 archive 仍可查询
- Extensibility Design Points:
  - 当前采用 sidecar JSON，不阻止未来抽出独立 `RunArchivePersistence`

#### Stage 3.1 - Planning
- Docs Governance Routing Decision:
  - 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
  - Requirements impact: `add`
  - Specs impact: `add`
  - Canonical destination:
    - stable requirements -> `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
    - stable specs -> `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
    - runtime implementation -> `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow`
    - workflow results -> Stage 4 `docs/change`
  - Related requirements:
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
  - Related specs:
    - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
  - Related lessons:
    - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\docs\lessons\flow-trigger-run-missing-server-context.md`
- Task Mapping:
  - `RC-P1-4`
    - `Server docs`: retained run archive contract
    - `SubProto`: archive config / save / load / prune / tests
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\todo.md`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\config.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\handler.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\run_archive.go`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\flow\runtime_fix_test.go`
- Parallelism Assessment:
  - 不使用子Agent
  - 原因：
    - archive save/load/prune 与 retained window 语义强耦合
    - 需要主代理统一控制 docs/runtime/test 的边界
- Dependencies:
  - `Server docs` 先固定 archive 默认关闭和 retained window 语义
  - `SubProto` 复用现有 retained window，而不是额外引入新的上限字段
- Risks and Notes:
  - Proto impact: `none`
  - archive 采用 sidecar JSON；未来若需要 PG / 独立 backend，再单独抽象
- Issue List:
  - none

阻塞：否
进入 3.2

#### Stage 3.2 - Implementation
- `RC-P1-4` 已完成：
  - `Server docs`：
    - `docs/requirements/flow_data_dag.md`
    - `docs/specs/flow.md`
    - 补齐 `flow.run_archive_enabled` 和 retained archive 语义
  - `SubProto`：
    - `flow/config.go`
    - `flow/handler.go`
    - `flow/run_archive.go`
    - `flow/runtime_fix_test.go`
    - 新增 `flow.run_archive_enabled` 配置解析
    - 终态 run 以 local JSON sidecar 归档到 `flow.base_dir/_runs/...`
    - `Init()` 预热 retained archive 到查询索引
    - prune 超限时同步删除旧 archive
    - delete 后 retained archive 仍可重载查询

#### Stage 3.3 - Review
- Checklist:
  - 需求覆盖：通过
    - retained archive、重启后查询、delete 后保留、超限裁剪均已落实
  - 架构合理性：通过
    - archive 采用 sidecar，不污染现有 flow definition persistence 接口
  - 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
    - I/O 仅发生在终态保存、启动预热和超限裁剪；查询路径继续复用内存索引
  - 可读性与一致性：通过
    - `finalizeRun -> archive -> prune` 顺序明确，save/load/prune 边界集中在 `run_archive.go`
  - 可扩展性与配置化：通过
    - 当前默认关闭，后续若要抽独立 archive backend 仍有清晰落点
  - 稳定性与安全：通过
    - archive 路径继续受 `flow_id/run_id` 校验约束
    - archive 失败会记录 warning，不静默吞掉
  - 测试覆盖情况：通过
    - retained archive reload、旧 archive pruning、delete 后 archive 查询已覆盖
  - 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
    - 本轮未使用子Agent
- Verification:
  - `D:\project\MyFlowHub3`
    - `GOWORK=D:\project\MyFlowHub3\.tmp\verify-run-control-phase3\go.work go test github.com/yttydcs/myflowhub-subproto/flow/... -count=1 -p 1`
- Review Result:
  - 通过

#### Stage 4 - Archive
- 使用 `$m-docs` 校验归档路由、requirements/specs 影响和索引更新。
- `$m-docs` Archive Output:
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\change\2026-04-02_server-flow-run-archive-contract.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\docs\change\2026-04-02_flow-run-archive-runtime.md`
- Index Updates:
  - `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\change\README.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-run-control-phase1\docs\change\README.md`
- Lessons Routing:
  - Lessons impact: `none`
  - 原因：
    - 本轮主要是 retained run 的持久化补强，没有形成新的跨模块排障模式
    - archive 相关排查线索已写入各自 `docs/change`
- Next Iteration Entry Point:
  - 若继续顺序执行，下一项可转入更高阶 flow 编排能力或单独拆新的 run archive backend/PG 支持
