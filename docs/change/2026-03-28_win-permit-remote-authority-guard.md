# 2026-03-28 Win 准入许可页 remote authority 超时收敛

## 变更背景 / 目标
- 在 [2026-03-28_win-permit-load-feedback.md](2026-03-28_win-permit-load-feedback.md) 修复 Win 本地 permit list 解析和页面内错误提示之后，用户仍在 `Permit Issuance` 页遇到：
  - `加载准入许可失败。`
  - `auth list_register_permits: request timed out`
- 本轮排查确认，问题不再是 Win 页面误判成功回包，而是当前 backend 的审批 / permit 管理动作仍以 authority 本地操作为前提。
- 目标：
  - 把 remote authority 场景下的模糊 timeout 收敛成显式 authority-local 限制提示
  - 在 Win orchestration 层提前失败，避免继续等待 auth timeout
  - 将该限制写入长期 requirements/specs 与 lessons

## 具体变更内容
### MyFlowHub-Win
- 更新 [service.go](../../repo/MyFlowHub-Win/internal/services/permission/service.go)
  - 为 `ListRegisterPermits`
  - `IssueRegisterPermit`
  - `RevokeRegisterPermit`
  - 增加 `ensureAuthorityLocalPermitAction(...)`
  - 在 `source_id != authority_id` 时直接返回 `requires authority-local session`
- 更新 [service_test.go](../../repo/MyFlowHub-Win/internal/services/permission/service_test.go)
  - 新增 remote authority permit 管理快速失败回归
- 更新 [PermitIssuance.vue](../../repo/MyFlowHub-Win/frontend/src/pages/PermitIssuance.vue)
  - 识别 `authorityId != current nodeId` 的 remote authority 场景
  - 受限时停止自动加载 permit 列表
  - 显示 authority-local 提示
  - 禁用 `Refresh / New Permit`
  - authority 本机场景下保持现有 permit 列表、签发、撤销路径不变
- 更新 [PermitIssuance.test.ts](../../repo/MyFlowHub-Win/frontend/src/pages/PermitIssuance.test.ts)
  - 补 remote authority 受限态回归
- 更新 [operations.ts](../../repo/MyFlowHub-Win/frontend/src/i18n/messages/operations.ts)
  - 新增 remote authority permit 限制提示中文文案
- 更新 [authority-admin-console.md](../../repo/MyFlowHub-Win/docs/requirements/authority-admin-console.md)
  - 澄清 permit 页在 remote authority 场景下必须显式提示并停止继续等待 timeout
- 更新 [authority-admin-console.md](../../repo/MyFlowHub-Win/docs/specs/authority-admin-console.md)
  - 澄清 permit 页 authority-local 受限态与 orchestration 快速失败约束
- repo 级归档：
  - [2026-03-28_win-permit-remote-authority-guard.md](../../repo/MyFlowHub-Win/docs/change/2026-03-28_win-permit-remote-authority-guard.md)
  - [authority-local-admin-actions.md](../../repo/MyFlowHub-Win/docs/lessons/authority-local-admin-actions.md)

## Requirements impact
- `updated`

## Specs impact
- `updated`

## Lessons impact
- `updated`

## Related requirements
- [authority-admin-console.md](../../repo/MyFlowHub-Win/docs/requirements/authority-admin-console.md)

## Related specs
- [authority-admin-console.md](../../repo/MyFlowHub-Win/docs/specs/authority-admin-console.md)
- [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)

## Related lessons
- [authority-local-admin-actions.md](../lessons/authority-local-admin-actions.md)

## 对应 plan.md 任务映射
- `PERMIT-GUARD-1`
- `PERMIT-GUARD-2`
- `DOC-CLARIFY-1`
- `VALIDATE-1`
- `REVIEW-1`
- `ARCHIVE-1`
- `MERGE-1`

## 经验 / 教训摘要
- `list_register_permits` 超时不一定是 Win 页面 bug；也可能是当前 backend 根本没有把该管理动作实现成通用 remote authority 链路。
- 对这种“后端能力边界”场景，页面应优先展示显式限制提示，而不是继续表现成普通失败态。
- authority-local guard 放到 Win orchestration 层后，页面外的其它调用者也不会再无意义地等满 auth timeout。

## 可复用排查线索
- 症状：
  - permit 页打开就提示 `加载准入许可失败。`
  - 详细错误是 `auth list_register_permits: request timed out`
  - 当前登录节点并不是 authority 节点
- 触发条件：
  - `sourceId != authorityId`
  - backend 仍要求审批 / permit 管理从 authority 本机发起
- 关键词：
  - `requires authority-local session`
  - `list_register_permits`
  - `authorityId != sourceId`
  - `data-permit-remote-authority`
- 快速检查：
  - 查看当前 session `nodeId` 是否等于 `authorityId`
  - 查看 [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md) 是否仍注明审批 / permit 管理建议从 authority 节点操作
  - 查看 [PermitIssuance.vue](../../repo/MyFlowHub-Win/frontend/src/pages/PermitIssuance.vue) 是否在 remote authority 下跳过自动加载并展示提示

## 关键设计决策与权衡
- 不扩到跨仓 remote authority permit 管理链路，只在 Win 侧做 authority-local guard
  - 优点：改动面小，行为和当前 backend 事实一致
  - 代价：这轮不提供真实 remote authority permit 管理能力
- 受限态使用页面内提示，而不是继续等 timeout
  - 优点：用户能直接理解限制边界
  - 代价：若后端将来补齐 remote 链路，需要回退 guard

## 测试与验证方式 / 结果
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./internal/services/permission/... -count=1`
  - 结果：通过
- `MyFlowHub-Win/frontend`
  - `npm test -- PermitIssuance`
  - 结果：通过
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：仍会打印 `Not found: time.Time`，但退出码为 0
- `MyFlowHub-Win/frontend`
  - `npm run build`
  - 结果：通过

## Merge 结果
- [repo/MyFlowHub-Win](../../repo/MyFlowHub-Win)
  - `main` 已 fast-forward 到 `2e10af8`
  - commit: `fix: 收敛准入许可页 remote authority 限制`

## 潜在影响与回滚方案
- 潜在影响：
  - remote authority 场景下 permit 页不再尝试自动加载，也不再把 timeout 暴露成普通失败
  - authority 本机场景保持原有 permit 列表、签发、撤销路径
- 回滚方案：
  - 回退 `MyFlowHub-Win` 中的 `service.go`、`service_test.go`、`PermitIssuance.vue`、`PermitIssuance.test.ts`、`operations.ts`
  - 回退 `MyFlowHub-Win` 中的 requirements/specs/lesson 更新
  - 回退 [2026-03-28_win-permit-remote-authority-guard.md](../../repo/MyFlowHub-Win/docs/change/2026-03-28_win-permit-remote-authority-guard.md)

## Worktree Cleanup
- 已移除并 prune：
  - `D:\project\MyFlowHub3\worktrees\fix-win-permit-list-timeout-route`
- 已删除本地 feature branch：
  - `fix/win-permit-list-timeout-route`

## 子Agent执行轨迹
- 无子 agent
