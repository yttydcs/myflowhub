# 变更背景 / 目标

- 背景：计划扩充节点能力，要求能力注册中心先挂载在 `exec` 子协议下，并支持逐级注册/逐级聚合。
- 目标：先完成协议与文档草案（不改现有 `exec.call` 运行时行为），为后续实现打基础。

# 具体变更内容（新增/修改）

## 协议层（Proto）

- 在 `exec` 协议中新增能力注册中心动作常量：
  - `cap_snapshot`
  - `cap_upsert`
  - `cap_withdraw`
  - `cap_heartbeat`
  - `cap_sync_resp`
  - `cap_query`
  - `cap_query_resp`
- 新增权限常量：
  - `exec.cap.sync`
  - `exec.cap.query`
- 新增类型草案：
  - `CapabilityDescriptor`
  - `CapabilityKey`
  - `CapSnapshotReq`
  - `CapUpsertReq`
  - `CapWithdrawReq`
  - `CapHeartbeatReq`
  - `CapSyncResp`
  - `CapQueryReq`
  - `CapabilityRoute`
  - `CapQueryResp`
- 文件：`repo/MyFlowHub-Proto/protocol/exec/types.go`

## 兼容壳同步

- SubProto 兼容别名同步：`repo/MyFlowHub-SubProto/exec/types.go`
- Server 兼容别名同步：`repo/MyFlowHub-Server/protocol/exec/types.go`

## 文档更新

- `exec` 协议文档新增“能力注册中心（挂载在 exec，逐级注册草案）”章节：
  - 核心模型：`local_caps` / `child_caps` / `subtree_index`
  - 同步动作定义与字段
  - 查询动作定义与字段
  - 与 `call` 的关系（保持兼容）
- 文件：`repo/MyFlowHub-Server/docs/7-exec.md`

## 协议映射文档再生成

- 由于新增 `exec` action 常量，重新生成协议映射文档：
  - `repo/MyFlowHub-Proto/docs/protocol_map.md`

# 对应计划任务映射

- Task A：扩展 Proto `exec` 草案类型  
  - `repo/MyFlowHub-Proto/protocol/exec/types.go`
- Task B：同步兼容层  
  - `repo/MyFlowHub-SubProto/exec/types.go`
  - `repo/MyFlowHub-Server/protocol/exec/types.go`
- Task C：更新文档草案  
  - `repo/MyFlowHub-Server/docs/7-exec.md`
- Task D：验证与收尾  
  - `repo/MyFlowHub-Proto/docs/protocol_map.md`（生成）

# 关键设计决策与权衡

- 不新增 subproto，先挂在 `exec` 内，降低接入与迁移成本。
- 保持 `call/call_resp` 原语义不变，避免影响现有 Flow/Exec 路径。
- 先定义 wire contract，再做运行时实现，减少“边做边改协议”风险。
- 采用逐级注册模型（而非 root 单点注册），满足断父后子树可用性目标。

# 测试与验证方式 / 结果

- `repo/MyFlowHub-Proto`：`go test ./...`（通过）
- `repo/MyFlowHub-SubProto/exec`：`go test ./...`（通过）
- `repo/MyFlowHub-Server`：`go test ./...`（通过）
- `repo/MyFlowHub-Proto`：`go run ./cmd/protocolmapgen -write -out docs/protocol_map.md`（执行成功）

# 潜在影响与回滚方案

- 潜在影响：
  - 当前为协议草案扩展，运行时尚未启用新 action，行为兼容风险低。
  - 下游如果严格校验 `exec` action 白名单，需要同步放开新 action。
- 回滚方案：
  - 回滚本次涉及的四个文件（协议、兼容壳、文档、protocol_map）。
  - 重新运行 `go test ./...` 验证恢复。

