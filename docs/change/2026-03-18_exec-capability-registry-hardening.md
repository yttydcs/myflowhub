# 变更背景 / 目标

- 背景：`exec` 能力注册中心已具备逐级注册、上行聚合与增量同步，但缺少失败重同步、连接关闭清理和权限硬约束。
- 目标：补齐运行时健壮性，确保上行状态可自愈、子连接断开后索引及时收敛、`cap` 面向权限可控。

# 具体变更内容（新增 / 修改 / 删除）

## 1) `cap_sync_resp` 处理与重同步

- 文件：`repo/MyFlowHub-SubProto/exec/handler.go`
- 新增 `handleCapSyncResp` 实现：
  - 识别上行同步请求（`capsnapshot/capupsert/capwithdraw/capheartbeat`）。
  - `code=1`：刷新上行租约时间并对齐 `epoch`。
  - `code in {404,409}` 或 `>=500`：清空上行缓存状态，触发一次全量 `cap_snapshot` 重同步。

## 2) 连接关闭事件清理子树能力

- 文件：`repo/MyFlowHub-SubProto/exec/handler.go`
- 新增 `conn.closed` 订阅：
  - 在 `OnReceive` 首次拿到 `server` 后注册事件监听。
  - 连接关闭时按 `node_id` 删除 `capChildren[node]`。
  - 删除后触发上行增量同步（通常为 `cap_withdraw`）。

## 3) 能力同步 / 查询权限校验

- 文件：`repo/MyFlowHub-SubProto/exec/handler.go`
- `cap_snapshot/cap_upsert/cap_withdraw/cap_heartbeat` 增加 `exec.cap.sync` 校验。
- `cap_query` 增加 `exec.cap.query` 校验。
- 拒绝时返回标准响应：`code=403, msg=permission denied`。

## 4) 回归测试补充

- 文件：
  - `repo/MyFlowHub-SubProto/exec/cap_registry_test.go`
  - `repo/MyFlowHub-SubProto/exec/cap_upstream_test.go`
  - `repo/MyFlowHub-SubProto/exec/resp_ids_test.go`
- 新增覆盖：
  - `cap_snapshot` 权限拒绝。
  - `cap_query` 权限拒绝。
  - `cap_sync_resp(409)` 触发全量重同步。
  - `conn.closed` 清理子能力并触发上行撤销。

## 5) 规范文档更新

- 文件：`repo/MyFlowHub-Server/docs/7-exec.md`
- 更新点：
  - 明确 `exec.cap.sync` / `exec.cap.query` 为当前生效权限。
  - 明确上行策略为“首次全量 + 稳态增量 + 心跳续租”。
  - 明确 `cap_sync_resp` 的失败重同步语义。

# 对应 plan 任务映射

- Task A：补齐 `cap_sync_resp` 失败恢复
  - `repo/MyFlowHub-SubProto/exec/handler.go`
- Task B：补齐连接关闭清理与上行收敛
  - `repo/MyFlowHub-SubProto/exec/handler.go`
- Task C：补齐 `cap` 权限校验
  - `repo/MyFlowHub-SubProto/exec/handler.go`
- Task D：补齐回归测试
  - `repo/MyFlowHub-SubProto/exec/cap_registry_test.go`
  - `repo/MyFlowHub-SubProto/exec/cap_upstream_test.go`
  - `repo/MyFlowHub-SubProto/exec/resp_ids_test.go`
- Task E：同步规范文档
  - `repo/MyFlowHub-Server/docs/7-exec.md`

# 关键设计决策与权衡

- 对 `cap_sync_resp` 仅在可恢复错误（404/409/5xx）触发自动重同步，避免对业务错误（如 400/403）形成重试风暴。
- 连接关闭仅清理该 child 快照并走现有增量 diff，避免全量重建的额外网络与 CPU 开销。
- 权限检查放在请求处理入口，尽早失败，降低后续状态机与锁开销。

# 测试与验证方式 / 结果

- 命令：`go test ./...`
- 目录：`repo/MyFlowHub-SubProto/exec`
- 结果：通过

# 潜在影响与回滚方案

- 潜在影响：
  - 默认权限若不包含 `exec.cap.sync/query`，新节点可能出现 403（属于预期安全收敛）。
  - 上行失败时会触发一次全量重同步，短时会增加一次上行流量。
- 回滚方案：
  - 回退 `repo/MyFlowHub-SubProto/exec/handler.go` 中新增权限校验与重同步逻辑。
  - 回退对应新增测试与文档变更。

# 实施例外说明

- 本轮在现有多仓主工作区直接实施并提交（未新建独占 worktree）。
- 已通过将变更提交落在对应仓库 `main`，并将归档文档保存在全局 `docs/change/`，保证可审计与可交接。
