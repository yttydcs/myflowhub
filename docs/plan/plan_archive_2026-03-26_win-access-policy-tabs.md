# Plan - Win Access Policy Tabs And Role Management

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `refactor/win-access-policy-tabs`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs`
- Current Stage: `4 archive complete, awaiting workflow end confirmation`

## Stage Records

### Initialization
- `guide.md`:
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`
- base/worktree confirmation:
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `refactor/win-access-policy-tabs`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs`
  - implementation will stay inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 将 Authority 管理中的中文“权限编排”统一改名为“访问策略”，并把 `Access Policy` 页面重构为带 tab 的访问策略控制台：首页保留当前策略编辑/保存/校验，新增“角色管理” tab，把角色权限编辑从主页抽离，并改为预置权限目录勾选而不是手填权限字符串。

#### Scope
- 必须:
  - 中文 UI 中原“权限编排”相关入口、标题、说明统一改为“访问策略”；英文维持 `Access Policy`
  - `Access Policy` 页面采用与 `VarPool` 一致的 tab 交互，默认进入“当前策略”tab
  - “当前策略”tab 保留现有能力：
    - authority 解析
    - policy load / save
    - default role / default perms 配置
    - node overrides
    - runtime snapshot
    - node perms lookup
  - 新增“角色管理”tab，负责：
    - 新增角色
    - 删除角色
    - 编辑角色权限
    - 查看角色权限摘要
  - 默认权限与角色权限都改为从预置权限目录勾选，不再要求用户手填 CSV
  - 默认角色与节点覆盖角色优先通过当前角色列表选择，减少自由输入
  - 对已有策略中不在预置目录内的未知权限，要显式展示并在保存时保留，不能静默丢失
  - 稳定 requirements/specs 同步反映“访问策略”命名和 tab 化职责边界
- 可选:
  - 为角色管理提供角色权限数量摘要和快捷说明
  - 为内置角色（`superadmin` / `admin` / `node`）提供更明显的默认语义提示
- 不做:
  - 不新增 `MyFlowHub-Server` 协议或权限元数据接口
  - 不改变 `auth.default_role` / `auth.default_perms` / `auth.node_roles` / `auth.role_perms` 的存储契约
  - 不修改注册审批页和 permit 页的行为

#### Use Cases
- Authority 管理员希望在中文环境里看到更准确的“访问策略”命名，而不是“权限编排”
- 管理员希望先在首页查看当前默认角色、节点覆盖、运行时状态，再按需切到“角色管理”集中维护角色
- 管理员希望通过勾选权限目录维护角色，而不是自己记忆并手输权限字符串
- 管理员加载了带历史/自定义权限的策略时，页面不会因为 catalog 不完整而误删原配置

#### Functional Requirements
- 左侧导航、页面标题、路由副标题中的中文文案应使用“访问策略”
- `Access Policy` 页面顶部必须提供 tab 切换，至少包含：
  - 当前策略
  - 角色管理
- 当前策略 tab 必须支持：
  - authority override 输入与 resolve
  - load / reload policy
  - default role 选择
  - default perms 勾选
  - node override 列表编辑
  - save options / save
  - runtime snapshot 浏览
  - 单节点权限查询
- 角色管理 tab 必须支持：
  - 角色列表展示
  - 新增角色名
  - 删除角色
  - 通过预置权限目录勾选角色权限
  - 当角色包含未知权限时明确展示为额外权限并在保存时保留
- 保存后的 payload 必须继续落回现有 `Policy` 结构，不改变后端 API
- 角色重复、节点重复、空角色名、非法 node ID 等错误必须继续显式校验

#### Non-functional Requirements
- 以前端改造为主，避免扩大到新的后端接口
- 不引入额外重复请求；tab 切换只切 UI，不重新加载 authority 数据
- 不能因为 catalog 化编辑而丢失未知权限或既有策略信息
- 交互需要比当前长表单更清晰，尤其是权限编辑不再依赖用户手写字符串
- 保持中英文 i18n 一致

#### Inputs / Outputs
- 输入:
  - 当前登录身份 `source_id / hub_id`
  - authority override
  - default role
  - default perms 选择集
  - role definitions（角色名 + 目录勾选权限 + 未知权限保留）
  - node role overrides
  - save options
  - node perms query `node_id`
- 输出:
  - authority 解析结果
  - 结构化访问策略表单状态
  - 保存后的 policy / warnings / runtime snapshot
  - 单节点最终角色与权限

#### Edge Cases
- authority override 非正整数
- policy 中的 `default_role` 不在显式 `role_perms` 中
- `default_perms` 或 `role_perms` 中包含 catalog 外权限
- 角色存在 `*` 时应避免与普通权限勾选语义冲突
- 没有任何显式角色时，默认角色和节点覆盖仍需要有清晰的可编辑体验
- 节点覆盖引用一个没有显式 `role_perms` 的角色时，仍需保留现有 warning 语义

#### Acceptance Criteria
- 中文环境里原“权限编排”入口和页面主文案改为“访问策略”
- `Access Policy` 页面出现 tab，默认进入当前策略页，并新增独立“角色管理”页
- 页面中不再要求用户通过 CSV 文本框手动编辑默认权限或角色权限
- 保存后仍通过现有 `SavePolicy` 成功下发，已有未知权限不会被静默删除
- 前端测试与构建通过

#### Risks
- 前端内置权限目录与未来服务端新增权限点可能不同步，需要显式保留未知权限
- `AccessPolicy.vue` 当前已较大，tab 化后若不做适度抽象会继续膨胀
- 角色选择改为更强约束后，必须兼容已有“无显式 role_perms 但被 default/node override 引用”的配置

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 保持现有 `PermissionService` / `useAccessPolicyStore` 契约不变，在前端完成访问策略页重构：
  - 用 `PageHero + tab` 组织 `Access Policy` 页面
  - 新增前端权限目录 helper，提供分组权限项、排序和未知权限保留能力
  - 把角色权限编辑从当前策略 tab 拆到角色管理 tab
  - 默认权限和角色权限统一改为 checkbox catalog 编辑，再序列化回现有 `Policy`

#### Alternatives Considered
- 方案 A（采用）：前端内置权限目录 + 未知权限保留 + 现有后端接口不变
  - 优点：改动面小，不需要 server / Wails 新接口
  - 代价：catalog 需要靠当前 specs 维护
- 方案 B：新增后端“权限目录元数据”接口，由 authority 动态返回可用权限项
  - 优点：理论上更权威
  - 代价：协议、服务、bindings、测试面都会扩大，不符合本轮最小安全改动
- 方案 C：保留 CSV 编辑，仅在 UI 上做 tab 拆分
  - 优点：实现更快
  - 代价：不能满足“不要让用户自己编辑权限”的目标

#### Module Responsibilities
- `frontend/src/pages/AccessPolicy.vue`
  - 负责 tab 布局、表单状态映射、保存/加载动作、运行时展示、节点权限查询
- `frontend/src/stores/accessPolicy.ts`
  - 继续负责 authority policy 的加载、保存和 `getNodePerms`
- `frontend/src/stores/accessPolicyCatalog.ts`（新增）
  - 负责预置权限目录、权限分组、未知权限提取、角色/默认权限序列化辅助
- `frontend/src/i18n/messages/operations.ts`
  - 负责访问策略新文案、tab 文案、目录化权限编辑文案
- `frontend/src/i18n/messages/shell.ts`
  - 负责左侧导航中文命名调整
- `frontend/src/router/index.ts`
  - 负责页面副标题文案同步
- `docs/requirements/authority-admin-console.md`
  - 更新稳定需求中的中文命名、tab 与角色管理 UX 要求
- `docs/specs/authority-admin-console.md`
  - 更新页面职责边界、tab 结构和 permission catalog 兼容策略

#### Data / Call Flow
1. 页面进入后按现有逻辑解析 authority 并加载 `Policy`
2. 加载结果映射为结构化 UI 状态：
  - 当前策略 tab：default role、default perms 勾选、node overrides
  - 角色管理 tab：role list、每个角色的已选 catalog perms、未知 perms
3. 用户在 tab 内编辑：
  - 当前策略 tab 不再编辑角色权限明细
  - 角色管理 tab 通过 checkbox 更新角色权限
4. 保存时把结构化 UI 状态重新组装成原 `Policy`
5. `SavePolicy`、runtime verify、node perms query 的调用链保持不变

#### Interface Drafts
- frontend helper:
  - `PermissionCatalogGroup`
  - `PermissionCatalogItem`
  - `splitKnownAndUnknownPerms(perms)`
  - `mergeSelectedPerms(known, unknown)`
- page-local editable state:
  - `activeTab`
  - `policyForm.defaultRole`
  - `policyForm.defaultPerms`
  - `policyForm.nodeRoles`
  - `policyForm.rolePerms`
  - `roleDraftName`
- backend / Wails interface:
  - no contract change
  - continue using `LoadPolicy`, `SavePolicy`, `GetNodePerms`

#### Error Handling and Safety
- catalog 外权限在 UI 中单独标记为“额外权限（只读保留）”
- `*` 作为特殊权限单独处理，避免与普通权限重复/冲突
- 角色名、node ID、重复项等继续在前端本地校验后再保存
- tab 切换不应清空已编辑状态
- 保存失败时保留当前 UI 状态，不因一次失败丢失用户编辑

#### Performance and Testing Strategy
- 权限目录为前端静态数据，不增加运行时请求
- 角色选项、摘要、unknown perms 通过 computed 推导，避免重复解析字符串
- 自动化验证:
  - `frontend/src/stores/authority_admin.test.ts`
  - 新增 `frontend/src/pages/AccessPolicy.test.ts`
  - `npm test`
  - `npm run build`

#### Extensibility Design Points
- 后续若 server 新增权限点，可只扩 catalog，不改保存协议
- 若未来真的有权限元数据接口，catalog helper 可以替换为远端数据源，而页面状态模型不必重做
- unknown perms 保留策略保证旧配置和未来配置在 catalog 未更新前仍可安全 round-trip

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 把 `Access Policy` 做成更清晰的“访问策略”页面，增加 tab 化组织与角色管理 tab，并把权限编辑改为目录勾选
- Current State:
  - 中文入口仍显示“权限编排”
  - `AccessPolicy.vue` 是单页长表单，`default_perms` / `role_perms` 仍依赖 CSV 文本编辑
  - 角色权限编辑与当前策略总览混在同一页，可读性一般

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- Requirements impact: updated
- Specs impact: updated
- Stable docs destination:
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- Change archive destination:
  - `docs/change/2026-03-26_win-access-policy-tabs.md`
- Lessons impact:
  - none（当前未发现需要沉淀为通用 lesson 的排障知识）

#### Related Requirements / Specs / Lessons
- Requirements:
  - `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs\docs\requirements\authority-admin-console.md`
- Specs:
  - `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs\docs\specs\authority-admin-console.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\file.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
