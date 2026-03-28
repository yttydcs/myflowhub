# 2026-03-28 Auth 活动 permit 列表与 Win 准入许可页收敛

## 变更背景 / 目标
- authority 管理端的“准入许可”页此前仍以“最近一次签发结果”作为主要视图，无法反映 authority 当前真实 permit 状态。
- 本轮目标是把 permit 管理明确收敛到 auth 协议和 auth runtime：
  - 补齐跨仓 `list_register_permits` 契约
  - 让 auth runtime 返回当前活动 permit 列表
  - 让 Win 页面改为 `Refresh + New Permit + 活动列表 + 行内 Revoke`

## 具体变更内容
### MyFlowHub-Proto
- 更新 [protocol/auth/types.go](../../repo/MyFlowHub-Proto/protocol/auth/types.go)
  - 新增 `ActionListRegisterPermits`
  - 新增 `ActionListRegisterPermitsResp`
  - 新增 `RegisterPermitInfo`
  - 新增 `ListRegisterPermitsReq`
  - 新增 `ListRegisterPermitsResp`
- 更新 [docs/protocol_map.md](../../repo/MyFlowHub-Proto/docs/protocol_map.md)
  - 重新生成 protocol map，纳入 permit list request / response / item
- 分仓归档：
  - [2026-03-28_proto-auth-permit-list-wire.md](../../repo/MyFlowHub-Proto/docs/change/2026-03-28_proto-auth-permit-list-wire.md)

### MyFlowHub-Server
- 更新 [protocol/auth/types.go](../../repo/MyFlowHub-Server/protocol/auth/types.go)
  - 补齐 permit list action 常量和 req / resp / item alias 导出
  - 顺手补齐 admission 相关缺失 alias，保持兼容壳完整
- 更新 [docs/specs/auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)
  - 新增 `list_register_permits` 契约说明
  - 明确列表只返回当前活动 permit
  - 明确列表权限沿用 `auth.permit.issue` 或 `auth.permit.revoke`
- 更新 [docs/specs/protocol_map.md](../../repo/MyFlowHub-Server/docs/specs/protocol_map.md)
  - 同步 Proto canonical protocol map 副本
- 分仓归档：
  - [2026-03-28_server-auth-permit-list-contract.md](../../repo/MyFlowHub-Server/docs/change/2026-03-28_server-auth-permit-list-contract.md)

### MyFlowHub-SubProto/auth
- 更新 [auth/types.go](../../repo/MyFlowHub-SubProto/auth/types.go)
  - 接入 permit list 相关 Proto alias
- 更新 [auth/actions_admission.go](../../repo/MyFlowHub-SubProto/auth/actions_admission.go)
  - 注册 `list_register_permits`
  - 增加 `handleListRegisterPermits`
  - 增加 `requireAnyActionPermission(...)`
- 更新 [auth/admission.go](../../repo/MyFlowHub-SubProto/auth/admission.go)
  - 新增 `listRegisterPermits(req)`
  - list 前执行 `cleanupExpiredAdmissionLocked()`
  - 支持 `device_id` 过滤、排序和分页
- 更新 [auth/perm_helpers.go](../../repo/MyFlowHub-SubProto/auth/perm_helpers.go)
  - 对仅有 `role` 但缺失 `perms` 的旧 binding 记录按当前角色配置回填权限
  - 避免 permit list 权限判断误把旧状态视为无权限
- 更新 [auth/admission_test.go](../../repo/MyFlowHub-SubProto/auth/admission_test.go)
  - 覆盖 permission denied
  - 覆盖仅有 `auth.permit.revoke` 也可 list
  - 覆盖 revoke / consume / expiry 后列表变化
- 分仓归档：
  - [2026-03-28_subproto-auth-permit-list-runtime.md](../../repo/MyFlowHub-SubProto/docs/change/2026-03-28_subproto-auth-permit-list-runtime.md)

