# Plan - Win Access Policy Dialog Editors

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `refactor/win-access-policy-dialog-editor`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-dialog-editor`
- Current Stage: `3.1 planning`

## Stage Records

### Initialization
- `guide.md`:
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`
  - 已读取 `frontend-design` 技能说明，用于收敛访问策略页面的交互层级，但保持现有产品视觉语言
- base/worktree confirmation:
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `refactor/win-access-policy-dialog-editor`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-dialog-editor`
  - implementation will stay inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 将当前过于臃肿的 `Access Policy` 页面收敛为“列表摘要 + 弹窗编辑”交互，尤其把默认准入和角色权限从整页大块表单改成更轻量的列表入口，进入弹窗后再做详细编辑。

#### Scope
- 必须:
  - 保留 `Access Policy` 的双 tab 结构：`Current Policy` / `Role Management`
  - `Current Policy` tab 中的默认准入改为紧凑摘要行或摘要卡，提供 `Edit` 动作打开弹窗
  - 默认准入弹窗中必须使用“权限列表 + Add / Remove”交互，不再在页面上直接展示整块 catalog 勾选区
  - `Role Management` tab 改为角色列表，每个角色至少展示：
    - role 名称
    - 权限数量摘要
    - 额外权限数量摘要
    - 编辑动作
    - 删除动作
  - 角色详细编辑必须通过弹窗进行，不再在 tab 内直接展开整块权限 catalog
  - 角色弹窗中的权限编辑必须改为“权限列表 + Add / Remove”交互
  - 默认准入和角色编辑都必须继续基于权限 catalog 提供选择，不允许用户自由输入任意权限字符串
  - 已有策略中的 unknown perms 仍需显式保留，不能因为 UI 收敛而丢失
  - `save / runtime snapshot / node perms lookup` 能力继续保留
  - 稳定 requirements/specs 需要同步记录新的交互模式
- 可选:
  - 角色列表和默认准入摘要中增加更直观的权限数量/风险提示
  - 角色弹窗中为 `superadmin/admin/node` 提供一键套用预设
- 不做:
  - 不修改 `MyFlowHub-Server` 协议或 Wails API
  - 不把 node overrides 也全面重构为独立弹窗体系，除非实现中证明当前列表仍明显阻塞主要目标
  - 不改变 `Policy` 存储契约

#### Use Cases
- 管理员打开访问策略页时，先看到清晰的默认准入摘要和角色列表，而不是整页密集表单
- 管理员想调整某个角色时，点击 `Edit` 进入弹窗，只关注当前角色，而不是同时看到整页所有角色的权限配置
- 管理员想增加或删除某个权限时，直接在弹窗内操作权限列表项，不需要在大块 catalog 里找勾选状态
- 管理员加载带 unknown perms 的历史策略时，页面仍能保留并提示这些权限

#### Functional Requirements
- 页面必须继续自动解析 authority，并在顶部显示当前身份与 authority
- 页面首屏不应继续保留大块与核心编辑无关的辅助说明或重复动作
- 默认准入区域必须只展示摘要信息和单一编辑入口
- 默认准入弹窗至少必须支持：
  - default role 选择
  - default perms 列表编辑
  - add permission
  - remove permission
  - unknown perms 保留展示
- 角色管理区域必须展示角色列表，而不是把所有角色详情直接铺开
- 角色编辑弹窗至少必须支持：
  - role 名称编辑
  - permission list 编辑
  - add permission
  - remove permission
  - unknown perms 保留展示
  - 内置角色预设套用
- 默认准入和角色权限选择项必须来自预置权限 catalog
- 保存后的 payload 必须继续序列化回现有 `Policy`
- 重复权限、空角色名、非法分隔符、非法 node ID 等错误仍需显式校验

#### Non-functional Requirements
- 继续以前端改造为主，避免扩大到新的后端接口
- 页面层级必须明显收敛，降低首次阅读成本
- 不增加额外 authority 请求；弹窗开关只影响前端状态
- 弹窗交互必须兼容键盘关闭、遮罩关闭和焦点恢复
- 保持中英文 i18n 一致

#### Inputs / Outputs
- 输入:
  - 当前登录身份 `sourceId / hubId`
  - default role
  - default perms 列表
  - role list
  - 单个 role 的 perms 列表
  - unknown perms
  - node role overrides
  - save options
  - node perms query `nodeId`
- 输出:
  - 访问策略摘要列表
  - 默认准入编辑弹窗状态
  - 角色编辑弹窗状态
  - 保存后的 `Policy` / warnings / runtime snapshot
  - 单节点最终角色与权限

#### Edge Cases
- authority 无法解析
- default role 指向未显式定义的 role
- perms 列表为空、重复或包含 `*`
- 策略中包含 catalog 外权限
- 删除某个角色后，default role 或 node override 仍引用该角色
- 弹窗关闭后必须保持未保存编辑状态的预期边界清晰

#### Acceptance Criteria
- 访问策略页相比当前版本显著收敛，不再把默认准入和所有角色权限整页铺开
- 角色区改为列表，每项通过编辑按钮进入弹窗
- 默认准入区改为摘要 + 弹窗编辑
- 弹窗中的权限编辑使用列表项的 add / remove，而不是整块勾选矩阵
- 保存、运行时预览和节点权限查询继续可用
- unknown perms 继续可见且可保留
- 前端测试与构建通过

#### Risks
- 从 checkbox grid 改成权限列表后，若选择器设计不清晰，可能降低批量编辑效率
- 弹窗内状态需要和页面主状态正确同步，否则容易出现关闭/保存后的脏数据问题
- 删除角色时若未处理引用关系，可能让默认准入或节点覆盖处于悬空状态

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 保持现有 `useAccessPolicyStore` / `Policy` 契约不变，只重构前端页面交互：
  - `Current Policy` tab 收敛为摘要区域、node overrides、save/runtime/query
  - `Role Management` tab 收敛为角色列表
  - 默认准入和角色详情编辑移入 `Overlay` 弹窗
  - 权限编辑从“整块 catalog 勾选矩阵”改为“按行选择的权限列表”
- 继续复用 `accessPolicyCatalog.ts` 作为权限来源和 unknown perms 保护层，不新增后端元数据接口

#### Alternatives Considered
- 方案 A（采用）：列表摘要 + 弹窗 + 权限列表编辑
  - 优点：显著降低页面密度，契合用户“列表 + 编辑”的操作路径
  - 代价：需要新增弹窗状态和行级编辑辅助函数
- 方案 B：保留当前 tab 结构，只压缩样式和折叠面板
  - 优点：实现更快
  - 代价：本质上仍是大表单，不能真正解决“太臃肿”
- 方案 C：继续用 checkbox grid，但放入弹窗
  - 优点：逻辑改动较小
  - 代价：弹窗里仍然会很重，不符合“权限列表 + add/remove”目标

#### Module Responsibilities
- `frontend/src/pages/AccessPolicy.vue`
  - 负责访问策略页摘要布局、角色列表、默认准入摘要、弹窗状态、保存/加载动作
- `frontend/src/stores/accessPolicyCatalog.ts`
  - 继续负责权限 catalog、预设角色权限、known/unknown perms 拆分与回写
- `frontend/src/i18n/messages/operations.ts`
  - 补充新的列表摘要、弹窗和权限列表编辑文案
- `docs/requirements/authority-admin-console.md`
  - 将默认准入/角色编辑的列表+弹窗交互写入长期需求
- `docs/specs/authority-admin-console.md`
  - 将页面状态模型和弹窗编辑契约写入长期规格

#### Data / Call Flow
1. 页面进入后继续自动加载 policy
2. store policy 映射为页面本地表单状态
3. tab 页面只展示摘要与列表
4. 点击默认准入或某个角色的 `Edit` 打开弹窗，并把当前状态映射为弹窗编辑态
5. 弹窗内通过权限列表的 add / remove / select 更新局部状态
6. 弹窗确认后回写到页面主表单状态
7. 页面保存时仍统一组装成现有 `Policy` 并调用 `SavePolicy`

#### Interface Drafts
- 新增页面局部弹窗状态:
  - `defaultAccessDialog`
  - `roleEditorDialog`
- 权限列表行模型:
  - `selectedPerms: string[]`
  - `unknownPerms: string[]`
  - 行级 add/remove/select helper
- 列表摘要项:
  - 默认准入摘要
  - 角色摘要行（role / perms count / extra count）

#### Error Handling and Safety
- 弹窗确认前必须校验空 role、重复权限、非法分隔符
- unknown perms 继续在弹窗内显式展示为保留项
- `*` 仍按独占权限处理
- 删除角色时若它被默认准入或 node override 引用，需要给出显式处理策略而不是静默破坏
- 弹窗关闭时不自动保存到后端；只有页面主保存才真正提交

#### Performance and Testing Strategy
- 弹窗状态为纯前端局部状态，不增加网络请求
- catalog 选项继续来自静态 helper
- 自动化验证:
  - `frontend/src/pages/AccessPolicy.test.ts`
  - `frontend/src/stores/accessPolicyCatalog.test.ts`
  - `npm test -- AccessPolicy accessPolicyCatalog authority_admin`
  - `npm run build`

#### Extensibility Design Points
- 未来若增加更多权限点，只需扩 `accessPolicyCatalog.ts`，不必改变弹窗编辑模型
- 若未来 node overrides 也要改成弹窗列表，可复用相同的摘要列表 + overlay 模式
- 弹窗状态与页面主状态分离后，后续可抽成独立子组件而不改协议

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 把访问策略页从整页大表单收敛为摘要列表 + 弹窗编辑，重点优化默认准入和角色管理
- Current State:
  - 默认准入仍直接渲染整块权限 catalog
  - 角色管理页直接把每个角色详情铺开，页面高度和阅读成本过高
  - 默认准入和角色详情没有形成“先看列表，再进入编辑”的操作路径

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- Requirements impact: `clarify`
- Specs impact: `clarify`
- Stable docs destination:
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- Change archive destination:
  - `docs/change/2026-03-27_win-access-policy-dialog-editors.md`
- Lessons impact:
  - none（当前没有新增可复用排障知识，属于交互收敛）

#### Related Requirements / Specs / Lessons
- Requirements:
  - `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-dialog-editor\docs\requirements\authority-admin-console.md`
- Specs:
  - `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-dialog-editor\docs\specs\authority-admin-console.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- Lessons:
  - `D:\project\MyFlowHub3\docs\lessons\README.md`

