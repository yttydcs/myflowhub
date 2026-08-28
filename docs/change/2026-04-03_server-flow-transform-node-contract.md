# 2026-04-03 Server Flow Transform Node Contract

## 变更背景 / 目标

- `MyFlowHub-Server/docs` 先前已经收口 `branch / foreach / subflow / cron`，但稳定 requirements/specs 仍缺少正式的纯计算节点描述。
- 本轮目标是把 `transform` 纳入 `flow` 的长期 requirements/specs，确保 `Server docs` 继续作为跨仓稳定真相。

## 具体变更内容

### 修改

- `repo/MyFlowHub-Server/docs/requirements/flow_data_dag.md`
  - 增加 `transform` 背景、目标、范围、功能需求和验收标准
  - 明确 `transform` 是无脚本、白名单式表达式树
- `repo/MyFlowHub-Server/docs/specs/flow.md`
  - 把 `transform` 纳入正式节点种类
  - 新增 `TransformExpr` 模型、白名单运算、`required` 语义与运行边界
  - 更新 graph validation 和执行语义说明

### 删除

- 无

## Requirements impact

- `updated`

## Specs impact

- `updated`

## Lessons impact

- `none`

## Related requirements

- `repo/MyFlowHub-Server/docs/requirements/flow_data_dag.md`

## Related specs

- `repo/MyFlowHub-Server/docs/specs/flow.md`

## Related lessons

- 无

## 对应 plan.md 任务映射

- `TR-DOC-1`
  - `repo/MyFlowHub-Server/docs/requirements/flow_data_dag.md`
  - `repo/MyFlowHub-Server/docs/specs/flow.md`

## 经验 / 教训摘要

- 对 `flow` 这类跨仓能力，长期真相必须先回到 `MyFlowHub-Server/docs`，不能把 runtime change log 误当成稳定规范。
- `transform` 的 DSL 如果不在 requirements/specs 中先明确禁止脚本执行，后续极易被误扩展成隐式脚本引擎。

## 可复用排查线索

- 症状：
  - 主线 specs 查不到 `transform`
  - 调用方误把 `required` 写进 `source` 对象内部
  - 编辑器或调用方对 `transform` 支持的运算范围理解不一致
- 触发条件：
  - runtime 已落地，但稳定 docs 未同步
  - 调用方只参考旧 change log 或聊天记录，不看 `Server docs`
- 关键词：
  - `TransformExpr`
  - `required=false`
  - `无脚本`
  - `白名单运算`
- 快速检查：
  1. 看 `docs/specs/flow.md` 是否包含 `Transform 表达式模型`
  2. 看 `docs/requirements/flow_data_dag.md` 是否把 `transform` 纳入 Must 与验收标准

## 关键设计决策与权衡

- 结构化表达式树
  - 好处：可审计、可校验、跨仓口径稳定
  - 代价：未来新增 op 需要继续补 docs 与 runtime
- `transform` 只做纯计算
  - 好处：不和 `set_var` / `varstore` 副作用混淆
  - 代价：某些链路需要多一个节点显式写回

## 测试与验证方式 / 结果

- 文档与 `SubProto` runtime 实现对照检查：通过
- root workflow 定向回归：
  - `D:\project\MyFlowHub3\.tmp\verify-transform-node\go.work`
  - transform 定向测试与邻近回归均通过
- `git diff --check`：通过

## 潜在影响

- 后续 `Proto/SubProto/Win` 讨论 `transform` 时，统一以 `repo/MyFlowHub-Server/docs/specs/flow.md` 为真相入口
- 编辑器若补表单，也必须遵循这里定义的 expression tree 和 `required` 层级语义

## 回滚方案

1. 回退 `repo/MyFlowHub-Server/docs/requirements/flow_data_dag.md`
2. 回退 `repo/MyFlowHub-Server/docs/specs/flow.md`

## 子Agent执行轨迹

- 本轮未使用子Agent
