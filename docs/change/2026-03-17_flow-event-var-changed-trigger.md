# 变更背景 / 目标

- 背景：`flow` 触发器仅支持 `interval`，无法基于业务事件与变量变更自动启动工作流。
- 目标：在不破坏现有 DAG 执行路径的前提下，新增 `event` 与 `var_changed` 两类触发器，并打通 Win 编辑器配置与服务端运行时触发链路。

# 具体变更内容

## 新增

- 协议层新增触发器字段：
  - `event_name`
  - `event_topic`
  - `var_owner`
  - `var_name`
  - 文件：`repo/MyFlowHub-Proto/protocol/flow/types.go`

- Flow 触发器能力新增：
  - 触发器标准化/校验：支持 `interval` / `event` / `var_changed`
  - 订阅事件总线：`topicbus.publish`、`varstore.changed`、`varstore.deleted`
  - 匹配触发后复用 `tryStartRun` 统一启动门禁
  - 文件：`repo/MyFlowHub-SubProto/flow/handler.go`

- TopicBus 到 Flow 的触发桥接：
  - 在 `publish` 处理路径发出 `topicbus.publish` 事件
  - 文件：`repo/MyFlowHub-SubProto/topicbus/topicbus.go`

- VarStore 到 Flow 的触发桥接：
  - 在变量变更与删除路径发出 `varstore.changed` / `varstore.deleted` 事件
  - 事件发布不依赖订阅者数量，避免漏触发
  - 文件：`repo/MyFlowHub-SubProto/varstore/varstore.go`

- Win 端触发器编辑能力：
  - 新增触发器类型选择：`interval` / `event` / `var_changed`
  - 新增对应字段编辑、保存与回显解析
  - 文件：
    - `repo/MyFlowHub-Win/frontend/src/stores/flow.ts`
    - `repo/MyFlowHub-Win/frontend/src/pages/Flow.vue`

- 测试新增：
  - Flow 触发器校验与匹配用例
  - TopicBus 事件发布用例
  - VarStore 事件发布用例
  - 文件：
    - `repo/MyFlowHub-SubProto/flow/trigger_test.go`
    - `repo/MyFlowHub-SubProto/topicbus/trigger_event_test.go`
    - `repo/MyFlowHub-SubProto/varstore/trigger_event_test.go`

## 修改

- Flow 协议文档补充三类触发器规范与匹配语义：
  - 文件：`repo/MyFlowHub-Server/docs/6-flow.md`

- 本地联调工作区加入 `topicbus` 子模块，保证跨模块增量测试可运行：
  - 文件：`go.work`

## 删除

- 无。

# 对应 plan 任务映射

- Task A：协议与 Flow 触发器扩展  
  - 对应：`repo/MyFlowHub-Proto/protocol/flow/types.go`、`repo/MyFlowHub-SubProto/flow/handler.go`
- Task B：事件源桥接（TopicBus/VarStore）  
  - 对应：`repo/MyFlowHub-SubProto/topicbus/topicbus.go`、`repo/MyFlowHub-SubProto/varstore/varstore.go`
- Task C：Win 编辑器触发器配置能力  
  - 对应：`repo/MyFlowHub-Win/frontend/src/stores/flow.ts`、`repo/MyFlowHub-Win/frontend/src/pages/Flow.vue`
- Task D：文档与测试补齐  
  - 对应：`repo/MyFlowHub-Server/docs/6-flow.md` 与新增测试文件

# 关键设计决策与权衡

- 触发入口统一：新增触发仅扩展“启动条件”，不改 DAG 执行器主路径，降低回归风险。
- 启动门禁复用：`interval/event/var_changed` 都走 `tryStartRun`，保持同一 flow 不并发运行。
- 事件匹配策略：
  - `event`：`event_name` 与 `event_topic` 均可选，但不能同时为空。
  - `var_changed`：`var_owner` 与 `var_name` 均可选，空值视为不过滤。
- 观测优先：在 TopicBus/VarStore 内部事件总线发事件，避免直接耦合 Flow 逻辑。

# 测试与验证方式 / 结果

- Go 测试：
  - `repo/MyFlowHub-Proto`: `go test ./...`（通过）
  - `repo/MyFlowHub-SubProto/flow`: `go test ./...`（通过）
  - `repo/MyFlowHub-SubProto/topicbus`: `go test ./...`（通过）
  - `repo/MyFlowHub-SubProto/varstore`: `go test ./...`（通过）
  - `repo/MyFlowHub-Server`: `go test ./...`（通过）
  - `repo/MyFlowHub-Win`: `go test ./...`（通过）
- 前端构建：
  - `repo/MyFlowHub-Win/frontend`: `npm run build`（通过）

# 潜在影响与回滚方案

- 潜在影响：
  - 高频事件可能导致 Flow 启动尝试增多（由 `tryStartRun` 并发门禁兜底）。
  - `event` 触发当前仅接入 `topicbus.publish` 事件源。
- 回滚方案：
  - 回滚 Flow/TopicBus/VarStore/Win 对应改动文件至前一版本；
  - 协议层可保留字段但不使用，或一并回滚 `Trigger` 字段扩展；
  - 回滚后重新执行上述测试矩阵确认恢复。

# 例外说明

- 本次变更在现有集成工作区内完成，未新建独占 worktree。原因：需要跨 `Proto`、`SubProto`、`Server`、`Win` 多仓同时修改并一次性联调验证。
