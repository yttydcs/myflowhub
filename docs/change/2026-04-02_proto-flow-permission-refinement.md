# 2026-04-02_proto-flow-permission-refinement

## 变更背景 / 目标

- `flow` 运行时已经开始区分“运行控制”和“只读观测”，Proto 需要提供稳定权限常量供下游共享。
- 本轮目标是在 Proto 中补齐 `flow.run` / `flow.read` 常量，并同步 canonical `protocol_map`。

## 具体变更内容

### 修改

- `protocol/flow/types.go`
  - 新增 `PermFlowRun = "flow.run"`
  - 新增 `PermFlowRead = "flow.read"`
- `docs/protocol_map.md`
  - 重新生成 Flow 常量映射，反映 `PermFlowRun` / `PermFlowRead`

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
- `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md`
- `D:\project\MyFlowHub3\docs\specs\protocol_map.md`

## Related lessons

- 无

## 对应 plan.md 任务映射

- `RC-P0-3`
  - Proto 权限常量
  - canonical protocol map sync

## 经验 / 教训摘要

- 只在 SubProto 内写死权限字符串会让 capability 描述、协议映射和下游消费继续漂移。
- Flow 权限常量进入 Proto 后，后续新增更细粒度权限时才有统一落点。

## 可复用排查线索

- 症状：
  - 下游模块找不到 `PermFlowRun` 或 `PermFlowRead`
  - `docs/protocol_map.md` 缺少 `flow.run` / `flow.read`
- 触发条件：
  - 修改了 `protocol/flow/types.go` 但未重新生成 `protocol_map`
- 关键词 / 错误文本：
  - `PermFlowRun`
  - `PermFlowRead`
  - `protocol_map`
- 快速检查：
  1. 看 `protocol/flow/types.go` 是否导出两个权限常量
  2. 看 `docs/protocol_map.md` 的 Flow 常量列表是否包含新权限
  3. 看 root workspace `docs/specs/protocol_map.md` 是否已同步副本

## 关键设计决策与权衡

- 采用两个稳定常量，而不是继续在下游模块各自维护字符串
  - 好处：统一协议入口，减少硬编码
  - 代价：后续权限矩阵继续扩展时，需要谨慎控制常量数量

## 测试与验证方式 / 结果

- `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1`
  - `$env:GOWORK='off'; go run ./cmd/protocolmapgen -write -out docs/protocol_map.md`
  - `$env:GOWORK='off'; go run ./cmd/protocolmapgen -check -out docs/protocol_map.md`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
- 结果：通过

## 潜在影响

- 未同步新版 Proto 的下游模块将无法复用 `PermFlowRun` / `PermFlowRead` 常量。

## 回滚方案

1. 回退 `protocol/flow/types.go`
2. 重新执行 `go run ./cmd/protocolmapgen -write -out docs/protocol_map.md`
3. 重新执行 `go test ./... -count=1 -p 1`

## 子Agent执行轨迹

- 本轮未使用子Agent
