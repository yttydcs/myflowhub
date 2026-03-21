# 变更背景 / 目标

- 背景：上一轮已实现上行 `cap_snapshot` 同步与 `cap_query` 回退，但上行同步仍偏全量。
- 目标：将上行同步优化为“增量优先”，减少父链通信开销，同时保留首包全量快照保障一致性。

# 具体变更内容

## 上行同步策略优化（exec）

- `maybeSyncSnapshotUpstream` 改为：
  - 首次/父节点切换：发送 `cap_snapshot`（全量）
  - 稳态变更：优先发送 `cap_upsert` / `cap_withdraw`（增量）
  - 无差异场景：按需发送 `cap_heartbeat` 续租
- 新增上行状态：
  - `capUpstreamLastAt`
  - `capUpstreamCache`
- 新增核心辅助逻辑：
  - `collectAggregatedCapsLocked`
  - `diffCapabilityMaps`
  - `capabilityDescriptorEqual`
  - `capabilitiesFromMap`
  - `capKeyToWire`
- 文件：`repo/MyFlowHub-SubProto/exec/handler.go`

## 兼容性

- 不改变 `exec.call` 语义。
- 不改变 `cap_query` 回退语义，仅优化同步通道发送策略。

# 对应任务映射

- Task A：增量同步策略实现
  - `repo/MyFlowHub-SubProto/exec/handler.go`
- Task B：回归验证
  - `repo/MyFlowHub-SubProto/exec/cap_upstream_test.go`

# 关键设计决策与权衡

- 用“本地缓存快照 diff”换取增量网络开销，代码复杂度可控。
- 仍保留首包全量 `snapshot`，保证父节点在会话初始化时状态完整。
- `heartbeat` 作为续租兜底，避免无变化时父节点租约过期。

# 测试与验证方式 / 结果

- 新增测试：
  - `TestCapUpsertSyncsIncrementallyToParent`
  - 文件：`repo/MyFlowHub-SubProto/exec/cap_upstream_test.go`
- 回归：
  - `repo/MyFlowHub-SubProto/exec`：`go test ./...`（通过）
  - `repo/MyFlowHub-SubProto/broker`：`go test ./...`（通过）
  - `repo/MyFlowHub-Server`：`go test ./...`（通过）

# 潜在影响与回滚方案

- 潜在影响：
  - 增量对比依赖 descriptor 比较逻辑，若未来字段扩展需同步更新比较函数。
  - 在极端高频变更下，仍可能出现较高同步频率（但相较全量已下降）。
- 回滚方案：
  - 回滚 `exec/handler.go` 增量 diff 相关改动，恢复全量 `cap_snapshot` 上送策略。
  - 保留 `cap_query` 回退能力不变。

