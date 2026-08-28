# Plan - flow-orchestrators

## Workflow Information
- Primary Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-SubProto`
- Branch: `feat/flow-orchestrators`
- Base: `main`
- Primary Worktree: `D:\project\MyFlowHub3\worktrees\subproto-flow-orchestrators`
- Additional Worktrees:
  - `D:\project\MyFlowHub3\worktrees\proto-flow-orchestrators`
  - `D:\project\MyFlowHub3\worktrees\server-flow-orchestrators`
- Final Status: `4 Archived / merged to control-plane / cleanup completed`

## Stage Records

### Initialization
- 使用 `$m-autoflow` 执行本轮工作流。
- `guide.md` 已读取，确认：
  - commit 信息使用中文
  - 稳定长期文档以 `repo/MyFlowHub-Server/docs` 为准
  - 所有临时实现必须放在 `D:\project\MyFlowHub3\worktrees\`
- 参与仓库：
  - `MyFlowHub-SubProto/flow`
  - `MyFlowHub-Proto/protocol/flow`
  - `MyFlowHub-Server/docs/requirements/flow_data_dag.md`
  - `MyFlowHub-Server/docs/specs/flow.md`

### Stage 1 - Requirements Analysis
- Goal:
  - 把 `flow` 从“可执行 DAG”补到最小可用的高阶编排工作流，首版落地 `branch`、`foreach`、`subflow`、`trigger.type=cron`。
- Must:
  - `branch` 使用显式规则匹配与 `edge.case` 路由
  - `foreach` 使用串行 body graph，暴露 `loop_item` / `loop_index`
  - `subflow` 只支持同执行器同步调用，禁止递归
  - `cron` 使用 5-field 表达式，执行器本地时区计算，不提供 timezone 字段
  - requirements/specs、Proto、SubProto 与测试必须同步
- Not In Scope:
  - 任意脚本表达式
  - `foreach` 并行 fan-out
  - cross-executor `subflow`
  - async `subflow`
  - cron timezone 字段
  - 重启后的 cron catch-up
- Acceptance:
  - `branch` 未选中路径节点显式标记为 `skipped`
  - `foreach` 结果数组顺序与输入一致
  - `subflow` 可同步执行目标 flow 并返回指定结果节点
  - `cron` 可按 5-field 表达式调度

### Stage 2 - Architecture Design
- Overall Solution:
  - `Proto` 只扩最小 wire 字段：`Trigger.Cron`、`Edge.Case`
  - `Server docs` 作为稳定真相，补齐 orchestrator 节点与 cron 约束
  - `SubProto` 继续复用现有 DAG executor，通过：
    - branch edge gating
    - nested graph execution helper
    - loop run context
    - synchronous subflow execution
    - in-process cron scheduler
- Alternatives Rejected:
  - branch 内嵌子图：merge 语义更重，不利于最小落地
  - parallel foreach：错误聚合和资源控制复杂度过高
  - self-call `run/status/detail` 实现 subflow：会污染 retained run 与递归控制
  - 引入第三方 cron 库：首版只需 5-field，内建实现更易审计
- Error / Safety:
  - `set` 阶段显式拒绝非法 `cron`、非法 `edge.case`、非数组 foreach source、未知 subflow、self-subflow
  - 运行时禁止静默降级

### Stage 3.1 - Planning
- 使用 `$m-docs` 完成归档路由判断：
  - Requirements impact: `add`
  - Specs impact: `add`
  - Related requirements:
    - `D:\project\MyFlowHub3\worktrees\server-flow-orchestrators\docs\requirements\flow_data_dag.md`
  - Related specs:
    - `D:\project\MyFlowHub3\worktrees\server-flow-orchestrators\docs\specs\flow.md`
  - Related lessons:
    - `D:\project\MyFlowHub3\worktrees\subproto-flow-orchestrators\docs\lessons\flow-trigger-run-missing-server-context.md`
- Task Mapping:
  - `ORCH-DOC-1`
    - 更新 `Server` requirements/specs
  - `ORCH-PROTO-1`
    - 更新 `Proto` flow trigger/edge 合同
  - `ORCH-RT-1`
    - 扩展 graph/spec 校验
  - `ORCH-RT-2`
    - 扩展运行时上下文与 binding source
  - `ORCH-RT-3`
    - 扩展执行器支持 `branch/foreach/subflow`
  - `ORCH-RT-4`
    - 扩展 scheduler 支持 `cron`
  - `ORCH-TEST-1`
    - 补 graph/runtime/trigger 回归测试
- Parallelism Assessment:
  - `Proto` 与 `Server docs` 理论上可并行
  - `SubProto` runtime / tests 写集重叠，不拆分
  - 本轮未使用子 Agent

### Stage 3.2 - Implementation
- `ORCH-DOC-1` 已完成：
  - `MyFlowHub-Server/docs/requirements/flow_data_dag.md`
  - `MyFlowHub-Server/docs/specs/flow.md`
  - 正式纳入 `branch` / `foreach` / `subflow` / `cron` 与 v1 边界
- `ORCH-PROTO-1` 已完成：
  - `MyFlowHub-Proto/protocol/flow/types.go`
  - `Trigger.Cron string`
  - `Edge.Case string`
  - `go run ./cmd/protocolmapgen -write -out docs/protocol_map.md` 执行后无差异
- `ORCH-RT-1` ~ `ORCH-RT-4` 已完成：
  - `MyFlowHub-SubProto/flow/handler.go`
  - `MyFlowHub-SubProto/flow/runtime_bindings.go`
  - 运行时新增：
    - `executeGraph(...)`
    - `shouldExecuteNode(...)`
    - `runContext.Loop`
    - `loop_item` / `loop_index`
    - `branchSpec` / `foreachSpec` / `subflowSpec`
    - `buildCronTriggerContext(...)`
    - `buildSubflowTriggerContext(...)`
    - `validateGraphForFlow(...)`
    - `validateGraphScoped(...)`
    - `parseCronExpr(...)`
    - `cronSchedule.NextAfter(...)`
- `ORCH-TEST-1` 已完成：
  - `MyFlowHub-SubProto/flow/graph_test.go`
  - `MyFlowHub-SubProto/flow/trigger_test.go`
  - `MyFlowHub-SubProto/flow/orchestrator_test.go`

### Stage 3.3 - Review
- Checklist:
  - 需求覆盖：通过
  - 架构合理性：通过
  - 性能风险：通过
  - 可读性与一致性：通过
  - 可扩展性与配置化：通过
  - 稳定性与安全：通过
  - 测试覆盖情况：通过
  - 子Agent治理与审计：通过（本轮未使用子 Agent）
- Verification:
  - `D:\project\MyFlowHub3\worktrees\proto-flow-orchestrators`
    - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - `D:\project\MyFlowHub3`
    - `GOWORK=D:\project\MyFlowHub3\.tmp\verify-flow-orchestrators\go.work go test github.com/yttydcs/myflowhub-subproto/flow -run 'TestValidateTrigger|TestCronScheduleNextAfter|TestValidateGraphRejectsBranchEdgeWithoutCase|TestValidateGraphRejectsLoopItemOutsideForeach|TestExecuteFlow_BranchSkipsUnselectedPath|TestExecuteFlow_ForeachAggregatesResults|TestExecuteFlow_SubflowReturnsResult' -count=1 -p 1`
    - `GOWORK=D:\project\MyFlowHub3\.tmp\verify-flow-orchestrators\go.work go test github.com/yttydcs/myflowhub-subproto/flow -run 'TestValidateGraph|TestExecuteFlow|TestHandleTopic|TestHandleVarChanged|TestTryStart|TestCronScheduleNextAfter' -count=1 -p 1`
- Known Baseline:
  - `github.com/yttydcs/myflowhub-subproto/flow -count=1 -p 1` 全量包测试仍被既有权限基线阻塞
  - 主要集中在 `delete/detail/run/status/list_runs/get` 相关 `403 permission denied`
  - 该问题早于本轮 orchestrator 变更存在，不视为本轮回归

### Stage 4 - Archive
- 使用 `$m-docs` 完成归档路由与索引更新。
- Change Outputs:
  - `docs/change/2026-04-03_flow-orchestrators-runtime.md`
  - `docs/change/2026-04-03_proto-flow-orchestrators.md`
  - `docs/change/2026-04-03_server-flow-orchestrators-contract.md`
- Plan Output:
  - `docs/plan/plan_archive_2026-04-03_flow-orchestrators.md`
- Root Index Updates:
  - `plan.md`
  - `docs/change/README.md`
  - `docs/plan/README.md`
- Lessons Routing:
  - Lessons impact: `none`
  - 原因：本轮新增的是稳定 orchestrator 语义与最小 runtime 扩展，没有形成新的高成本排障路径

### Workflow End
- 控制面合并：
  - `MyFlowHub-SubProto/flow/*`
  - `MyFlowHub-Proto/protocol/flow/types.go`
  - `MyFlowHub-Server/docs/requirements/flow_data_dag.md`
  - `MyFlowHub-Server/docs/specs/flow.md`
- 根级归档：
  - 三份 `change` 文档已归并到 `docs/change/`
  - 本文已归档到 `docs/plan/`
- Worktree Cleanup:
  - `D:\project\MyFlowHub3\worktrees\subproto-flow-orchestrators`
  - `D:\project\MyFlowHub3\worktrees\proto-flow-orchestrators`
  - `D:\project\MyFlowHub3\worktrees\server-flow-orchestrators`
  - 已完成移除与 prune
