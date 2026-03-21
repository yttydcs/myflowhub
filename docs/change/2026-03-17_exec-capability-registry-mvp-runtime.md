# 变更背景 / 目标

- 背景：已完成 `exec` 能力注册中心协议草案，本轮继续落地最小可用运行时。
- 目标：实现 `cap_snapshot/cap_query` 可用链路，并补齐 `upsert/withdraw/heartbeat` 基础处理与测试。

# 具体变更内容

## SubProto exec 运行时

- 扩展 `exec` action 注册：
  - `cap_snapshot`
  - `cap_upsert`
  - `cap_withdraw`
  - `cap_heartbeat`
  - `cap_sync_resp`
  - `cap_query`
  - `cap_query_resp`
  - 文件：`repo/MyFlowHub-SubProto/exec/actions.go`

- `exec` handler 新增能力注册中心最小状态与处理逻辑：
  - 本地能力索引：`capLocal`（由 `RegisterMethod` 自动登记）
  - 子节点快照：`capChildren`（按 `from_node` + `epoch` + `lease` 管理）
  - 请求处理：
    - `handleCapSnapshot`
    - `handleCapUpsert`
    - `handleCapWithdraw`
    - `handleCapHeartbeat`
    - `handleCapQuery`
  - 响应路径：
    - `sendCapSyncRespByHeader`
    - `sendCapQueryRespByHeader`
  - 查询聚合：
    - 本地能力 + 子节点能力合并
    - 支持 `method` 精确/前缀过滤、`provider_node` 过滤、`limit` 限制
  - 文件：`repo/MyFlowHub-SubProto/exec/handler.go`

## Broker 支持

- 新增 `cap_query_resp` 进程内投递器：
  - `SharedExecCapQueryBroker()`
  - 文件：`repo/MyFlowHub-SubProto/broker/exec_cap_query.go`

## 工作区配置

- 为联调新 broker 能力，`go.work` 增加：
  - `./repo/MyFlowHub-SubProto/broker`

# 对应任务映射

- Task A：能力注册中心最小实现
  - `repo/MyFlowHub-SubProto/exec/handler.go`
  - `repo/MyFlowHub-SubProto/exec/actions.go`
- Task B：查询响应投递
  - `repo/MyFlowHub-SubProto/broker/exec_cap_query.go`
- Task C：测试覆盖
  - `repo/MyFlowHub-SubProto/exec/cap_registry_test.go`

# 关键设计决策与权衡

- 保持 `exec.call` 语义不变，新增能力面走独立 action，降低回归风险。
- `snapshot` 采用 `epoch` 防止旧数据覆盖新会话状态。
- `lease` 到期按查询时清理，避免额外后台任务复杂度。
- `query` 默认限流（`maxCapQueryLimit`）避免一次返回过大。
- 本版对同步来源做“连接 nodeID 与 from_node 一致性”校验，降低伪造风险。

# 测试与验证方式 / 结果

- `repo/MyFlowHub-SubProto/exec`：`go test ./...`（通过）
- `repo/MyFlowHub-SubProto/broker`：`go test ./...`（通过）
- `repo/MyFlowHub-Proto`：`go test ./...`（通过）
- `repo/MyFlowHub-Server`：`go test ./...`（通过）

新增测试：
- `TestCapSnapshotAndQuery`
- `TestCapSnapshotRejectsSourceMismatch`
- `TestCapSnapshotRejectsStaleEpoch`
- 文件：`repo/MyFlowHub-SubProto/exec/cap_registry_test.go`

# 潜在影响与回滚方案

- 潜在影响：
  - 新增 exec actions 会进入 handler 分发；旧客户端不受影响。
  - 当前实现为“本节点聚合 + 一跳子节点快照”MVP，尚未实现跨多级自动上行聚合。
- 回滚方案：
  - 回滚 `exec/actions.go`、`exec/handler.go`、`broker/exec_cap_query.go`、测试文件与 `go.work` 相关行。
  - 回滚后运行 `go test ./...` 验证恢复。

