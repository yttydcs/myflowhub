# 2026-04-02 Proto Flow Detail

## 变更背景 / 目标

- `flow.detail` 已经在 Win 本地类型和旧 SubProto dirty worktree 中存在，但 `MyFlowHub-Proto` 主线仍缺少正式协议字典。
- 本轮目标是把 `detail/detail_resp` 契约补回 clean branch，避免继续依赖本地 ad hoc payload。

## 具体变更内容

### 新增

- `protocol/flow/types.go`
  - 新增 `ActionDetail` / `ActionDetailResp`
  - 新增 `DetailReq` / `DetailResp`

### 修改

- `docs/protocol_map.md`
  - 通过 `protocolmapgen` 重新生成 flow action / payload 映射

### 删除

- 无

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `none`

## Related requirements

- `repo/MyFlowHub-Server/docs/requirements/flow_data_dag.md`

## Related specs

- `repo/MyFlowHub-Server/docs/specs/flow.md`

## Related lessons

- 无

## 对应 plan.md 任务映射

- `PROTO-DTL-1`
  - `protocol/flow/types.go`
  - `docs/protocol_map.md`
- `PROTO-VAL-1`
  - `go test ./...`
  - `git diff --check`

## 经验 / 教训摘要

- `flow.detail` 这类协议入口一旦先在消费端落本地类型，就会迅速造成 Proto/SubProto/Win 分叉。
- 协议字典应先回到 `MyFlowHub-Proto`，再让下游通过 alias 使用。

## 可复用排查线索

- 症状：
  - `undefined: protocol.ActionDetail`
  - `undefined: protocol.DetailReq`
  - `docs/protocol_map.md is out of date`
- 触发条件：
  - SubProto 引用了新的 detail action，但 Proto 未补契约
  - 修改 `protocol/*/types.go` 后未重跑 `protocolmapgen`
- 关键词 / 错误文本：
  - `ActionDetail`
  - `DetailReq`
  - `DetailResp`
  - `docs/protocol_map.md is out of date`
- 快速检查：
  1. 看 `protocol/flow/types.go` 是否包含 `ActionDetail` / `DetailReq`
  2. 运行 `go run ./cmd/protocolmapgen -check -out docs/protocol_map.md`

## 关键设计决策与权衡

- 采用共享 Proto 类型，不继续扩大 Win 本地 detail payload 的使用范围
  - 好处：Proto/SubProto/Win 共用同一份 JSON 契约
  - 代价：需要同步更新 canonical `docs/protocol_map.md`

## 测试与验证方式 / 结果

- `GOWORK=off go run ./cmd/protocolmapgen -write -out docs/protocol_map.md`
  - 结果：通过
- `GOWORK=off go test ./...`
  - 结果：通过
- `git diff --check`
  - 结果：通过

## 潜在影响

- 下游 `SubProto` / Win 可以直接消费共享 `detail` 契约
- 若下游仍保留本地 detail 类型，短期不会立刻坏，但会继续增加漂移风险

## 回滚方案

1. 回退 `protocol/flow/types.go` 中的 `detail` action 和 payload
2. 重新运行 `protocolmapgen` 恢复 `docs/protocol_map.md`

## 子Agent执行轨迹

- 本轮未使用子Agent
