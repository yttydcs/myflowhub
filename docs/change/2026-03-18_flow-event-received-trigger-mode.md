# 变更背景 / 目标

- 背景：现有 `flow.event` 触发只覆盖 `topicbus.publish` 场景，无法显式表达“收到下行事件后触发”。
- 目标：为 `flow.event` 增加事件模式，支持 `received`（接收事件）触发，并保持对既有 `publish` 工作流的兼容。

# 具体变更内容（新增 / 修改 / 删除）

## 1) 协议扩展：Trigger 增加 `event_mode`

- 文件：`repo/MyFlowHub-Proto/protocol/flow/types.go`
- 新增字段：
  - `event_mode`：`publish` | `received` | `any`
- 兼容策略：未设置时按 `publish` 处理（旧流程无需修改）。

## 2) Flow 运行时支持事件模式匹配

- 文件：`repo/MyFlowHub-SubProto/flow/handler.go`
- 新增：
  - 事件模式常量与规范化函数；
  - `event_mode` 校验逻辑；
  - 订阅 `topicbus.received` 事件；
  - 事件匹配时按 `event_mode + event_name + event_topic` 联合过滤。

## 3) TopicBus 发出 `topicbus.received`

- 文件：`repo/MyFlowHub-SubProto/topicbus/topicbus.go`
- 新增行为：
  - `publish` 消息由父连接下发到本节点时，向 EventBus 发出 `topicbus.received`。
  - 保留 `topicbus.publish` 事件，不破坏原触发链路。

## 4) Win Flow 编辑器支持 `event_mode`

- 文件：
  - `repo/MyFlowHub-Win/frontend/src/stores/flow.ts`
  - `repo/MyFlowHub-Win/frontend/src/pages/Flow.vue`
- 新增 UI / 状态：
  - Event Mode 下拉：`publish` / `received` / `any`
  - 保存与加载时透传 `trigger.event_mode`。

## 5) 文档更新

- 文件：`repo/MyFlowHub-Server/docs/specs/flow.md`
- 更新点：
  - `event` 触发来源改为 `topicbus.publish` / `topicbus.received`
  - `set.trigger` 增加 `event_mode` 定义与默认值说明。

# 对应 plan 任务映射

- Task A：扩展 `flow` 触发器模型（`event_mode`）
  - `repo/MyFlowHub-Proto/protocol/flow/types.go`
  - `repo/MyFlowHub-SubProto/flow/handler.go`
- Task B：实现 TopicBus 接收事件源
  - `repo/MyFlowHub-SubProto/topicbus/topicbus.go`
- Task C：打通 Win 编辑器配置
  - `repo/MyFlowHub-Win/frontend/src/stores/flow.ts`
  - `repo/MyFlowHub-Win/frontend/src/pages/Flow.vue`
- Task D：测试与文档
  - `repo/MyFlowHub-SubProto/flow/trigger_test.go`
  - `repo/MyFlowHub-SubProto/topicbus/trigger_event_test.go`
  - `repo/MyFlowHub-Server/docs/specs/flow.md`

# 关键设计决策与权衡

- 采用 `event_mode` 而非新增 `trigger.type`，避免触发器类型膨胀，保留统一事件过滤模型。
- `received` 仅在“父连接下发 publish”时发出，减少对源发布节点的重复触发影响。
- `any` 用于兼容混合场景（同一 flow 同时接受 publish/received）。

# 测试与验证方式 / 结果

- `repo/MyFlowHub-SubProto/flow`：`go test ./...`（通过）
- `repo/MyFlowHub-SubProto/topicbus`：`go test ./...`（通过）
- `repo/MyFlowHub-Proto`：`go test ./...`（通过）
- `repo/MyFlowHub-Win/frontend`：`npm run build`（通过）

# 潜在影响与回滚方案

- 潜在影响：
  - 新增 `event_mode` 后，若外部手工构造非法值会被 `flow.set` 拒绝（`400`）。
  - `received` 依赖父子连接方向，非树形拓扑下需额外定义接收语义。
- 回滚方案：
  - 回退 `flow` trigger 的 `event_mode` 解析与校验；
  - 回退 `topicbus.received` 事件发出逻辑；
  - 回退 Win 编辑器 Event Mode UI 字段。

# 实施例外说明

- 本轮在现有多仓主工作区直接实施并提交（未新建独占 worktree）。
- 已通过对应仓库提交 + 全局 `docs/change/` 归档方式完成收敛，便于后续交接与追踪。

