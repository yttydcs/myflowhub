# 2026-04-02_proto-flow-cancel-run

## 变更背景 / 目标

- Flow wire 已支持 `delete/detail`，但还没有显式 `cancel_run` 动作，无法为下游提供稳定的单 run 取消契约。
- 本轮目标是在 Proto 仓补齐 `cancel_run` 常量与 payload，并更新 canonical `protocol_map`。

## 具体变更内容（新增 / 修改 / 删除）

### 修改

- `protocol/flow/types.go`
  - 新增 `ActionCancelRun` / `ActionCancelRunResp`
  - 新增 `CancelRunReq` / `CancelRunResp`
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

- `RC-P0-1`
  - Proto wire contract
  - canonical protocol map sync

## 关键设计决策与权衡

- 首轮只增加 `cancel_run` 所需的最小字段集合，不在 Proto 层新增新的权限常量。
- `CancelRunResp` 单独回显 `status`，避免调用方必须再追加一次 `status` 查询才能知道取消是否生效。

## 测试与验证方式 / 结果

- `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1`
  - `$env:GOWORK='off'; go run ./cmd/protocolmapgen -write -out docs/protocol_map.md`
  - `$env:GOWORK='off'; go run ./cmd/protocolmapgen -check -out docs/protocol_map.md`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
- 结果：通过

## 潜在影响

- 下游若仍使用旧版 Proto，将无法识别 `cancel_run` / `cancel_run_resp`。

## 回滚方案

1. 回退 `protocol/flow/types.go`
2. 重新运行 `go run ./cmd/protocolmapgen -write -out docs/protocol_map.md`
3. 重新执行 `go test ./... -count=1 -p 1`
