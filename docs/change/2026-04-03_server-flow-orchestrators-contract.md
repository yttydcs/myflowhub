# 2026-04-03_server-flow-orchestrators-contract

## 变更背景 / 目标

- 稳定文档此前仍把 `branch/foreach` 视为未来扩展，尚未覆盖 `subflow` 与 `cron`。
- 本轮目标是把 Flow 高阶编排与 `cron` trigger 正式纳入 `requirements/specs`，作为 Proto/SubProto 的长期真相。

## 具体变更内容

### 修改

- `docs/requirements/flow_data_dag.md`
  - 新增高阶编排背景与目标
  - 将 `branch` / `foreach` / `subflow` / `cron` 纳入 Must
  - 调整 Not In Scope 为脚本、并行 foreach、cross-executor subflow、cron 时区/补跑
  - 增加场景、功能需求、边界异常和验收标准
- `docs/specs/flow.md`
  - 触发器定义新增 `cron`
  - 图模型新增 `edge.case`
  - 输入绑定新增 `loop_item` / `loop_index`
  - 正式节点类型新增 `branch` / `foreach` / `subflow`
  - 执行语义新增 `skipped`、foreach body graph、subflow child flow 和 cron scheduler 约束

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

- `ORCH-DOC-1` -> `docs/requirements/flow_data_dag.md`, `docs/specs/flow.md`

## 经验 / 教训摘要

- `flow` 从“数据流 DAG”继续扩展到“可编排工作流”时，长期文档必须先把 v1 边界写死，否则 runtime 很容易凭聊天语境漂移。

## 可复用排查线索

- 症状:
  - 文档还把 `branch/foreach` 记成未来扩展
  - 实现已支持 `cron`，但 stable spec 没写
- 关键词:
  - `edge.case`
  - `loop_item`
  - `subflow`
  - `trigger.type=cron`
- 快速检查:
  - 先看 `docs/requirements/flow_data_dag.md`
  - 再看 `docs/specs/flow.md` 的 trigger、graph 和 node sections

## 关键设计决策与权衡

- requirements 把 `branch/foreach/subflow/cron` 纳入 Must
  - 好处: 运行时和编辑器可以围绕稳定目标继续演进
  - 代价: 未来扩 scope 需要继续走文档治理流程
- specs 对 v1 做了明确收口
  - `foreach` 串行
  - `subflow` 同执行器同步
  - `cron` 无时区字段

## 测试与验证方式 / 结果

- 文档对照检查:
  - 与 `MyFlowHub-Proto/protocol/flow/types.go` 新增字段一致
  - 与 `MyFlowHub-SubProto/flow` 新增节点和调度行为一致
- 结果: 通过

## 潜在影响

- 后续实现和前端编辑器都应以这两份稳定文档为准，不再把这些能力视为 future-only。

## 回滚方案

1. 回退 `docs/requirements/flow_data_dag.md`
2. 回退 `docs/specs/flow.md`
3. 恢复“高阶编排节点尚未纳入长期真相”的旧文档口径

## 子Agent执行轨迹

- 本轮未使用子Agent
