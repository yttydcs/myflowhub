# 2026-04-03 Flow Transform Node Runtime

## 变更背景 / 目标

- `flow` 已具备数据流 DAG、局部变量、run control 与高阶编排节点，但仍缺少正式的纯计算节点。
- 本轮目标是在 `MyFlowHub-SubProto/flow` 主线补齐 `transform`，让 flow 可在图内直接完成结构化运算，而不是继续依赖外部 capability 或调用方预处理。

## 具体变更内容

### 新增

- `repo/MyFlowHub-SubProto/flow/transform_test.go`
  - 覆盖 transform graph validation、数值加一、`foreach` 内 nested object/array、`coalesce(required=false)` 与运行时错误路径
- `repo/MyFlowHub-SubProto/docs/change/2026-04-03_flow-transform-node-runtime.md`
  - 记录本仓 runtime 归档

### 修改

- `repo/MyFlowHub-SubProto/flow/runtime_bindings.go`
  - 新增 transform spec / expr decode、shape validation、source validation 和 evaluator
  - 新增白名单运算：`add/sub/mul/div/mod/neg/abs/min/max`、`eq/ne/gt/gte/lt/lte`、`and/or/not/coalesce/if`、`concat/lower/upper/trim`、`len`
- `repo/MyFlowHub-SubProto/flow/handler.go`
  - `executeNode(...)` 支持 `transform`
  - `validateSetNodeKindAndSpec(...)` 接受 `transform`
- `repo/MyFlowHub-SubProto/flow/graph_test.go`
  - legacy kind 错误消息更新为包含 `transform`
- `repo/MyFlowHub-SubProto/docs/change/README.md`
  - 新增 transform runtime 归档索引

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

- `TR-RT-1`
  - `repo/MyFlowHub-SubProto/flow/runtime_bindings.go`
  - `repo/MyFlowHub-SubProto/flow/handler.go`
  - `repo/MyFlowHub-SubProto/flow/graph_test.go`
- `TR-RT-2`
  - `repo/MyFlowHub-SubProto/flow/runtime_bindings.go`
  - `repo/MyFlowHub-SubProto/flow/handler.go`
- `TR-TEST-1`
  - `repo/MyFlowHub-SubProto/flow/transform_test.go`

## 经验 / 教训摘要

- `transform` 的正确边界是“纯计算节点”，不要继续把动态计算塞进 `compose` 模板或外部 utility capability。
- `required=false` 应定义在 `source` 所在表达式节点同级，保持和 binding 语义清晰对齐。
- 复用既有 binding source 能让 `flow_var`、`loop_item/loop_index` 与祖先校验规则继续保持一致。

## 可复用排查线索

- 症状：
  - `op unsupported`
  - `requires exactly`
  - `required source missing`
  - `loop_item only allowed in foreach body`
  - `divide by zero`
- 触发条件：
  - 表达式同时声明多个变体
  - `transform` 使用未知运算或错误参数个数
  - `required=false` 写到了错误层级
  - 在非 `foreach.body` 环境引用 `loop_item/loop_index`
- 关键词：
  - `must define exactly one of literal, source, op, object or array`
  - `required source missing`
  - `requires number`
  - `divide by zero`
- 快速检查：
  1. 看 `runtime_bindings.go` 是否存在 `decodeNodeTransformSpec`、`evaluateTransformExpr`
  2. 看 `handler.go` 是否在 `executeNode` 和 graph validation 接入 `transform`
  3. 看 stable specs 是否明确 `required` 是表达式节点同级字段

## 关键设计决策与权衡

- 结构化表达式树而非字符串表达式
  - 好处：可审计、可静态校验、无脚本注入面
  - 代价：首版表达能力受白名单限制
- `transform` 只产出节点结果，不直接写局部变量
  - 好处：职责清晰，可和 `set_var` / `call` 组合
  - 代价：写回中间值时需要额外节点

## 测试与验证方式 / 结果

- transform 定向测试：
  - workspace：`D:\project\MyFlowHub3\.tmp\verify-transform-node\go.work`
  - 结果：通过
- 邻近回归测试：
  - 同 workspace 下 `compose / set_var / branch / foreach / subflow / cron` 邻近用例
  - 结果：通过
- 全量 `flow` 包：
  - 结果：失败
  - 已知失败：`TestLoadFlowsFromDiskKeepsLegacyKindsForCompatibility`、`TestFlowDeleteFileFailureKeepsState`
  - 判断：与本轮无关

## 潜在影响

- `flow set` 正式接受 `kind=transform`
- `foreach.body` 现在可直接在图内消费 `loop_item/loop_index` 做结构化变换
- 编辑器尚未补 transform 表单时，仍需先用高级 JSON 模式写入

## 回滚方案

1. 回退 `repo/MyFlowHub-SubProto/flow/handler.go`
2. 回退 `repo/MyFlowHub-SubProto/flow/runtime_bindings.go`
3. 回退 `repo/MyFlowHub-SubProto/flow/transform_test.go` 与 graph 测试相关改动
4. 同步回退 `repo/MyFlowHub-Server/docs` 中的 transform requirements/specs

## 子Agent执行轨迹

- 本轮未使用子Agent
