# Authority Administration Must Follow The Node Tree

## Summary
vNext 中父子链路同时是路由边和 authority 边。具备对应权限的节点可以沿 remote authority 链路发起管理操作，但转发必须保留原始 `Principal`、deadline、拓扑代次和 policy generation；不能把“请求经过父链路”简化为“任意中继拥有管理员权限”。

## Lookup Hints

- `requires authority-local session`
- `auth list_pending_registers: request timed out`
- `auth list_register_permits: request timed out`
- `routed source`、`source mismatch`
- `Principal`、`TopologyEpoch`、policy generation

## Symptoms
- `Registration Approvals` 或 `Permit Issuance` 在 remote authority 场景仍报：
  - `auth list_pending_registers: request timed out`
  - `auth list_register_permits: request timed out`
  - `requires authority-local session`
- authority 本机可操作，但其它已登录管理节点仍被 UI 或服务层阻断
- 请求能到中间节点，但 authority 侧日志出现 routed source / source mismatch 类拒绝

## Impact

- 合法管理员无法跨节点管理准入与 permit，或中继错误提升权限。
- reparent 后若旧代次仍有效，可能把管理命令发送到错误 authority。

## Trigger Conditions
- `sourceId != authorityId`
- 消费方仍依赖旧版 `myflowhub-subproto/auth`
- Win orchestration / 页面层残留 authority-local guard
- authority 尚未建立 `SourceID -> 当前入站连接` 的 descendant route index

## Root Cause

- UI、服务层或旧消费者残留 authority-local guard。
- 转发层改写了原始授权主体，或 authority 使用了过期 route/policy generation。
- 旧多仓依赖版本不一致，使 UI、runtime 和权限规范处于不同基线。

## Investigation Trail
1. 对比当前 session `nodeId` 和解析出的 `authorityId`。
2. 查看 [节点树、链路与资源架构](../specs/node-tree-link-resource-architecture.md) 中的 authority 和路由约束。
3. 查看调用方是否还存在 authority-local guard：
   - 关键字：`requires authority-local session`
   - 关键字：`authority-local`
4. 查看 authority 侧该 `SourceID` 的路由索引是否落在当前入站连接上：
   - 关键词：`GetByNode(sourceID)`
   - 关键词：`source mismatch`
   - 关键词：`routed source`

## Resolution
- 清理调用层的 authority-local 快速失败逻辑，统一走当前 monorepo 的 Command 与 policy 校验。
- 确认请求保持真实 `Principal` 和目标 authority，不引入平台专用旁路 action。
- 若 authority 仍拒绝请求，优先修复 descendant route index，而不是绕过 source 校验。

## Prevention / Guardrails
- 不要再把 authority-local guard 当成长期方案；它只适用于历史版本排障。
- 行为升级后，必须同步更新 requirements/specs/lessons，避免 UI、runtime 和文档再次背离。
- 远程 authority 管理失败时，先查版本与 route ownership，再查页面状态；不要直接回退为“只能 authority 本机操作”。

## Related Docs

- [节点树、链路与资源架构](../specs/node-tree-link-resource-architecture.md)
- [运行生命周期](../specs/operational-lifecycle.md)
- [Hub](../features/hub.md)
- [历史 remote authority guard 变更](../change/2026-03-28_win-permit-remote-authority-guard.md)
