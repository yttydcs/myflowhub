# Plan - Win 准入许可远程 authority 超时收敛

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-permit-list-timeout-route`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-list-timeout-route`
- Current Stage: `4 archive complete, workflow ended`

## Stage Records

### Initialization
- `guide.md`
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`
  - 已读取 `$m-docs` 的 requirement impact 和 indexing 规则入口
- base/worktree confirmation
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `fix/win-permit-list-timeout-route`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-list-timeout-route`
  - implementation stayed inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 收敛 `Permit Issuance` 页在远程 authority 场景下的真实超时，把“当前后端不支持远程 authority permit 管理”从模糊 timeout 改成前端可理解、可预期的限制提示。

#### Scope
- 必须
  - 识别 `sourceId != authorityId` 的远程 authority 场景
  - permit 页在该场景下不再自动触发 `list_register_permits` 并等待超时
  - 页面需要给出明确的本地 authority 限制提示，并禁用不成立的 permit 管理动作
  - Go orchestration 层对 permit 管理动作提前返回可读错误，避免继续等待 auth timeout
  - 补齐前端和 Go 回归测试
- 不做
  - 不扩到 `MyFlowHub-Server` / `MyFlowHub-SubProto` 的远程 authority 分布式审批链路
  - 不修改 auth wire / protocol
  - 不改访问策略页

#### Acceptance Criteria
- 在 `sourceId != authorityId` 时，permit 页不再出现 `auth list_register_permits: request timed out`
- 页面能明确提示“当前需要在 authority 节点本地操作”
- authority 本地场景下现有 permit 列表、签发、撤销路径不变
- `go test ./internal/services/permission/...` 和 `npm test -- PermitIssuance` 通过

### Stage 2 - Architecture Design
#### Overall Solution
- 在 Win `PermissionService` 中为 permit 管理动作增加 `authority-local` 前置校验。
- permit 页基于 authority store 的 `sourceId/authorityId` 计算 `remoteAuthorityBlocked`。
- 当受限时，permit 页跳过自动加载并展示稳定提示，同时禁用刷新和新建。
- 继续保留 authority 本地场景下的原有加载和动作链路。

#### Alternatives Considered
- 方案 A（采用）：Win orchestration guard + 页面受限态
  - 优点：最小安全改动，行为与当前 backend 事实一致
  - 代价：不提供真实 remote authority permit 管理能力
- 方案 B：扩到 Server/SubProto 实现远程 authority permit 管理
  - 不选理由：涉及跨仓 runtime 行为，明显超出本轮 Win 缺陷修复范围

### Stage 3.1 - Planning
#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- Requirements impact: `clarify`
- Specs impact: `clarify`
- Related requirements
  - `repo/MyFlowHub-Win/docs/requirements/authority-admin-console.md`
- Related specs
  - `repo/MyFlowHub-Win/docs/specs/authority-admin-console.md`
  - `repo/MyFlowHub-Server/docs/specs/auth.md`
- Related lessons
  - `none`

#### Executable Task List
- [x] `PERMIT-GUARD-1` 在 Go orchestration 层增加 permit authority-local guard
- [x] `PERMIT-GUARD-2` 在 permit 页增加 remote authority 受限提示与按钮禁用
- [x] `DOC-CLARIFY-1` 更新 requirements/specs，明确当前远程 authority 限制提示要求
- [x] `VALIDATE-1` 补测试并执行 Go / 前端验证
- [x] `REVIEW-1` 完成 3.3 checklist
- [x] `ARCHIVE-1` 归档到 `docs/change`
- [x] `MERGE-1` 合并回 `repo/MyFlowHub-Win/main`
- [x] `CLEANUP-1` 删除 worktree / local branch

### Stage 3.2 - Implementation
#### Task Mapping
- `PERMIT-GUARD-1`
  - `internal/services/permission/service.go`
  - `internal/services/permission/service_test.go`
- `PERMIT-GUARD-2`
  - `frontend/src/pages/PermitIssuance.vue`
  - `frontend/src/pages/PermitIssuance.test.ts`
- `DOC-CLARIFY-1`
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`

#### File-level Change Summary
- `internal/services/permission/service.go`
  - 为 permit 管理动作增加 authority-local 前置校验
- `internal/services/permission/service_test.go`
  - 覆盖 remote authority 快速失败
- `frontend/src/pages/PermitIssuance.vue`
  - remote authority 下跳过自动加载
  - 展示 authority-local 受限提示
  - 禁用 `Refresh / New Permit`
- `frontend/src/pages/PermitIssuance.test.ts`
  - 补 remote authority 受限态回归
- `frontend/src/i18n/messages/operations.ts`
  - 新增 permit remote authority 提示文案
- `docs/requirements/authority-admin-console.md`
  - 澄清 permit 页 remote authority 限制提示要求
- `docs/specs/authority-admin-console.md`
  - 澄清 authority-local 受限态与快速失败契约

#### Validation
- `$env:GOWORK='off'; go test ./internal/services/permission/... -count=1`
  - 结果：通过
- `npm test -- PermitIssuance`
  - 结果：通过
- `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：仍会打印 `Not found: time.Time`，但退出码为 0
- `npm run build`
  - 结果：通过

### Stage 3.3 - Code Review
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
- 子Agent治理与审计：通过

### Stage 4 - Change Archive
- repo archive
  - `repo/MyFlowHub-Win/docs/change/2026-03-28_win-permit-remote-authority-guard.md`
- repo lesson
  - `repo/MyFlowHub-Win/docs/lessons/authority-local-admin-actions.md`
- workspace archive
  - `docs/change/2026-03-28_win-permit-remote-authority-guard.md`
- workspace lesson
  - `docs/lessons/authority-local-admin-actions.md`
- index updates
  - `repo/MyFlowHub-Win/docs/change/README.md`
  - `repo/MyFlowHub-Win/docs/lessons/README.md`
  - `docs/change/README.md`
  - `docs/plan/README.md`
  - `docs/lessons/README.md`
  - `plan.md`

## Merge Results
- `repo/MyFlowHub-Win/main`
  - fast-forward 到 `2e10af8`
  - commit: `fix: 收敛准入许可页 remote authority 限制`

## Worktree Cleanup
- 已移除并 prune：
  - `D:\project\MyFlowHub3\worktrees\fix-win-permit-list-timeout-route`
- 已删除本地 feature branch：
  - `fix/win-permit-list-timeout-route`

## Notes
- 本轮不承诺 remote authority permit 管理真实可用，只把当前 backend 边界显式化。
