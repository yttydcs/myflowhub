# Plan - Flow Transform Node

## Workflow Information
- Control Repo: `D:\project\MyFlowHub3`
- Workflow Date: `2026-04-03`
- Participating Repos:
  - `MyFlowHub-SubProto`
  - `MyFlowHub-Server`
- Base Branch:
  - `SubProto`: `main`
  - `Server`: `main`
- Worktrees:
  - `D:\project\MyFlowHub3\worktrees\subproto-transform-node`
  - `D:\project\MyFlowHub3\worktrees\server-transform-node`
- Worktree Branch:
  - `feat/transform-node`
- Current Stage: `4 已完成并结束 workflow`

## Stage Records

### Initialization
- 已读取 `guide.md` 和 `$m-autoflow` 初始化 / 分阶段规则。
- 已确认稳定长期真相位于 `repo/MyFlowHub-Server/docs`。
- 已为 `SubProto` 与 `Server` 分别创建独立 worktree 和 `feat/transform-node` 分支。
- 因控制仓存在未提交的 flow/docs 基线改动，本轮 worktree 先从控制面复制当前 `flow/*` 与 `Server docs` 基线文件，避免直接在控制仓实现。

### Stage 1 - Requirements Analysis
#### Goal
- 为 `flow` 新增正式纯计算节点 `transform`，用显式、白名单式表达式树支持数值、布尔、字符串以及 object/array 结构化运算。

#### Scope
- Must:
  - 新增 `kind=transform`
  - 支持 `literal` / `source` / `object` / `array` / `op + args`
  - `transform.source` 复用现有 binding source，并支持同级 `required`
  - 更新 `MyFlowHub-Server/docs` 中 `flow` requirements/specs
  - 补齐 graph/runtime/regression 测试
- Optional:
  - 为后续扩运算族保留集中分发点
- Out of Scope:
  - 任意脚本执行
  - 自由表达式字符串
  - 用户自定义函数
  - Win Flow 编辑器表单支持

#### Acceptance Criteria
1. `transform(add)` 可实现“上游数字 + 1”。
2. 同一 `transform` 节点可嵌套 object/array 与多层运算。
3. 非法 op / 参数个数 / source 在 `set` 阶段被拒绝。
4. 运行时类型不匹配、除零、必填来源缺失会明确失败。

### Stage 2 - Architecture Design
#### Overall Solution
- `MyFlowHub-SubProto/flow/runtime_bindings.go`
  - 增加 transform spec / expr decode、shape validation、source validation、evaluator
- `MyFlowHub-SubProto/flow/handler.go`
  - 接入 `transform` 执行分支和 graph 校验入口
- `MyFlowHub-Server/docs`
  - 更新 requirements/specs，正式纳入 `transform`

#### Key Decisions
- 采用结构化表达式树，不采用表达式字符串。
- `transform` 只产出节点结果，不直接写 `varstore`。
- `required=false` 挂在 `source` 所在表达式节点同级，不挂在 `source` 对象内部。

### Stage 3.1 - Planning
#### Task IDs
- `TR-DOC-1`
- `TR-RT-1`
- `TR-RT-2`
- `TR-TEST-1`

#### Executable Checklist
- [x] `TR-DOC-1` 更新稳定 requirements/specs
- [x] `TR-RT-1` 新增 transform spec / validation
- [x] `TR-RT-2` 新增 transform runtime / handler 接入
- [x] `TR-TEST-1` 补 graph/runtime/regression 测试
- [x] Stage 3.3 自审
- [x] Stage 4 归档

### Stage 3.2 - Implementation
#### SubProto Write Set
- `flow/runtime_bindings.go`
- `flow/handler.go`
- `flow/graph_test.go`
- `flow/transform_test.go`
- `docs/change/README.md`
- `docs/change/2026-04-03_flow-transform-node-runtime.md`

#### Server Write Set
- `docs/requirements/flow_data_dag.md`
- `docs/specs/flow.md`

#### Validation
- transform 定向测试：
  - workspace：`D:\project\MyFlowHub3\.tmp\verify-transform-node\go.work`
  - `go test -run 'TestValidateGraphRejectsLegacyKind|TestValidateGraphAllowsTransformNode|TestValidateGraphRejectsTransformUnknownOp|TestValidateGraphRejectsTransformInvalidArity|TestValidateGraphRejectsTransformLoopSourceOutsideForeach|TestExecuteFlow_TransformAddsNodeResultNumber|TestExecuteFlow_TransformForeachBuildsNestedObjectArray|TestExecuteFlow_TransformCoalesceOptionalSource|TestExecuteFlow_TransformFailsOnRuntimeErrors' -count=1`
  - 结果：通过
- 邻近回归：
  - `go test -run 'TestValidateGraphOK|TestExecuteFlow_BindsAncestorResultsAndCompose|TestExecuteFlow_SetsAndReadsLocalVars|TestValidateTrigger|TestCronScheduleNextAfter|TestExecuteFlow_BranchSkipsUnselectedPath|TestExecuteFlow_ForeachAggregatesResults|TestExecuteFlow_SubflowReturnsResult' -count=1`
  - 结果：通过
- 全量 `flow`：
  - `go test -count=1`
  - 结果：失败
  - 已知基线失败：`TestLoadFlowsFromDiskKeepsLegacyKindsForCompatibility`、`TestFlowDeleteFileFailureKeepsState`
  - 结论：与本轮 `transform` 无关

### Stage 3.3 - Code Review
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
- 子Agent治理与审计：通过（本轮未使用子Agent）

### Stage 4 - Change Archive
- `MyFlowHub-SubProto/docs/change/2026-04-03_flow-transform-node-runtime.md`
- root `docs/change/2026-04-03_flow-transform-node-runtime.md`
- root `docs/change/2026-04-03_server-flow-transform-node-contract.md`

## Final Outcome
- `flow` 主线正式支持 `transform`。
- `MyFlowHub-Server/docs` 已纳入 `transform` 的长期 requirements/specs。
- workflow 已结束，结果已回灌控制仓并归档到 workspace 根索引。
