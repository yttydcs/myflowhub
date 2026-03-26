# Plan - Win Authority Console Refactor

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `feat/win-register-approval`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-register-approval`
- Current Stage: `4 archive complete, awaiting workflow end confirmation`

## Stage Records

### Initialization
- `guide.md`: workspace root `guide.md` and `$m-autoflow` initialization/stage docs already read.
- base/worktree confirmation:
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-register-approval`
  - dedicated branch: `feat/win-register-approval`
  - implementation will stay inside the worktree only.

### Stage 1 - Requirements Analysis
#### Goal
- 将 Authority 管理能力从单一 `Permissions` 页面重构为一组独立入口，分别覆盖权限编排、注册审批、permit 签发，并提高权限编辑的可用性。

#### Scope
- 必须:
  - 左侧新增一组 3 个独立入口，分别承载：
    - 权限编排
    - 注册审批
    - permit 签发
  - 页面文案英文环境保持全英文，中文环境保持全文中文。
  - 权限编排页面保留现有能力，但重做交互布局，提高可读性和编辑效率。
  - 注册审批页面支持查看待审批注册列表，并执行 approve / reject。
  - permit 页面支持 issue / revoke，且对当前会话新签发的 permit 提供清晰反馈。
- 可选:
  - 对审批列表增加本地筛选 / 快速统计，只要不引入新的协议依赖。
  - 对最新签发 permit 提供便捷复制与快速回收入口。
- 不做:
  - 不修改 `MyFlowHub-Server` 协议。
  - 不新增 permit 列表协议或持久化 permit 历史。
  - 不调整 Home 登录 / 注册主流程语义。

#### Use Cases
- Authority 节点管理员希望单独进入“注册审批”页面，集中处理首次注册请求。
- 管理员希望在“permit 签发”页面快速给指定设备签发一次性准入 token，并在必要时撤销。
- 管理员希望在“权限编排”页面更直观地调整默认角色、角色权限和节点覆盖，而不是在单一长表单里滚动编辑。

#### Functional Requirements
- 左侧导航必须出现同一组的 3 个入口，并分别路由到独立页面。
- 权限编排页面必须继续支持：
  - authority 解析
  - policy load / save
  - runtime list_roles 预览
  - get_perms 查询
- 注册审批页面必须支持：
  - authority 解析
  - 待审批注册列表加载与刷新
  - 查看 `request_id`、`device_id`、`requested_role`、`display_name`、创建时间、过期时间
  - 对单条申请执行 approve / reject
- permit 页面必须支持：
  - authority 解析
  - 按 `device_id + role + expires_at(optional)` 签发 permit
  - 按 permit token 撤销 permit
  - 显示最近一次成功签发结果
- 所有页面都必须在未连接、未登录、缺少 node/hub 身份时显式阻止操作并给出一致错误。

#### Non-functional Requirements
- 复用现有 `PermissionService` / `AuthService` / `ManagementService`，保持改动面可控。
- 前端结构应从“单页堆叠”重构为更清晰的职责分离，避免逻辑继续膨胀。
- 不引入可避免的重复请求；动作成功后只刷新必要状态。
- permit token 默认不做本地持久化，避免在本地配置中长期保存敏感准入凭证。

#### Inputs / Outputs
- 输入:
  - 当前登录身份 `source_id / hub_id`
  - authority override
  - policy 编辑表单
  - approval 的 `request_id / role / reason`
  - permit 的 `device_id / role / expires_at / permit`
- 输出:
  - authority 解析结果
  - policy / runtime / node perms 数据
  - pending register 列表与 approve/reject 结果
  - permit issue/revoke 结果与最新签发 token

#### Edge Cases
- authority override 非正整数。
- 待审批列表为空。
- approve 时 role 留空，需按协议保持空值而不是强制前端填充。
- reject 原因为空。
- permit 撤销输入空 token。
- permit 页面无法查询历史 permit，页面只能展示本会话最近一次成功签发结果。

#### Acceptance Criteria
- 左侧导航出现 3 个独立入口，英文环境全英文，中文环境全文中文。
- `Permissions` 单页被重构为职责清晰的 3 个页面，且原有 policy 能力不回退。
- 注册审批页面可以完成 list pending + approve + reject 基础链路。
- permit 页面可以完成 issue + revoke，并展示最近一次签发结果。
- Go 测试、前端测试、前端构建、Wails bindings 生成通过。

