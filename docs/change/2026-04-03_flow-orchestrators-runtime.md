# 2026-04-03_flow-orchestrators-runtime

## 变更背景 / 目标

- 当前 `flow` 已支持数据流 DAG、`set_var`、`detail`、run archive、active-run limit 和 trigger dedup，但仍缺少真正的高阶编排能力。
- 本轮目标是在不引入脚本执行和跨执行器复杂度的前提下，为运行时补齐：
  - `branch`
  - `foreach`
  - `subflow`
  - `trigger.type=cron`

## 具体变更内容

### 修改

- `flow/handler.go`
  - `validateTrigger(...)` 新增 `cron` 校验
  - `applySetLocal(...)` / `loadFlowsFromDisk()` 改为复用 `validateGraphForFlow(...)`
  - `executeFlow(...)` 抽为可复用的 `executeGraph(...)`
  - 新增 branch 路由判定和 `skipped` 节点状态
  - `executeNode(...)` 支持 `branch` / `foreach` / `subflow`
  - `restartScheduler(...)` 同时支持 `interval` 与 `cron`
  - 新增 cron 解析与下一次触发时间计算 helper
- `flow/runtime_bindings.go`
  - `runContext` 新增 loop 上下文
  - `bindingSource` 新增 `loop_item` / `loop_index` 运行时解析
  - 新增 `branchSpec` / `foreachSpec` / `subflowSpec`
  - 新增 branch 匹配求值、subflow trigger context、foreach body 校验与 edge index helper
- `flow/graph_test.go`
  - 更新节点种类错误提示断言
- `flow/trigger_test.go`
  - 新增 `cron` trigger 校验
  - 新增 cron `NextAfter(...)` 覆盖
- `flow/orchestrator_test.go`
  - 新增 branch edge 校验
  - 新增 loop source 校验
  - 新增 branch skip / foreach 聚合 / subflow 返回结果运行时用例

### 删除

- 无

## Requirements impact

- `updated`

## Specs impact

- `updated`

## Lessons impact

- `none`

## Related requirements

- `D:\project\MyFlowHub3\worktrees\server-flow-orchestrators\docs\requirements\flow_data_dag.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\server-flow-orchestrators\docs\specs\flow.md`

## Related lessons

- `D:\project\MyFlowHub3\worktrees\subproto-flow-orchestrators\docs\lessons\flow-trigger-run-missing-server-context.md`

## 对应 plan.md 任务映射

- `ORCH-RT-1` -> `flow/handler.go`, `flow/runtime_bindings.go`
- `ORCH-RT-2` -> `flow/handler.go`, `flow/runtime_bindings.go`
- `ORCH-RT-3` -> `flow/handler.go`, `flow/runtime_bindings.go`, `flow/orchestrator_test.go`
- `ORCH-RT-4` -> `flow/handler.go`, `flow/trigger_test.go`
- `ORCH-TEST-1` -> `flow/graph_test.go`, `flow/trigger_test.go`, `flow/orchestrator_test.go`

## 经验 / 教训摘要

- `branch` 首版采用 `edge.case` 路由，比内嵌子图更容易和现有 DAG merge 语义对齐。
- `foreach` 首版采用隔离 child context，不让迭代内 `set_var` 泄漏到外层，先保证边界清晰。
- `subflow` 首版直接复用内存执行器，不生成顶层 retained run，避免把编排能力和观测面意外耦合。

## 可复用排查线索

- 症状:
  - branch 节点执行后，后续节点没有按预期跳过或 merge
  - foreach 运行成功但结果数组顺序不对
  - subflow 命中自身或递归链时报错
  - cron flow 不触发
- 触发条件:
  - `edge.case` 缺失或与 branch case 名不一致
  - foreach source 不是数组
  - subflow 指向不存在的 flow 或递归调用
  - cron 表达式非法
- 关键词:
  - `edge.case`
  - `loop_item`
  - `loop_index`
  - `subflow recursion detected`
  - `parseCronExpr`
- 快速检查:
  - 先看 `validateGraphForFlow(...)` 是否已拒绝非法图
  - 再看 `executeGraph(...)` 是否把未选中路径标成 `skipped`
  - 再看 `restartScheduler(...)` 是否进入了 `cron` 分支

## 关键设计决策与权衡

- `branch` 使用规则匹配 + `edge.case`，不引入脚本执行
  - 好处: 契约可静态校验，执行期只需边筛选
  - 代价: 条件表达力保持在最小集合
- `foreach` 保持串行
  - 好处: 顺序稳定、资源可控、错误传播清晰
  - 代价: 不追求吞吐
- `subflow` 采用同执行器同步调用
  - 好处: 实现面最小，不需要扩协议动作
  - 代价: 首版不支持跨执行器拆分
- `cron` 采用内置 5-field 解析器
  - 好处: 审计简单，无第三方依赖
  - 代价: 只覆盖首版最小语义

## 测试与验证方式 / 结果

- `Proto`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - 结果: 通过
- `SubProto/flow` 定向验证:
  - 临时 workspace: `D:\project\MyFlowHub3\.tmp\verify-flow-orchestrators\go.work`
  - `go test github.com/yttydcs/myflowhub-subproto/flow -run 'TestValidateTrigger|TestCronScheduleNextAfter|TestValidateGraphRejectsBranchEdgeWithoutCase|TestValidateGraphRejectsLoopItemOutsideForeach|TestExecuteFlow_BranchSkipsUnselectedPath|TestExecuteFlow_ForeachAggregatesResults|TestExecuteFlow_SubflowReturnsResult' -count=1 -p 1`
  - 结果: 通过
- `SubProto/flow` 扩展定向回归:
  - `go test github.com/yttydcs/myflowhub-subproto/flow -run 'TestValidateGraph|TestExecuteFlow|TestHandleTopic|TestHandleVarChanged|TestTryStart|TestCronScheduleNextAfter' -count=1 -p 1`
  - 结果: 通过
- `SubProto/flow` 全量包回归:
  - `go test github.com/yttydcs/myflowhub-subproto/flow -count=1 -p 1`
  - 结果: 失败
  - 已知失败: `delete/detail/run/status/list_runs/get` 相关权限基线仍返回 `403`
  - 判断: 属于当前基线问题，不是本轮 orchestration 改动引入的新增失败类型

## 潜在影响

- 新 flow 定义现在可声明 `branch` / `foreach` / `subflow` / `cron`。
- branch 未选中的节点会显式进入 `skipped` 状态。
- foreach body graph 只在迭代内生效，不把局部变量回写到外层。
- cron 调度按执行器本地时区解释表达式，但不支持时区字段和补跑。

## 回滚方案

1. 回退 `flow/handler.go` 与 `flow/runtime_bindings.go`
2. 回退 `flow/graph_test.go`、`flow/trigger_test.go`、`flow/orchestrator_test.go`
3. 恢复仅支持 `call/compose/set_var` 和 `interval/event/var_changed` 的旧运行时

## 子Agent执行轨迹

- 本轮未使用子Agent
