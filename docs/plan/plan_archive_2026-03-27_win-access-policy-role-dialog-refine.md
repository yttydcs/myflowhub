# Plan - Win Access Policy Role Dialog Refine

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-access-policy-role-dialog-refine`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine`
- Current Stage: `4 archive complete`

## Stage Records

### Initialization
- `guide.md`:
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`
- base/worktree confirmation:
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `fix/win-access-policy-role-dialog-refine`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine`
  - implementation will stay inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 继续收敛 `Access Policy` 的角色编辑体验，重点修复角色弹窗的可见性、可操作性和列表密度问题，使角色编辑保持单列、紧凑、易扫读。

#### Scope
- 必须:
  - 保留 `Access Policy` 的双 tab 结构，不改动 `Current Policy / Role Management` 的页面职责
  - 角色编辑弹窗必须保留在 `Overlay` 中，但需要补足内边距，避免输入框或按钮的 focus ring 被裁切
  - 角色编辑弹窗主体必须改为单列布局，不再左右分栏
  - 角色编辑弹窗中的“添加权限”必须改为显式选择流程：
    - 点击 `Add Permission`
    - 打开权限选择列表 / 选择弹窗
    - 选择一个 catalog 权限后再追加到当前角色
  - 角色权限列表不得继续支持直接改写现有权限项；现有权限项只允许查看和删除
  - `Apply Built-in Preset` 必须恢复为可感知、可用的动作，点击后要明确更新当前角色的权限列表
  - 角色列表和角色弹窗内的权限列表都需要改成参考 `Flow` 页面列表的一行式紧凑布局
  - 目录外权限 `unknown perms` 仍需保留展示和保存
  - 不改变现有 `Policy` 序列化契约和 authority 请求链路
- 可选:
  - 在权限选择列表里补充简短说明，帮助区分权限分组
  - 对 `*` 权限保留显式提示，避免用户误以为还可以继续叠加普通权限
- 不做:
  - 不修改 `Current Policy` tab 的默认准入编辑方式，除非本轮实现证明必须同步调整
  - 不修改 `MyFlowHub-Server` 协议或 Wails API
  - 不新增权限自由输入入口

#### Use Cases
- 管理员编辑角色时，输入框获得焦点后，外层 focus ring 能完整显示，不会贴边裁切
- 管理员打开角色弹窗时，能按单列顺序依次看到角色名、预设动作、权限列表和保留的额外权限，不需要左右来回扫读
- 管理员想新增权限时，通过显式选择列表挑一个 catalog 权限加入，而不是先创建一行再手动改下拉框
- 管理员想移除权限时，可以直接在权限行右侧点击删除
- 管理员在角色列表页浏览多个角色时，能够像 `Flow` 列表一样快速看完名称、摘要和操作，不被多行块状内容拉高页面

#### Functional Requirements
- 角色弹窗必须继续支持 role 名称编辑、内置预设套用、known perms 维护、unknown perms 保留、保存和取消
- 角色弹窗中的权限新增入口必须只从内置权限 catalog 选择，不允许手填
- 角色弹窗中的权限列表项必须至少展示：
  - 权限名
  - 可选的简短说明或分组信息
  - 删除动作
- 当权限列表中存在 `*` 时，新增权限流程必须正确处理独占语义，不能产生 `* + 普通权限` 的已知权限组合
- `Apply Built-in Preset` 点击后必须立即反映到当前弹窗状态
- 角色列表页必须继续展示角色引用信息和删除约束，但布局改为更紧凑的一行式摘要
- 角色删除时，若该角色仍被默认准入或节点覆盖引用，仍需阻止删除并提示
- 保存后的 payload 必须继续序列化回现有 `Policy`

#### Non-functional Requirements
- 继续以前端改造为主，避免扩大到新的后端接口
- 交互层级要比当前更轻，避免在弹窗中重新引入“大块表单”的阅读成本
- 弹窗滚动和内边距调整不得破坏现有圆角、焦点管理和遮罩关闭行为
- 角色列表与权限列表样式应尽量贴近现有 `Flow` 页面列表的密度和节奏，保持产品内一致性
- 保持中英文 i18n 一致

#### Inputs / Outputs
- 输入:
  - 当前角色名
  - 当前角色 known perms
  - 当前角色 unknown perms
  - 内置角色预设
  - 内置权限 catalog