#### Risks
- 现有 `PermissionService` 作用域扩大后，方法命名和职责边界必须保持清晰，避免再次形成“超级服务”。
- permit 协议没有 list 能力，UI 只能做“签发/撤销 + 最近结果”，需要通过文案明确边界。
- 页面拆分后会涉及路由、导航、i18n 和 bindings 多点同步，漏改会直接导致构建失败。

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 保持现有后端协议和 Wails 基础设施不变，在 Win 端做 Authority Console 重构：
  - `PermissionService` 扩展 authority 管理接口；
  - 前端拆出 3 个独立页面和 store；
  - 左侧导航新增独立 Authority 组；
  - 权限编排页面重做布局和表单组织。

#### Alternatives Considered
- 方案 A（采用）：保留单个 Wails authority orchestration service，前端拆分为 3 个页面/store。
  - 优点：后端改动小，authority 解析逻辑可复用。
  - 代价：`PermissionService` 会承载除 policy 外的 authority 管理动作，需要明确命名。
- 方案 B：新增 3 个 Wails 服务分别处理 policy / approval / permit。
  - 优点：服务名更纯粹。
  - 代价：bindings、依赖注入、共享 authority 解析与测试面更分散，本轮改动面更大。

#### Module Responsibilities
- `internal/services/auth`
  - 提供 typed auth action 调用：list pending / approve / reject / issue permit / revoke permit。
- `internal/services/permission`
  - 继续承载 authority orchestration：resolve authority、policy load/save、node perms、approval/permit 编排接口。
- `frontend/src/stores/authority*.ts`
  - 页面状态与调用封装，负责 authority 上下文、表单状态、回刷策略。
- `frontend/src/pages/*`
  - 3 个独立页面分别呈现 policy、approval、permit 交互。
- `frontend/src/layout/AppShell.vue` / `frontend/src/router/index.ts`
  - 左侧导航与路由编排。

#### Data / Call Flow
1. 页面读取登录态并同步 `sourceId/hubId`。
2. 用户在任一 authority 页面解析 authority。
3. policy 页面：
   - load policy -> load runtime roles
   - save policy -> persist/apply/invalidate/verify
4. approval 页面：
   - list pending -> approve/reject -> refresh list
5. permit 页面：
   - issue permit -> show latest issued token
   - revoke permit -> clear / update latest token when matched

#### Interface Drafts
- `PermissionService`
  - existing:
    - `ResolveAuthority`
    - `LoadPolicy`
    - `SavePolicy`
    - `GetNodePerms`
  - new:
    - `ListPendingRegisters`
    - `ApproveRegister`
    - `RejectRegister`
    - `IssueRegisterPermit`
    - `RevokeRegisterPermit`
- frontend pages:
  - `/access-policy`
  - `/registration-approvals`
  - `/permit-issuance`

#### Error Handling and Safety
- authority 未解析、本地身份缺失、关键字段为空时，前端本地立即失败。
- 后端 typed action 必须校验必填字段，并透传服务端错误码 / 消息。
- permit token 不持久化到本地配置，只保留内存态最近结果。
- 所有动作按钮在 loading/saving 时禁止重复提交。

#### Performance and Testing Strategy
- authority 页面只在显式加载或动作成功后刷新必要数据，不做无意义轮询。
- Go 测试：
  - 扩展 `internal/services/permission/service_test.go`
  - 为新增 auth typed helper 增加最小覆盖
- 前端测试：
  - 新增 authority 相关 store tests，覆盖 authority 解析、approval/permit 结果状态更新
- 验证：
  - `go test ./... -count=1`
  - `npm test`
  - `npm run build`
  - `wails generate module`

#### Extensibility Design Points
- 未来若 Server 新增 permit list/query，可在 permit 页面无破坏接入。
- 未来若 authority 管理继续扩展，可沿相同模式新增独立页面，而不再把能力塞回单一页面。
- 前端若后续需要共享 authority 头部状态，可抽公共组件而不改页面契约。

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 将现有 Authority 管理重构为一组独立页面，覆盖 policy / approval / permit 三类操作，并提升 policy 页的易用性。
- Current State:
  - 现有仓内仅有 `/permissions`，功能集中在 policy 编辑。
  - `AuthService` 只暴露 policy 相关 typed action，缺少审批和 permit 的 GUI 接口。
  - 左侧导航没有独立 Authority 分组。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: add
- Specs impact: add
- Stable docs destination:
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- Change archive destination:
  - `docs/change/2026-03-26_win-authority-console-refactor.md`