#### Executable Task List
- [x] DOC-APD-1 更新 authority admin console 的 requirements/specs
- [x] IMPL-APD-1 收敛访问策略页面主视图为摘要列表
- [x] IMPL-APD-2 实现默认准入与角色编辑弹窗
- [x] TEST-APD-1 更新前端测试并完成构建验证
- [x] REVIEW-APD-1 完成 3.3 代码复核
- [x] ARCHIVE-APD-1 归档到 `docs/change`

#### Task Details
##### DOC-APD-1 - 稳定文档更新
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-dialog-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-dialog-editor\plan.md`
- Goal:
  - 将“列表 + 弹窗编辑”的交互边界写入 requirements/specs
- Files / Modules:
  - `docs/requirements/authority-admin-console.md`
  - `docs/specs/authority-admin-console.md`
- Acceptance:
  - 稳定文档不再把当前实现描述为整页直接编辑
- Test Points:
  - 文档自检
- Rollback:
  - 回退文档修改

##### IMPL-APD-1 - 主视图收敛
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-dialog-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-dialog-editor\plan.md`
- Goal:
  - 把默认准入和角色管理从大块表单改为摘要列表
- Files / Modules:
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/i18n/messages/operations.ts`
- Acceptance:
  - 页面主视图明显收敛，角色管理变为列表
- Test Points:
  - `npm test -- AccessPolicy`
  - `npm run build`
- Rollback:
  - 回退页面布局和文案

##### IMPL-APD-2 - 弹窗编辑与权限列表
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-dialog-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-dialog-editor\plan.md`
- Goal:
  - 通过弹窗完成默认准入和角色详情编辑，并将权限编辑改为 add/remove 列表