### MyFlowHub-Win
- 更新 [internal/services/auth/authority.go](../../repo/MyFlowHub-Win/internal/services/auth/authority.go)
  - 新增 permit list typed req / resp / item
  - 新增 `ListRegisterPermits` / `ListRegisterPermitsSimple`
- 更新 [internal/services/permission/service.go](../../repo/MyFlowHub-Win/internal/services/permission/service.go)
  - 新增 `ListRegisterPermits(...)`
  - 新增面向前端的 request / result struct 和转换逻辑
- 更新 [frontend/src/stores/permitIssuance.ts](../../repo/MyFlowHub-Win/frontend/src/stores/permitIssuance.ts)
  - 从 `lastIssued / lastRevoke` 切换到列表状态
  - 新增 `loading`、`busyPermit`、`total`、`items`
  - 新增 `loadPermits()`，issue / revoke 成功后统一刷新真实列表
- 更新 [frontend/src/pages/PermitIssuance.vue](../../repo/MyFlowHub-Win/frontend/src/pages/PermitIssuance.vue)
  - 顶部动作收敛为 `Refresh` 和 `New Permit`
  - 主体改为活动 permit 列表、空态和 loading 态
  - 列表行内直接 `Revoke`
  - 删除 latest-only 结果卡和手工 revoke dialog
- 更新 [frontend/src/pages/PermitIssuance.test.ts](../../repo/MyFlowHub-Win/frontend/src/pages/PermitIssuance.test.ts)
  - 改为列表流测试
- 更新 [frontend/src/stores/authority_admin.test.ts](../../repo/MyFlowHub-Win/frontend/src/stores/authority_admin.test.ts)
  - 补 list -> issue -> list -> revoke -> list 回归
- 更新 [docs/requirements/authority-admin-console.md](../../repo/MyFlowHub-Win/docs/requirements/authority-admin-console.md)
  - 明确 Win authority console 需要暴露 `ListRegisterPermits`
- 更新 [docs/specs/authority-admin-console.md](../../repo/MyFlowHub-Win/docs/specs/authority-admin-console.md)
  - 明确 permit issuance 页改为活动 permit 列表视图
- 分仓归档：
  - [2026-03-28_win-active-permit-list.md](../../repo/MyFlowHub-Win/docs/change/2026-03-28_win-active-permit-list.md)

## Requirements impact
- `updated`

## Specs impact
- `updated`

## Lessons impact
- `none`

## Related requirements
- [authority-admin-console.md](../../repo/MyFlowHub-Win/docs/requirements/authority-admin-console.md)

## Related specs
- [protocol_map.md](../../repo/MyFlowHub-Proto/docs/protocol_map.md)
- [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)
- [protocol_map.md](../../repo/MyFlowHub-Server/docs/specs/protocol_map.md)
- [authority-admin-console.md](../../repo/MyFlowHub-Win/docs/specs/authority-admin-console.md)

## Related lessons
- `none`

## 对应 plan.md 任务映射
- `PROTO-PERMIT-1`
- `SERVER-PERMIT-1`
- `SUBPROTO-PERMIT-1`
- `WIN-PERMIT-1`
- `WIN-PERMIT-2`
- `VALIDATE-PERMIT-1`
- `REVIEW-PERMIT-1`
- `ARCHIVE-PERMIT-1`
- `MERGE-PERMIT-1`

## 经验 / 教训摘要
- 所谓“真实后端 permit 列表”就是 auth 协议和 auth runtime，而不是前端自己维护一份 latest 快照。
- permit list 不需要新拆一个 `auth.permit.list` 权限；沿用 `auth.permit.issue` 或 `auth.permit.revoke` 即可满足当前模型。
- 旧 whitelist / binding 记录不能假设同时带有 `role` 和 `perms`；运行时按角色配置回填 `perms` 才能避免伪回归。

## 可复用排查线索
- 症状：
  - 准入许可页仍只显示最近一次签发结果
  - `list_register_permits` 返回 `4403 permission denied`，但节点已有 `auth.permit.revoke`
  - revoke / consume / expiry 后 permit 仍留在列表里