- Lessons impact:
  - none (当前未发现跨 workflow 复用的排障知识)

#### Related Requirements / Specs / Lessons
- Requirements:
  - `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-register-approval\docs\requirements\authority-admin-console.md`
- Specs:
  - `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-register-approval\docs\specs\authority-admin-console.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- Lessons:
  - none

#### Executable Task List
- [x] DOCS-1 新增 authority admin requirements/specs 与索引
- [x] IMPL-1 扩展 auth / permission 服务支持 approval + permit
- [x] IMPL-2 拆分 authority 前端路由、导航和 store
- [x] IMPL-3 重做 policy 页面交互布局并补 approval / permit 页面
- [x] IMPL-4 生成 bindings、补测试并完成构建验证

#### Task Details
##### DOCS-1 - Authority 稳定文档
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-register-approval`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-register-approval\plan.md`
- Goal: 为 Authority Console 重构建立稳定 requirements/specs 真相入口
- Files / Modules:
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
  - `docs/requirements/README.md`
  - `docs/specs/README.md`
- Write Set:
  - 同上
- Acceptance:
  - requirements/specs 可导航，且不把稳定真相只留在 change
- Test Points:
  - 文档自检与交叉链接检查
- Rollback:
  - 删除新增叶子文档并恢复索引

##### IMPL-1 - Authority 服务扩展
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-register-approval`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-register-approval\plan.md`
- Goal: 让 Win 端已有 authority orchestration service 可覆盖 approval / permit 动作
- Files / Modules:
  - `internal/services/auth/*`
  - `internal/services/permission/*`
  - `app.go`
- Write Set:
  - auth typed request/response helpers
  - permission orchestration methods and tests
- Acceptance:
  - 前端可通过 Wails 调用 approval / permit 接口
  - authority resolve 逻辑保持兼容
- Test Points:
  - Go tests covering new orchestration / helper logic
- Rollback:
  - 回退新增 authority methods 与注入改动

##### IMPL-2 - Authority 页面拆分与导航
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-register-approval`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-register-approval\plan.md`
- Goal: 左侧形成独立 Authority 组，并拆出 3 个清晰入口
- Files / Modules:
  - `frontend/src/router/index.ts`
  - `frontend/src/layout/AppShell.vue`
  - `frontend/src/i18n/messages/*.ts`
  - `frontend/src/stores/*.ts`
- Write Set:
  - nav / route / store state
- Acceptance:
  - 左侧出现 3 个 authority 入口
  - 中英文文案符合“全英文 / 全中文”要求
- Test Points:
  - 前端 store tests
  - `npm run build`
- Rollback:
  - 回退 nav / route / store 拆分

##### IMPL-3 - Policy / Approval / Permit 页面实现
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-register-approval`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-register-approval\plan.md`
- Goal: 提升 policy 页易用性，并补齐 approval / permit 页面
- Files / Modules:
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/pages/RegistrationApprovals.vue`
  - `frontend/src/pages/PermitIssuance.vue`
  - related stores / shared helpers
- Write Set:
  - new pages
  - refactored policy layout
  - approval/permit forms and result cards
- Acceptance:
  - policy 页比原版本更易读、分区更清晰
  - approval / permit 页面核心动作可用
- Test Points:
  - `npm test`
  - `npm run build`
- Rollback:
  - 回退新页面并恢复旧 `/permissions`

##### IMPL-4 - Binding / Tests / Validation
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-register-approval`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-register-approval\plan.md`
- Goal: 确保 bindings、测试与构建都覆盖本轮变更
- Files / Modules:
  - `frontend/wailsjs/**`
  - `*_test.go`
  - `frontend/src/**/*.test.ts`
- Write Set:
  - generated bindings
  - tests
- Acceptance:
  - `go test ./... -count=1`
  - `npm test`
  - `npm run build`
  - `wails generate module`
- Test Points:
  - 见 Acceptance
- Rollback:
  - 回退测试与 bindings 改动

#### Dependencies
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- Wails generated bindings workflow
- 现有 `AuthService` / `ManagementService` / `PermissionService`

#### Risks and Notes
- permit token 只保留内存态最近结果，是安全优先的默认策略；若后续需要持久 permit 历史，应另起 workflow 明确安全边界。
- 若实现过程中发现必须新增 Server 的 permit list/query 契约，需要返回 `3.1` 更新 plan，不直接扩 scope。
- 原 `/permissions` 页面将被替换为新 `AccessPolicy` 页面；若发现其它模块强依赖旧路由名，需要同步修正。

#### Parallelism Assessment
- 不使用子 Agent。
- 理由:
  - 后端 service 扩展、bindings、前端 store/page/i18n 写集重叠明显
  - policy 页面重构与 approval/permit 接口联调共享同一 authority 语义，拆分后集成成本高

#### Issue List
- none

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Task Execution Record
- `DOCS-1`
  - 新增 `docs/requirements/authority-admin-console.md` 与 `docs/specs/authority-admin-console.md`。
  - 更新 `docs/requirements/README.md` 与 `docs/specs/README.md` 索引，建立稳定真相入口。
- `IMPL-1`
  - 在 `internal/services/auth/authority.go` 增加 Win 本地 approval / permit typed payload，避免在 `GOWORK=off` 模式下强制 bump `myflowhub-proto`。
  - 扩展 `internal/services/auth/service.go` 与 `internal/services/permission/service.go`，补齐 `ListPendingRegisters`、`ApproveRegister`、`RejectRegister`、`IssueRegisterPermit`、`RevokeRegisterPermit`。
  - 新增 / 更新 Go tests：`internal/services/auth/authority_test.go`、`internal/services/permission/service_test.go`。
- `IMPL-2`
  - 左侧导航新增 Authority 组，拆分为 `Access Policy`、`Registration Approvals`、`Permit Issuance` 三个独立入口。
  - 路由新增 `/access-policy`、`/registration-approvals`、`/permit-issuance`，并保留 `/permissions`、`/approvals`、`/permits` 重定向兼容。
  - 新增共享 authority store，并更新 shell / operations i18n 文案，满足英文全英文、中文全中文。
- `IMPL-3`
  - 删除旧单页 `Permissions.vue` / `permissions.ts`。
  - 新增 `AccessPolicy.vue`，将 authority 解析、policy 编辑、运行时预览、节点权限查询拆成独立区域，并加入表单校验、摘要卡片与更清晰的动作分组。
  - 新增 `RegistrationApprovals.vue` 与 `PermitIssuance.vue`，分别承载审批与 permit 核心链路。
  - 新增 `accessPolicy.ts`、`registrationApprovals.ts`、`permitIssuance.ts` 页面级 store。
- `IMPL-4`
  - 更新前端 store tests：`frontend/src/stores/authority_admin.test.ts`。
  - 验证通过：
    - `$env:GOWORK='off'; go test ./... -count=1`
    - `npm test`
    - `npm run build`
    - `$env:GOWORK='off'; wails generate module`
  - 说明：
    - 直接执行 `wails generate module` 会因 worktree 不在根 `go.work` modules 列表内失败，本 workflow 固定使用 `GOWORK=off`。

#### Scope Control
- 未扩展到 Server 协议修改。
- 未新增 permit 历史持久化或查询接口。
- 未改动 Home 登录 / 注册主流程语义。

#### Issue List
- none

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 三个独立入口、独立页面、policy/approval/permit 核心链路、双语文案和旧路由兼容均已落地。
- 架构合理性：通过
  - 保持单个 `PermissionService` 作为 authority orchestration 边界，前端按页面拆 store，避免继续堆叠单页逻辑。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 页面改为显式加载和动作后定向刷新，无新增轮询或明显重复请求。
  - `npm run build` 仍有现存主包 chunk size warning，但本轮未新增额外构建阻塞。
- 可读性与一致性：通过
  - 新页面命名与路由语义对齐，输入校验与错误提示在三页中保持一致。
- 可扩展性与配置化：通过
  - approval / permit 数据结构收敛在 Win 本地 typed helper，后续若 proto 升级或增加 permit list 可平滑替换。
- 稳定性与安全：通过
  - 前端对未连接 / 未登录 / 缺少 node/hub 身份做显式阻止；后端校验必填字段；permit token 仅保留最近一次内存态结果。
- 测试覆盖情况：通过
  - Go tests、Vitest、Vite build、Wails module generation 均已执行通过。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 未使用子 Agent，所有改动与验证均由主 Agent 在单一 worktree 内完成。

### Stage 4 - Change Archive
- 使用 `$m-docs` 完成变更归档与索引更新。
- Change archive:
  - `docs/change/2026-03-26_win-authority-console-refactor.md`
- Lessons impact:
  - none
- Workflow status:
  - 归档已完成，等待用户确认是否结束当前 workflow。