- Lessons:
  - none

#### Executable Task List
- [ ] DOCS-AP-1 更新 authority admin console 的 requirements/specs
- [ ] IMPL-AP-1 重构访问策略页命名、tab 与当前策略主页布局
- [ ] IMPL-AP-2 实现角色管理 tab 与目录化权限编辑
- [ ] TEST-AP-1 补前端回归测试并完成构建验证
- [ ] REVIEW-AP-1 完成 3.3 代码复核
- [ ] ARCHIVE-AP-1 归档到 `docs/change`

#### Task Details
##### DOCS-AP-1 - 稳定文档更新
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs`
- Plan Path: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs\plan.md`
- Goal:
  - 将“访问策略”命名、tab 布局、角色管理和目录化权限编辑写入稳定 requirements/specs
- Files / Modules:
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- Write Set:
  - 同上
- Acceptance:
  - 稳定文档描述当前策略 tab / 角色管理 tab / unknown perms 保留边界
- Test Points:
  - 文档自检，确认 requirements/specs 与实现边界一致
- Rollback:
  - 回退上述文档修改

##### IMPL-AP-1 - 访问策略命名与页面结构重构
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs`
- Plan Path: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs\plan.md`
- Goal:
  - 将中文“权限编排”改为“访问策略”，并把 `Access Policy` 改为带 tab 的页面
