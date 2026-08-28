# 2026-04-03_proto-flow-orchestrators

## 变更背景 / 目标

- Flow 运行时本轮需要支持高阶编排节点与 `cron` trigger。
- Proto 层只需要补最小 wire 字段，不把运行时执行细节写入协议字典。

## 具体变更内容

### 修改

- `protocol/flow/types.go`
  - `Trigger` 新增 `Cron string \`json:"cron,omitempty"\``
  - `Edge` 新增 `Case string \`json:"case,omitempty"\``

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

- 无

## 对应 plan.md 任务映射

- `ORCH-PROTO-1` -> `protocol/flow/types.go`

## 经验 / 教训摘要

- 高阶编排首版在 Proto 层只补 `cron` 和 `edge.case` 两个字段，保持 `Node.Spec` 继续承载节点专属 JSON。

## 可复用排查线索

- 关键词:
  - `Trigger.Cron`
  - `Edge.Case`
- 快速检查:
  - 若下游拿不到 `cron` / `case` 字段，先确认是否已切到本次 Proto 分支或版本

## 关键设计决策与权衡

- 不为 `branch/foreach/subflow` 新增顶级 struct
  - 好处: 保持 Proto 改动面最小
  - 代价: 更具体的节点 spec 约束留在稳定 specs 和运行时校验层

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./... -count=1 -p 1`
- 结果: 通过

## 潜在影响

- 下游可以通过 `Trigger.Cron` 表达 cron 触发器
- 下游可以通过 `Edge.Case` 表达 branch 路由边

## 回滚方案

1. 回退 `protocol/flow/types.go`
2. 恢复只含 `every_ms` 的 trigger 与无条件 edge 模型

## 子Agent执行轨迹

- 本轮未使用子Agent