- 输出:
  - 单列角色编辑弹窗
  - 通过选择列表新增的权限项
  - 紧凑角色列表
  - 保存后的 `Policy`

#### Edge Cases
- 角色名为空或包含非法分隔符
- 角色已有 `*` 权限
- 权限 catalog 已被全部选完
- 历史策略包含 catalog 外权限
- 当前角色名不是内置角色，不能套用预设
- 角色列表项被默认准入或节点覆盖引用

#### Acceptance Criteria
- 角色编辑弹窗中，获得焦点的输入控件外圈能完整显示，不再被裁切
- 角色编辑弹窗改为单列布局，内容可滚动但主操作区保持清晰
- `Add Permission` 改为打开选择列表后再新增；现有权限不再支持直接改写
- `Apply Built-in Preset` 点击后能明确更新列表
- 角色列表和权限列表都比当前更紧凑，整体接近 `Flow` 页面列表的单行节奏
- unknown perms 继续可见且保存时不丢失
- 前端测试与构建通过

#### Risks
- 角色弹窗中再嵌套一个权限选择 overlay 时，需要确认焦点栈和 Esc 行为仍稳定
- 若 `*` 权限处理不严谨，可能让“新增权限”与“套用预设”产生冲突语义
- 列表过度压缩可能导致权限说明信息不够直观，需要在密度和可读性之间保持平衡

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 保持现有 `useAccessPolicyStore` / `Policy` / `accessPolicyCatalog.ts` 契约不变，只重构 `AccessPolicy.vue` 的角色管理视图和角色编辑弹窗：
  - 角色列表改为更接近 `Flow.vue` 的单行摘要列表
  - 角色弹窗改为单列堆叠布局
  - 角色弹窗增加独立的权限选择 overlay 状态，用于 `Add Permission`
  - 现有权限项改为只读列表行 + 删除按钮，不再使用行内 `<select>` 改写
  - 通过滚动容器的内层 padding 解决 focus ring 裁切

#### Alternatives Considered
- 方案 A（采用）：角色弹窗单列化 + 嵌套权限选择 overlay + 紧凑权限行
  - 优点：最符合用户明确要求，交互路径清晰，现有权限不会误编辑
  - 代价：需要新增一层弹窗状态和测试覆盖
- 方案 B：保留现有列表行 `<select>`，只做样式收紧和 padding 修正
  - 优点：改动最小
  - 代价：不满足“不要能直接改，用添加方式添加”的要求
- 方案 C：改为下拉菜单或 popover 内联选择
  - 优点：比嵌套 overlay 轻
  - 代价：当前项目没有现成稳定的权限选择 popover 交互，焦点和滚动控制更容易分散

#### Module Responsibilities
- `frontend/src/pages/AccessPolicy.vue`
  - 负责角色列表压缩、角色弹窗单列布局、权限选择 overlay、预设套用和焦点安全边距
- `frontend/src/stores/accessPolicyCatalog.ts`
  - 继续提供权限目录、预设角色权限、known/unknown perms 拆分与 metadata 查询
- `frontend/src/i18n/messages/operations.ts`
  - 补充权限选择、紧凑列表和动作反馈相关文案
- `docs/requirements/authority-admin-console.md`
  - 记录角色弹窗“单列 + 选择式添加 + 只读权限行”的交互约束
- `docs/specs/authority-admin-console.md`
  - 记录角色编辑弹窗的局部状态模型和权限选择流程

#### Data / Call Flow
1. 页面加载 policy，继续映射为 `policyForm`
2. 打开角色弹窗时，把目标 role 写入 `roleEditorDialog`
3. 点击 `Add Permission` 打开权限选择 overlay，并根据当前已选权限过滤候选
4. 用户选择一个权限后，前端将其追加到 `roleEditorDialog.perms`
5. 点击 `Apply Built-in Preset` 时，用对应内置预设覆盖当前 known perms
6. 点击权限行右侧删除按钮时，从 `roleEditorDialog.perms` 中移除该权限
7. 点击保存后，继续通过现有 `submitRoleEditorDialog` 回写到 `policyForm.rolePerms`

#### Interface Drafts
- 保留:
  - `roleEditorDialog`
- 新增局部状态:
  - `rolePermissionPickerDialog`
    - `open`
    - `selectedPerm` 或同等候选态
- 权限列表项模型:
  - `perm`
  - `label`
  - `groupLabel`
  - `description`
- 角色列表行模型:
  - `role`
  - `known perm count`
  - `unknown perm count`
  - `references`
  - `edit/remove actions`

