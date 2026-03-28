# Plan - Win 准入许可页加载反馈与头部动作收敛

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-permit-issuance-timeout-ui`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-issuance-timeout-ui`
- Current Stage: `4 archive complete, workflow ended`

## Stage Records

### Initialization
- guide.md:
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`、`references/templates.md`
  - 已读取 `$m-docs` 的 requirement/spec impact 与 indexing 规则
- base/worktree confirmation:
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `fix/win-permit-issuance-timeout-ui`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-issuance-timeout-ui`
  - implementation stayed inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 修复准入许可页的 permit list 解析/加载失败体验，并收紧头部布局。

#### Scope
- 必须:
  - 修复 `ListRegisterPermitsResp` 的响应码解析缺口
  - permit 页加载失败时改为页面内错误提示，不直接弹侵入式 toast
  - 将 `Refresh / New Permit` 移到“活动许可”卡片右上角
  - 移除 `共 X 条 / 实时` 标签
  - 补齐 Go / 前端回归测试
- 不做:
  - 不改 auth 协议或其它 repo
  - 不重做注册审批页或访问策略页
  - 不新增 permit 历史查询或轮询

#### Acceptance Criteria
- permit list 成功回包不再被误判
- 页面首屏失败时显示页面内错误态
- permit card header 中出现 `Refresh / New Permit`
- `共 X 条 / 实时` 标签消失
- Go / 前端测试通过

### Stage 2 - Architecture Design
#### Overall Solution
- 在 `internal/services/auth/service.go` 中为 `ListRegisterPermitsResp` 补齐统一 `code/msg` 抽取。
- 在 `PermitIssuance.vue` 中新增本地 `loadState.error`，将 permit list 失败反馈局部化到 permit card。
- 将按钮从介绍卡移动到“活动许可”卡片 header actions，并删除 permit 总数和“实时”标签。

#### Alternatives Considered
- 方案 A（采用）：修复响应码解析 + 页面内错误提示 + permit card header 收敛
  - 优点：最小安全改动，直接命中当前用户问题
  - 代价：若 authority 端真实不回包，页面仍只会显示失败状态
- 方案 B：只改布局，不动超时链路
  - 不选理由：用户的核心问题仍在
- 方案 C：扩大到 authority 解析或跨 hop permit forwarding
  - 不选理由：超出本轮单仓范围，且缺少证据支撑

### Stage 3.1 - Planning
#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划与归档路由。
- 结论：
  - Requirements impact: `none`
  - Specs impact: `none`
  - Related requirements:
    - `repo/MyFlowHub-Win/docs/requirements/authority-admin-console.md`
  - Related specs:
    - `repo/MyFlowHub-Win/docs/specs/authority-admin-console.md`
    - `repo/MyFlowHub-Server/docs/specs/auth.md`
  - Related lessons: `none`

#### Executable Task List
- [x] `PERMIT-FIX-1` 修复 permit list auth response code 解析
- [x] `PERMIT-FIX-2` 收敛 permit 页加载失败反馈和头部布局
- [x] `PERMIT-FIX-3` 补齐 Go / 前端回归测试
- [x] `REVIEW-1` 完成 3.3 checklist
- [x] `ARCHIVE-1` 写入 repo / workspace 归档
- [x] `MERGE-1` 合并回 `repo/MyFlowHub-Win/main`
- [x] `CLEANUP-1` 删除 worktree / local branch

#### Task Details
##### `PERMIT-FIX-1` - Permit List Response Code Fix
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-issuance-timeout-ui`
- Goal:
  - 让 `ListRegisterPermitsResp` 正确走统一 `code/msg` 判定
- Files:
  - `internal/services/auth/service.go`
  - `internal/services/auth/service_test.go`
- Acceptance:
  - permit list 成功回包不再被误判为 `code=0`
- Tests:
  - `$env:GOWORK='off'; go test ./internal/services/auth/... -count=1`
- Rollback:
  - 回退响应码抽取和对应测试

##### `PERMIT-FIX-2` - Permit Page Feedback And Header Cleanup
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-issuance-timeout-ui`
- Goal:
  - 让 permit 页在失败时可读、可重试，并把主动作收敛到 permit card header
- Files:
  - `frontend/src/pages/PermitIssuance.vue`
  - `frontend/src/i18n/messages/operations.ts`
- Acceptance:
  - 自动加载失败仅显示页面内错误态
  - `Refresh / New Permit` 位于 permit card header
  - 不再显示 `共 X 条 / 实时`
- Tests:
  - `npm test -- PermitIssuance`
- Rollback:
  - 回退 permit 页面和对应文案

##### `PERMIT-FIX-3` - Regression Coverage
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-permit-issuance-timeout-ui`
- Goal:
  - 用 Go / 前端测试固定本轮行为
- Files:
  - `internal/services/auth/service_test.go`
  - `frontend/src/pages/PermitIssuance.test.ts`
- Acceptance:
  - 覆盖 permit list response code 解析和页面错误/布局状态
- Tests:
  - `$env:GOWORK='off'; go test ./internal/services/auth/... -count=1`
  - `npm test -- PermitIssuance`
- Rollback:
  - 回退新增测试

### Stage 3.2 - Implementation
#### Change Summary
- `internal/services/auth/service.go`
  - `extractAuthCodeMsg()` 补齐 `*ListRegisterPermitsResp`
- `frontend/src/pages/PermitIssuance.vue`
  - 新增 `loadState.error`
  - permit list 失败时改为 card 内错误提示
  - `Refresh / New Permit` 挪到“活动许可”卡片右上角
  - 移除 permit 总数和“实时”标签
- `frontend/src/pages/PermitIssuance.test.ts`
  - 补动作位置和首屏错误态断言
- `frontend/src/i18n/messages/operations.ts`
  - 新增 `Failed to load permits.` 中文文案

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
- repo archive:
  - `repo/MyFlowHub-Win/docs/change/2026-03-28_win-permit-load-feedback.md`
- workspace archive:
  - `docs/change/2026-03-28_win-permit-load-feedback.md`
- index updates:
  - `repo/MyFlowHub-Win/docs/change/README.md`
  - `docs/change/README.md`
  - `docs/plan/README.md`
  - `plan.md`

## Validation Results
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./internal/services/auth/... -count=1`
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

## Merge Results
- `repo/MyFlowHub-Win/main`
  - fast-forward 到 `0ddcdec`
  - commit: `fix: 收敛准入许可页加载反馈`

## Worktree Cleanup
- 已移除并 prune：
  - `D:\project\MyFlowHub3\worktrees\fix-win-permit-issuance-timeout-ui`
- 已删除本地 feature branch：
  - `fix/win-permit-issuance-timeout-ui`

## Notes
- 若现场仍看到 `auth list_register_permits: request timed out`，则更可能是 authority 端真实没有回 `list_register_permits_resp`，而不是 Win 本地解析问题。
