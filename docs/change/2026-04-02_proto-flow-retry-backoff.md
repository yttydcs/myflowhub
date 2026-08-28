# 2026-04-02_proto-flow-retry-backoff

## 变更背景 / 目标

- `flow` 需要为节点失败重试提供稳定的 backoff 字段，Proto 必须先定义 wire contract。
- 本轮目标是在 `Node` 上新增 `retry_backoff_ms` 可选字段。

## 具体变更内容

### 修改

- `protocol/flow/types.go`
  - `Node` 新增 `RetryBackoffMs *int \`json:"retry_backoff_ms,omitempty"\``

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

- `RC-P1-1`
  - Proto `Node` retry backoff field

## 经验 / 教训摘要

- `protocol_map` 当前只映射 action / type / constant，不覆盖 struct 字段明细；因此字段级扩展不一定带来 canonical map 正文变化。
- 即便 `docs/protocol_map.md` 无变化，Proto 仍应通过 `go test ./...` 回归确认生成器门禁不受影响。

## 可复用排查线索

- 症状：
  - 下游编译时找不到 `RetryBackoffMs`
  - graph JSON 带 `retry_backoff_ms` 但 Proto 结构无法反序列化
- 触发条件：
  - 先改了 runtime/docs，没有先改 Proto `Node`
- 关键词 / 错误文本：
  - `retry_backoff_ms`
  - `RetryBackoffMs`
- 快速检查：
  1. 看 `protocol/flow/types.go` 的 `Node` 是否已声明 `RetryBackoffMs`
  2. 运行 `go test ./... -count=1 -p 1`

## 关键设计决策与权衡

- 只新增单个可选字段，不引入策略对象
  - 好处：向后兼容、落点清晰
  - 代价：更复杂的退避策略需要后续再扩

## 测试与验证方式 / 结果

- `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
- 结果：通过

## 潜在影响

- 新版 Proto 可接受 `retry_backoff_ms`；旧版下游若未同步，仍无法识别该字段。

## 回滚方案

1. 回退 `protocol/flow/types.go`
2. 重新执行 `go test ./... -count=1 -p 1`

## 子Agent执行轨迹

- 本轮未使用子Agent