#### Error Handling and Safety
- 继续沿用角色名非空、非法分隔符、角色重名校验
- 权限选择 overlay 必须过滤已选项，并正确处理 `*` 独占语义
- 若已无可添加权限，`Add Permission` 必须显式禁用
- unknown perms 继续单独展示，不混入可选列表
- 嵌套 overlay 必须继续复用 overlay stack，保持 Esc 只关闭最上层

#### Performance and Testing Strategy
- 所有新增交互都是前端局部状态，不增加新的 authority 请求
- 验证重点:
  - `frontend/src/pages/AccessPolicy.test.ts`
  - `npm test -- AccessPolicy`
  - `npm run build`
- 若可行，补一次浏览器端界面 smoke check，确认角色弹窗滚动、内边距和选择弹窗交互

#### Extensibility Design Points
- 后续若默认准入弹窗也要改成同样的“选择式新增权限”，可以复用相同的 picker 状态与行组件模式
- 权限行的紧凑展示方式后续可下沉成共享列表项组件，但本轮先保持最小改动
- 保留 `accessPolicyCatalog.ts` 作为唯一权限元数据来源，避免 UI 侧散落硬编码

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 修复访问策略角色弹窗的焦点裁切、布局臃肿和权限编辑路径不清晰问题
- Current State:
  - 角色弹窗虽然已支持滚动，但滚动区边距不足，focus ring 仍可能被裁切
  - 角色弹窗仍是左右分栏，和用户期望的单列编辑路径不一致
  - 权限项仍通过行内 `<select>` 直接改写，不符合“只通过添加来新增”的目标
  - 角色列表与权限列表的密度仍高于 `Flow` 页面列表

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- Requirements impact: `clarify`
- Specs impact: `clarify`
- Stable docs destination:
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- Change archive destination:
  - `docs/change/2026-03-27_win-access-policy-role-dialog-refine.md`
- Lessons impact:
  - none（当前属于页面交互 refine，没有形成新的可复用排障模式）

#### Related Requirements / Specs / Lessons
- Requirements:
  - `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine\docs\requirements\authority-admin-console.md`
- Specs:
  - `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine\docs\specs\authority-admin-console.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- Lessons:
  - `D:\project\MyFlowHub3\docs\lessons\README.md`

#### Executable Task List
- [x] DOC-APR-1 更新 authority admin console 的 requirements/specs
- [x] IMPL-APR-1 收敛角色列表为 Flow 风格紧凑单行摘要
- [x] IMPL-APR-2 重构角色编辑弹窗为单列布局并修复 focus ring 裁切
- [x] IMPL-APR-3 改造角色权限编辑为“选择后新增 + 行内删除”
- [x] TEST-APR-1 更新前端测试并完成验证
- [x] REVIEW-APR-1 完成 3.3 代码复核
- [x] ARCHIVE-APR-1 归档到 `docs/change`

#### Task Details
##### DOC-APR-1 - 稳定文档更新
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine\plan.md`
- Goal:
  - 把角色弹窗单列化、选择式添加权限和紧凑列表约束写入 requirements/specs
- Files / Modules:
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- Acceptance:
  - 稳定文档不再把角色权限编辑描述为可直接改写的列表行
- Test Points:
  - 文档自检
- Rollback:
  - 回退文档修改

##### IMPL-APR-1 - 角色列表收敛
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine\plan.md`
- Goal:
  - 把角色列表和权限列表收敛为更接近 `Flow.vue` 的单行紧凑样式
- Files / Modules:
  - `frontend/src/pages/AccessPolicy.vue`
- Acceptance:
  - 角色列表和权限列表不再是高块状卡片，扫读路径更接近 `Flow` 列表
- Test Points:
  - `npm test -- AccessPolicy`
  - 浏览器 smoke check（如可行）
- Rollback:
  - 回退列表样式和布局

##### IMPL-APR-2 - 角色弹窗单列化与焦点安全边距
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine\plan.md`
- Goal:
  - 修复弹窗滚动区的 focus ring 裁切，并把弹窗改为单列布局
- Files / Modules:
  - `frontend/src/pages/AccessPolicy.vue`
- Acceptance:
  - 输入控件 focus ring 可完整显示
  - 弹窗主体改为单列
- Test Points:
  - `npm test -- AccessPolicy`
  - `npm run build`
