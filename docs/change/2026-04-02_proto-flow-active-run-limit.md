# 2026-04-02_proto-flow-active-run-limit

## 变更背景 / 目标

- `flow` 运行时已经补到显式 `cancel_run`、`list_runs`、run/read 权限与 retry backoff，但“活动 run 上限”仍停留在隐式行为。
- 本轮目标是在 Proto 层为 flow 定义增加可选字段 `max_active_runs`，让下游可以稳定表达 active-run limit，同时保持旧 flow 的兼容语义。

## 具体变更内容

### 修改

- `protocol/flow/types.go`
  - `SetReq` 新增 `MaxActiveRuns *int \`json:"max_active_runs,omitempty"\``
  - `GetResp` 新增 `MaxActiveRuns *int \`json:"max_active_runs,omitempty"\``

### 删除

- 无

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `none`

## Related requirements

- `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\requirements\flow_data_dag.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\server-run-control-phase1\docs\specs\flow.md`

## Related lessons

- 无

## 对应 plan.md 任务映射

- `RC-P1-2`
  - Proto `SetReq/GetResp` active-run limit field

## 经验 / 教训摘要

- `nil`、`0`、`>0` 三种语义必须通过指针字段区分，不能退回普通 `int` 默认值。
- 这次是字段级扩展，不会触发 `protocol_map` 的 action/type/constant 清单变化；但仍需要跑 Proto 回归，确保生成器门禁不受影响。

## 可复用排查线索

- 症状：
  - 下游反序列化时读不到 `max_active_runs`
  - `get` 读回 flow 定义时丢失 active-run limit 配置
- 触发条件：
  - 只修改了运行时和文档，没有先改 Proto `SetReq/GetResp`
- 关键词 / 错误文本：
  - `max_active_runs`
  - `MaxActiveRuns`
- 快速检查：
  1. 看 `protocol/flow/types.go` 的 `SetReq` 是否声明 `MaxActiveRuns`
  2. 看 `protocol/flow/types.go` 的 `GetResp` 是否声明 `MaxActiveRuns`
  3. 运行 `go test ./... -count=1 -p 1`

## 关键设计决策与权衡

- 采用单个可选字段 `max_active_runs`
  - 好处：兼容旧 flow，并保留未来扩展到 `2/3/...` 并发上限的空间
  - 代价：更复杂的 queue / cancel_previous 策略仍需后续迭代单独表达

## 测试与验证方式 / 结果

- `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
- 结果：通过

## 潜在影响

- 同步到新版 Proto 的下游模块可以稳定读写 `max_active_runs`。
- 尚未同步新版 Proto 的下游模块仍无法识别该字段。

## 回滚方案

1. 回退 `protocol/flow/types.go`
2. 重新执行 `go test ./... -count=1 -p 1`

## 子Agent执行轨迹

- 本轮未使用子Agent
