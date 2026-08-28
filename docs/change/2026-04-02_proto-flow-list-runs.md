# 2026-04-02_proto-flow-list-runs

## 变更背景 / 目标

- Flow 当前只有 `status/detail/list`，但没有专门的 run history wire contract。
- 本轮目标是在 Proto 仓补齐 `list_runs` 动作、payload 和 canonical `protocol_map`。

## 具体变更内容（新增 / 修改 / 删除）

### 修改

- `protocol/flow/types.go`
  - 新增 `ActionListRuns` / `ActionListRunsResp`
  - 新增 `ListRunsReq` / `ListRunsResp`
  - 新增 `RunSummary`
- `docs/protocol_map.md`
  - 通过 `protocolmapgen` 生成 Flow 的新 action / payload 映射

### 删除

- 无

## Requirements impact

- none

## Specs impact

- none

## Lessons impact

- none

## 对应 plan / todo 任务映射

- `RC-P0-2`
  - Proto wire contract
  - canonical protocol map sync

## 关键设计决策与权衡

- `RunSummary` 只给摘要字段，不并入完整 runtime context。
- 时间字段使用 `started_at_ms` / `ended_at_ms`
  - 好处：单位明确，避免下游猜测时间格式
  - 代价：相较 RFC3339 可读性稍弱

## 测试与验证方式 / 结果

- `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1`
  - `$env:GOWORK='off'; go run ./cmd/protocolmapgen -write -out docs/protocol_map.md`
  - `$env:GOWORK='off'; go run ./cmd/protocolmapgen -check -out docs/protocol_map.md`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
- 结果：通过

## 潜在影响

- 下游若未同步新版 Proto，将无法识别 `list_runs` / `list_runs_resp`。

## 回滚方案

1. 回退 `protocol/flow/types.go`
2. 重新运行 `go run ./cmd/protocolmapgen -write -out docs/protocol_map.md`
3. 重新执行 `go test ./... -count=1 -p 1`