- Rollback:
  - 回退弹窗布局和滚动容器结构

##### IMPL-APR-3 - 选择式权限新增与预设动作修复
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine\plan.md`
- Goal:
  - 让角色权限只能通过 picker 新增、通过按钮删除，并确保内置预设动作可用
- Files / Modules:
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/stores/accessPolicyCatalog.ts`
  - `frontend/src/i18n/messages/operations.ts`
- Acceptance:
  - 现有权限不再支持行内改写
  - `Add Permission` 通过选择列表追加
  - `Apply Built-in Preset` 点击后有明确效果
- Test Points:
  - `npm test -- AccessPolicy accessPolicyCatalog`
  - `npm run build`
- Rollback:
  - 回退 picker 状态和权限列表交互

##### TEST-APR-1 - 前端回归验证
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine\plan.md`
- Goal:
  - 锁定角色弹窗新结构、权限选择流程和列表密度回归
- Files / Modules:
  - `frontend/src/pages/AccessPolicy.test.ts`
- Acceptance:
  - 测试覆盖角色列表、角色弹窗滚动区、预设动作和权限 picker 关键路径
- Test Points:
  - `npm test -- AccessPolicy`
  - `npm run build`
- Rollback:
  - 回退新增测试

#### Dependencies
- `AccessPolicy.vue` 与 `accessPolicyCatalog.ts`、i18n 文案高度耦合
- 权限 picker 若复用 `Overlay`，会依赖现有 `overlayStack` 的多层覆盖行为
- 角色列表视觉收敛会参考 `frontend/src/pages/Flow.vue` 的列表节奏，但不抽成共享组件

#### Risks and Notes
- 嵌套 overlay 若处理不当，可能影响 Esc 和 backdrop 关闭顺序
- 若只压缩列表但不保留必要说明，用户可能难以判断权限分组与作用
- 当前 worktree 基于 `main@f5520d2` 创建，包含上一轮“角色弹窗可滚动”的基线提交

#### Parallelism Assessment
- 不派发子Agent
- 原因:
  - 角色列表、弹窗布局、权限选择和文案写集高度耦合
  - 当前会话未获得显式子Agent委派授权
- Owner:
  - 主Agent

#### Issue List
- none

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Completed Work
- `DOC-APR-1`
  - requirements/specs 已补充角色弹窗单列化、只读权限行和选择式新增权限约束
- `IMPL-APR-1`
  - 角色列表改成接近 `Flow` 页列表的一行式摘要
  - 角色行只保留名称、基线标识、摘要信息和编辑/移除动作
- `IMPL-APR-2`
  - 角色弹窗改成单列堆叠布局
  - 滚动容器增加内层 padding，避免 focus ring 被 overflow 裁切
  - 角色编辑主体宽度收敛到更紧凑的 `max-w-2xl`
- `IMPL-APR-3`
  - 角色权限列表改成只读行 + 删除按钮
  - `Add Permission` 改成打开独立权限选择 overlay
  - `Apply Built-in Preset` 改为显式覆盖当前 known perms，并补充成功反馈
  - 角色权限选择列表展示权限名、分组和说明，点击 `Select` 后直接追加

#### Files Changed
- `frontend/src/pages/AccessPolicy.vue`
- `frontend/src/pages/AccessPolicy.test.ts`
- `frontend/src/i18n/messages/operations.ts`
- `docs/requirements/authority-admin-console.md`
- `docs/specs/authority-admin-console.md`

### Stage 3.3 - Review
#### Review Summary
- 已对角色列表密度、角色弹窗单列结构、权限 picker 流程、`*` 独占语义和 unknown perms 保留路径完成自检。
- 代码复核结论：无新增阻塞性问题。

#### Review Checks
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过

### Stage 4 - Validation and Archive
#### Validation Results
- `npm test -- AccessPolicy`
  - 通过
- `GOWORK=off wails generate module`
  - 通过，用于 fresh worktree 补齐 `frontend/wailsjs/**`
- `npm run build`
  - 通过
- `chrome-devtools` browser smoke check
  - 未形成有效页面验证；Vite dev host 可访问，但 `/#/access-policy` 呈现空白根视图，未能据此做可信视觉确认

#### Archive Outputs
- change:
  - `docs/change/2026-03-27_win-access-policy-role-dialog-refine.md`
- docs index:
  - `docs/change/README.md`

阻塞：否
可继续下一轮 workflow / 可申请结束 workflow
