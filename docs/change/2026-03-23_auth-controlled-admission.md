# 2026-03-23 Auth 受控准入与一次性角色 Permit

## 变更背景 / 目标
- 将当前开放式 `auth register` 改造成可配置的受控准入：
  - 普通注册可先进入 pending，待审批后再重试 `register`
  - 已授权节点可签发一次性、角色绑定、绑定 `device_id` 的 permit
- 保持已有网络兼容：
  - 默认仍是开放注册
  - 既有已注册节点的 `register` 继续作为幂等 rebind
  - `hubruntime` / `bootstrap` 可通过 `parent.join_permit` 自动上联

## 具体变更内容
### Proto
- `MyFlowHub-Proto/protocol/auth/types.go`
  - 为 `register/login/resp` 补齐 `display_name`
  - `register` 新增 `requested_role`、`join_permit`
  - `register_resp` 新增 `status`、`request_id`、`reason`
  - 新增审批 / permit 相关 action 常量与 payload struct
- `MyFlowHub-Proto/docs/protocol_map.md`
  - 同步生成后的协议映射

### Core
- `MyFlowHub-Core/config/config.go`
  - 新增：
    - `auth.register.require_approval`
    - `auth.register.pending_ttl_sec`
    - `auth.register.permit_ttl_sec`
    - `parent.join_permit`
- `MyFlowHub-Core/kit/permission/permission.go`
  - 新增：
    - `auth.pending.list`
    - `auth.register.approve`
    - `auth.register.reject`
    - `auth.permit.issue`
    - `auth.permit.revoke`
- `MyFlowHub-Core/bootstrap/selfregister.go`
  - `SelfRegisterOptions` 支持 `JoinPermit`
  - `parseRegisterResp` 能识别 `approved/pending/rejected`

### SubProto/auth
- `MyFlowHub-SubProto/auth`
  - 新增 pending / approved / permit 状态机
  - 普通 register 在审批模式下返回 `status=pending`
  - `approve_register` 只预留身份，不直接完成 whitelist/binding
  - permit 一次性消费，绑定 `device_id + role + expiry`
  - 新增 list / approve / reject / issue / revoke action，并受 `auth.*` 权限保护
  - `trusted_nodes.json.meta` 开始实际持久化：
    - `pending_registers`
    - `approved_registers`
    - `register_permits`

### Server
- `MyFlowHub-Server/hubruntime`
  - 新增 `ParentJoinPermit` 选项与 `HUB_PARENT_JOIN_PERMIT`
  - pre-start `SelfRegister` 与持久 parent 连接上的 register rebind 都会透传 permit
- `MyFlowHub-Server/docs/specs/auth.md`
  - 更新长期 auth 规范，写清 pending / permit / approval / runtime 语义

## 对应计划任务映射
- `PROTO1` / `PROTO2`
- `CORE1` / `CORE2` / `CORE3`
- `AUTH1` / `AUTH2` / `AUTH3` / `AUTH4`
- `SRV1` / `SRV2` / `DOC1`

## 关键设计决策与权衡
- 仍以 `register` 作为统一入口，不新增平行协议，减少 client/runtime 变更面。
- ordinary approval 采用“先 pending，approve 后重试 register”的流程，避免在未批准前发放正式 `node_id` 给申请方。
- permit 采用 authority 保存的一次性 opaque token，而不是自校验签名 blob，优先满足一次性消费、撤销与最小改动面。
- 对已存在 whitelist 的 `device_id`，register 继续按幂等 rebind 处理；这使 parent bootstrap 可以先靠 permit 完成首次准入，再在持久 parent 连接上无感 rebind。

## 测试与验证
- Proto：
  - `go test github.com/yttydcs/myflowhub-proto/... -count=1 -p 1`
- Core：
  - `go test github.com/yttydcs/myflowhub-core/... -count=1 -p 1`
- SubProto/auth：
  - `go test github.com/yttydcs/myflowhub-subproto/auth/... -count=1 -p 1`
- Server/hubruntime：
  - `go test github.com/yttydcs/myflowhub-server/hubruntime/... -count=1 -p 1`
- 本轮联调使用 workflow-local `go.work`：
  - `d:\project\MyFlowHub3\worktrees\feat-auth-admission-control\go.work`

## 潜在影响
- 开启 `auth.register.require_approval=true` 后，普通新节点首次注册会收到 `status=pending`，需要外部审批流配合。
- `approve_register` 只是“批准并预留身份”，申请方仍需重试 `register` 才会真正入网。
- 如果 parent 网络开启审批但未配置 `parent.join_permit`，`hubruntime` 启动前 bootstrap 会显式失败。

## 回滚方案
- 回退以下改动即可恢复开放注册：
  - Proto：`protocol/auth/types.go` 与 `docs/protocol_map.md`
  - Core：`config/config.go`、`kit/permission/permission.go`、`bootstrap/selfregister.go`
  - SubProto/auth：admission state、action、持久化与测试文件
  - Server：`hubruntime/*` 与 `docs/specs/auth.md`
