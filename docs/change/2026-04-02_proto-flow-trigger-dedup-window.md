# 2026-04-02_proto-flow-trigger-dedup-window

## 变更背景 / 目标

- `flow` 前面已经补齐 active-run limit，但 trigger 侧仍缺少对短窗口重复启动的显式协议入口。
- 本轮目标是在 Proto 层为 `Trigger` 增加可选字段 `dedup_window_ms`，供下游显式声明 trigger dedup 窗口，同时保持默认关闭。

## 具体变更内容

### 修改

- `protocol/flow/types.go`
  - `Trigger` 新增 `DedupWindowMs *int \`json:"dedup_window_ms,omitempty"\``

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

- `RC-P1-3`
  - Proto `Trigger.dedup_window_ms` field

## 经验 / 教训摘要

- trigger dedup 需要区分“未设置”和“显式 0”，因此仍应使用指针字段表达默认关闭语义。
- 这次只是已有 struct 的字段扩展，不涉及 action/type/constant 清单变化，因此 `protocol_map` 无需改动。

## 可复用排查线索

- 症状：
  - 下游定义里写了 `dedup_window_ms`，但反序列化后字段丢失
  - `flow get` 或 `set`/持久化链路无法保留 dedup 配置
- 触发条件：
  - 只改了运行时或文档，没有先扩展 Proto `Trigger`
- 关键词 / 错误文本：
  - `dedup_window_ms`
  - `DedupWindowMs`
- 快速检查：
  1. 看 `protocol/flow/types.go` 的 `Trigger` 是否声明 `DedupWindowMs`
  2. 运行 `go test ./... -count=1 -p 1`

## 关键设计决策与权衡

- 将 dedup 作为 trigger 级字段，而不是 flow 级字段
  - 好处：只对实际使用 trigger 的 flow 生效，且能明确绑定到具体 trigger 语义
  - 代价：若未来出现多 trigger 组合，仍需继续扩展表达方式

## 测试与验证方式 / 结果

- `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
- 结果：通过

## 潜在影响

- 升级到新版 Proto 的下游模块可以稳定读写 `trigger.dedup_window_ms`。
- 尚未同步新版 Proto 的下游模块仍无法识别该字段。

## 回滚方案

1. 回退 `protocol/flow/types.go`
2. 重新执行 `go test ./... -count=1 -p 1`

## 子Agent执行轨迹

- 本轮未使用子Agent