- Files / Modules:
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/router/index.ts`
  - `frontend/src/i18n/messages/operations.ts`
  - `frontend/src/i18n/messages/shell.ts`
- Write Set:
  - 同上
- Acceptance:
  - 页面默认进入当前策略 tab，文案命名统一为“访问策略”
- Test Points:
  - `npm test`
  - `npm run build`
- Rollback:
  - 回退页面布局和 i18n / route 文案

##### IMPL-AP-2 - 角色管理与目录化权限编辑
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs`
- Plan Path: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs\plan.md`
- Goal:
  - 用目录勾选替代手填权限，把角色编辑抽到单独 tab，并保留未知权限
- Files / Modules:
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/stores/accessPolicy.ts`
  - `frontend/src/stores/accessPolicyCatalog.ts`
- Write Set:
  - 同上
- Acceptance:
  - 默认权限与角色权限不再依赖 CSV 文本框
  - 保存 round-trip 后不会静默丢失未知权限
- Test Points:
  - `npm test`
  - `npm run build`
- Rollback:
  - 回退 catalog helper 和页面权限编辑逻辑

##### TEST-AP-1 - 前端回归验证
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs`
- Plan Path: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs\plan.md`
- Goal:
  - 锁定访问策略页 tab、目录化权限编辑和未知权限保留行为
