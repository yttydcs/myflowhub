# 2026-03-21_proto-flow-delete

## 变更背景 / 目标
为 Flow 协议补齐删除部署能力，提供 `delete/delete_resp` 及 `flow.delete` 权限常量，作为 SubProto/Win 删除部署链路的协议基础。

## 具体变更内容（新增 / 修改 / 删除）
- 修改 `protocol/flow/types.go`：
  - 新增动作常量：`ActionDelete`、`ActionDeleteResp`。
  - 新增权限常量：`PermFlowDelete`。
  - 新增结构体：`DeleteReq`、`DeleteResp`。
- 修改 `docs/protocol_map.md`：
  - Flow 映射中新增 `delete/delete_resp`、`DeleteReq/DeleteResp`、`PermFlowDelete`。

## 对应 plan.md 任务映射
- `PROTO-DEL-1`：完成（新增 action/权限常量）。
- `PROTO-DEL-2`：完成（新增 DeleteReq/DeleteResp）。
- `PROTO-DEL-3`：完成（同步 protocol_map）。
- `PROTO-DEL-4`：完成（测试通过）。

## 关键设计决策与权衡（性能 / 扩展性）
- 仅做协议字典扩展，不引入运行时逻辑，保持向后兼容与低风险发布。
- 保持命名与现有 set/run/status/list/get 风格一致，降低下游接入成本。

## 测试与验证方式 / 结果
- 执行：`GOWORK=off go test ./...`
- 结果：通过（`internal/protocolmap` 通过，其余协议包无测试文件）。

## 潜在影响与回滚方案
- 潜在影响：下游若未同步新 action 常量，delete 请求不会被识别。
- 回滚方案：回退 `protocol/flow/types.go` 与 `docs/protocol_map.md` 本次增量。 
