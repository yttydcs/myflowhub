# 变更背景 / 目标

- 背景：上一轮已落地 `exec` 能力注册中心 MVP（本地+子节点快照存储与本地查询）。
- 目标：继续补齐“逐级可用”关键能力：
  1) 子树聚合结果自动上行同步到父节点；
  2) 本地 `cap_query` 无命中时自动上行查询回退。

# 具体变更内容

## exec handler：上行自动同步

- 新增上行同步状态：
  - `capUpstreamParent`
  - `capUpstreamEpoch`
  - `capUpstreamReqSeq`
  - `capUpstreamSent`
- 新增逻辑：
  - `maybeSyncSnapshotUpstream`：检测父连接并上送 `cap_snapshot`
  - `buildSnapshotReqForUpstream`：构造上送请求（带递增 `epoch`）
  - `snapshotAggregatedCapsLocked`：聚合本地 + 子节点能力快照
- 触发时机：
  - `cap_snapshot/upsert/withdraw/heartbeat` 成功后强制上送
  - `OnReceive` 入口按需做一次非强制补发（首次/父切换后）
- 文件：`repo/MyFlowHub-SubProto/exec/handler.go`

## exec handler：query 上行回退

- 新增：
  - `queryCapabilityUpstream`：本地查询无命中时向父节点发 `cap_query` 并等待 `cap_query_resp`
  - 等待机制复用 broker，带超时控制（3s）
- 行为：
  - 本地命中优先返回；
  - 本地未命中且存在父节点时，上行查询并回包给原请求方。
- 文件：`repo/MyFlowHub-SubProto/exec/handler.go`

## Broker

- 复用上一轮新增的 `SharedExecCapQueryBroker` 进行请求-响应配对。
- 文件：`repo/MyFlowHub-SubProto/broker/exec_cap_query.go`

# 对应任务映射

- Task A：实现上行自动 snapshot 同步
  - `repo/MyFlowHub-SubProto/exec/handler.go`
- Task B：实现 `cap_query` 上行回退
  - `repo/MyFlowHub-SubProto/exec/handler.go`
- Task C：测试覆盖
  - `repo/MyFlowHub-SubProto/exec/cap_upstream_test.go`

# 关键设计决策与权衡

- 上行同步采用“聚合快照上送”，实现简单、状态一致性更直观；代价是上送包体较增量更大。
- 通过 `epoch` 抵御乱序/陈旧覆盖；父切换时自动触发重新上送。
- `cap_query` 回退采用“按请求阻塞等待 broker 投递”实现，避免引入额外异步状态机；代价是单次请求会等待上行超时窗口。

# 测试与验证方式 / 结果

- 新增测试：
  - `TestCapSnapshotSyncsUpstreamToParent`
  - `TestCapQueryFallsBackToParent`
  - 文件：`repo/MyFlowHub-SubProto/exec/cap_upstream_test.go`
- 回归测试：
  - `repo/MyFlowHub-SubProto/exec`：`go test ./...`（通过）
  - `repo/MyFlowHub-SubProto/broker`：`go test ./...`（通过）
  - `repo/MyFlowHub-Server`：`go test ./...`（通过）

# 潜在影响与回滚方案

- 潜在影响：
  - 在高频同步场景，上行 `cap_snapshot` 发送频率会提升。
  - 上行查询回退为同步等待路径，极端网络抖动会增加请求耗时。
- 回滚方案：
  - 回滚 `exec/handler.go` 的上行同步与上行查询回退逻辑；
  - 保留本地查询能力（MVP 版本）；
  - 回滚后执行 `go test ./...` 复验。