- 触发条件：
  - Win 侧还在使用 latest-only store
  - SubProto list 前没有做过期清理
  - 旧 binding 记录只有 `role` 没有 `perms`
- 关键词：
  - `list_register_permits`
  - `permitStore.loadPermits`
  - `requireAnyActionPermission`
  - `cleanupExpiredAdmissionLocked`
  - `lookupByNode`
- 快速检查：
  - 看 [protocol/auth/types.go](../../repo/MyFlowHub-Proto/protocol/auth/types.go) 是否已定义 permit list action / struct
  - 看 [docs/specs/auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md) 是否已记录列表权限与“活动 permit”语义
  - 看 [auth/perm_helpers.go](../../repo/MyFlowHub-SubProto/auth/perm_helpers.go) 是否会对缺失 `perms` 的角色记录做回填
  - 看 [frontend/src/pages/PermitIssuance.vue](../../repo/MyFlowHub-Win/frontend/src/pages/PermitIssuance.vue) 是否在 issue / revoke 后回到真实列表加载链路

## 关键设计决策与权衡
- 采用“Proto canonical wire -> Server stable spec -> SubProto runtime -> Win UI”顺序收敛，而不是单独做前端假列表
  - 优点：语义统一，后续其它客户端也能直接复用
  - 代价：需要跨 4 个 repo 联动
- 列表只展示当前活动 permit，不补历史查询
  - 优点：界面和状态更简单，和当前 auth runtime 能力匹配
  - 代价：如果后续需要审计历史，需要单独新增能力
- 列表权限复用现有 `issue / revoke`
  - 优点：改动面最小，避免无谓扩张权限模型
  - 代价：列表权限无法再单独细分

## 测试与验证方式 / 结果
- `MyFlowHub-Proto`
  - `GOWORK=off go test ./... -count=1 -p 1`
  - `GOWORK=off go run ./cmd/protocolmapgen -check -out docs/protocol_map.md`
  - 结果：通过
- `MyFlowHub-Server`
  - 临时 `go.work` 环境下执行 `go test ./protocol/... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-SubProto/auth`
  - 临时 `go.work` 环境下执行 `go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Win`
  - 临时 `go.work` 环境下执行 `go test ./internal/services/... -count=1 -p 1`
  - `npm test -- PermitIssuance authority_admin`
  - `$env:GOWORK='off'; wails generate module`
  - `npm run build`
  - 结果：通过

## Merge 结果
- [repo/MyFlowHub-Proto](../../repo/MyFlowHub-Proto)
  - `main` 已 fast-forward 到 `7a4b09a`
- [repo/MyFlowHub-Server](../../repo/MyFlowHub-Server)
  - `main` 已 fast-forward 到 `ef5de15`
- [repo/MyFlowHub-SubProto](../../repo/MyFlowHub-SubProto)
  - `main` 已 fast-forward 到 `ddd1a7b`
- [repo/MyFlowHub-Win](../../repo/MyFlowHub-Win)
  - `main` 已 fast-forward 到 `8d385db`

## 潜在影响与回滚方案
- 潜在影响：
  - 准入许可页不再保留 latest-only 卡片；用户入口统一收敛到活动 permit 列表
  - 旧 binding 记录在运行时会更接近当前角色配置，而不是保留“role 存在但 perms 为空”的脏状态
  - 仍停留在旧 Proto / Server / SubProto / Win 基线的下游无法识别 permit list action
- 回滚方案：
  - `MyFlowHub-Proto`：回退 permit list action / struct 与 protocol map
  - `MyFlowHub-Server`：回退 protocol 壳、`docs/specs/auth.md` 和 `docs/specs/protocol_map.md`
  - `MyFlowHub-SubProto`：回退 permit list action、权限回填修复与相关测试
  - `MyFlowHub-Win`：回退服务层、列表 store、页面与相关测试 / i18n / docs

## 子Agent执行轨迹
- 无子 agent