- Files / Modules:
  - `frontend/src/pages/AccessPolicy.vue`
  - `frontend/src/stores/accessPolicyCatalog.ts`
  - `frontend/src/components/ui/overlay/Overlay.vue`
- Acceptance:
  - 默认准入与角色编辑都通过弹窗完成
  - 权限编辑不再使用整块 checkbox grid
  - unknown perms 继续保留
- Test Points:
  - `npm test -- AccessPolicy accessPolicyCatalog`
  - `npm run build`
- Rollback:
  - 回退弹窗状态与权限列表逻辑

##### TEST-APD-1 - 前端回归验证
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-dialog-editor`
- Plan Path: `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-dialog-editor\plan.md`
- Goal:
  - 锁定访问策略页的新交互结构和保存语义
- Files / Modules:
  - `frontend/src/pages/AccessPolicy.test.ts`
  - `frontend/src/stores/accessPolicyCatalog.test.ts`
- Acceptance:
  - 测试覆盖默认准入/角色列表/弹窗编辑关键路径
- Test Points:
  - `npm test -- AccessPolicy accessPolicyCatalog authority_admin`
  - `npm run build`
- Rollback:
  - 回退新增测试

#### Dependencies
- `AccessPolicy.vue` 与 `accessPolicyCatalog.ts`、i18n 文案高度耦合
- 弹窗交互依赖现有 `Overlay` 组件的焦点恢复和关闭机制
- unknown perms 保留逻辑依赖现有 `Policy` round-trip 序列化保持稳定

#### Risks and Notes
- 若角色删除策略处理不清晰，容易引入悬空引用
- 权限列表改为逐行选择后，需要明确去重策略，避免产生重复权限项
- 当前 worktree 根 `plan.md` 是从已结束 workflow 拷贝出来的旧文件，已被本轮 workflow 控制文档替换

#### Parallelism Assessment
- 不派发子Agent
- 原因:
  - 页面布局、弹窗状态、文案和稳定 docs 写集高度耦合
  - 当前会话未获得显式子Agent委派授权
- Owner:
  - 主Agent

#### Issue List
- none

### Stage 3.2 - Implementation
#### Completed Work
- `IMPL-APD-1`
  - `AccessPolicy.vue` 主视图收敛为摘要列表布局
  - 默认准入改成摘要卡 + 编辑入口
  - 节点覆盖改成紧凑列表，每行只保留摘要、编辑和移除动作
  - 角色管理改成紧凑列表，每行只保留摘要、编辑和移除动作
  - 保存、运行时和节点查询合并为单一操作面板
- `IMPL-APD-2`
  - 默认准入编辑迁移到 `Overlay`
  - 节点覆盖新建/编辑迁移到 `Overlay`
  - 角色新增/编辑迁移到 `Overlay`
  - 权限编辑改为目录驱动的逐行 `Add / Remove` 列表
  - `accessPolicyCatalog.ts` 新增 flattened options 和 metadata lookup helper
  - unknown perms 继续只读保留
  - 删除仍被默认准入或节点覆盖引用的角色时显式阻止
  - 角色重命名时同步更新默认角色和节点覆盖引用

#### Files Changed
- `frontend/src/pages/AccessPolicy.vue`
- `frontend/src/stores/accessPolicyCatalog.ts`
- `frontend/src/i18n/messages/operations.ts`
- `frontend/src/pages/AccessPolicy.test.ts`
- `frontend/src/stores/accessPolicyCatalog.test.ts`

### Stage 3.3 - Review
#### Review Summary
- 已对改动后的页面状态、保存序列化路径、角色引用关系和 unknown perms 保留逻辑做自检。
- 代码复核结论：无新增阻塞性问题。

#### Review Checks
- 默认准入不再整页展开，详细编辑仅在弹窗中完成
- 节点覆盖不再整页展开，详细编辑仅在弹窗中完成
- 角色管理为列表视图，详情编辑仅在弹窗中完成
- 右侧保存/运行时/查询不再拆成多个大卡片，运行时详情默认折叠
- 权限选择源仍然仅来自内置 catalog
- unknown perms 在默认准入和角色编辑中均继续保留
- 删除被引用角色时会被阻止，避免悬空引用

### Stage 4 - Validation and Archive
#### Validation Results
- `npm test -- AccessPolicy accessPolicyCatalog authority_admin`
  - 通过
- `wails generate module`
  - 已执行，用于 fresh worktree 补齐 `frontend/wailsjs/**`
- `npm run build`
  - 通过

#### Archive Outputs
- change:
  - `docs/change/2026-03-27_win-access-policy-dialog-editors.md`
- docs index:
  - `docs/change/README.md`

阻塞：否
可结束 workflow / 可申请合并