- Files / Modules:
  - `frontend/src/stores/authority_admin.test.ts`
  - `frontend/src/pages/AccessPolicy.test.ts`
- Write Set:
  - 同上
- Acceptance:
  - 自动化测试覆盖 tab 切换和关键保存语义
- Test Points:
  - `npm test`
  - `npm run build`
- Rollback:
  - 回退新增测试

#### Dependencies
- `AccessPolicy.vue` 与 i18n 文案、route meta 紧耦合
- 权限目录需要以 `repo/MyFlowHub-Server/docs/specs/*.md` 中已有权限点为基线
- unknown perms 保留逻辑依赖现有 `Policy` round-trip 序列化保持稳定

#### Risks and Notes
- 本轮不新增后端接口，因此 catalog 的完整性依赖文档与既有默认角色定义
- 如果当前真实配置包含大量 catalog 外权限，UI 会出现较多“额外权限（只读保留）”提示，但这比误删更安全
- 当前 worktree 根 `plan.md` 原本是已结束 workflow 的残留，已被当前 workflow 控制文档替换

#### Parallelism Assessment
- 不派发子Agent
- 原因:
  - 页面结构、权限编辑模型、i18n 和稳定 docs 写集紧耦合
  - 当前会话未获得显式子Agent委派授权
- Owner:
  - 主Agent

#### Issue List
- none

阻塞：否
进入 3.2

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 中文“访问策略”命名、tab 化结构、角色管理抽离、目录化权限编辑、unknown perms 保留、稳定 docs 更新均已落实。
- 架构合理性：通过
  - 继续复用现有 `PermissionService` / `useAccessPolicyStore` 契约，把改动收敛在前端页面、helper 和文案层。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 权限目录为前端静态数据；tab 切换不发请求；仅在显式 load/save/query 时访问 authority。
- 可读性与一致性：通过
  - 页面职责按“当前策略 / 角色管理”拆开，权限编辑从 CSV 改为分组勾选，导航/route/i18n 命名一致。
- 可扩展性与配置化：通过
  - 新增权限目录 helper，并对 catalog 外权限做保留；后续扩权限点时只需补 catalog。
- 稳定性与安全：通过
  - unknown perms 默认保留而不是丢弃；`*` 在编辑态做独占归一化，避免歧义。
- 测试覆盖情况：通过
  - `npm test`
  - `$env:GOWORK='off'; wails generate module`
  - `npm run build`
  - `git diff --check`
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子Agent。

### Stage 4 - Change Archive
- 使用 `$m-docs` 校验 requirements/specs/change 路由和索引更新。
- Change:
  - `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs\docs\change\2026-03-26_win-access-policy-tabs.md`
- Index updated:
  - `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs\docs\change\README.md`
- Lessons impact:
  - none

是否结束 workflow：待用户确认
