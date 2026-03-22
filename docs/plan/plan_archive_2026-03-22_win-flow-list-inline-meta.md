# Win Flow 列表同排与时间弱化 Workflow Plan

## 项目目标与当前状态

- 目标：
  - 调整 `Flow` 页本地项目列表，让 `name` 恢复主文本颜色
  - 让更新时间保持灰色弱化样式
  - 让左侧名称区与右侧按钮区保持更稳定的同排观感
- 当前状态：
  - 独占分支已创建：`fix/win-flow-list-inline-meta`
  - 独占 worktree 已创建：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-list-inline-meta`
  - 已完成需求分析与架构设计，确认本轮仍只修改 Win 前端 `Flow` 页本地项目列表
  - 已识别关键边界：继续隐藏 `Flow ID` / `Project ID`，不把 `name` 为空时的兜底值回退为 `flowId`

## Workflow 信息

- 仓库：`MyFlowHub-Win`
- 分支：`fix/win-flow-list-inline-meta`
- Base：`main`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-list-inline-meta`
- 当前阶段：`4 归档变更（已完成，等待用户确认是否结束 workflow）`
- 计划文档：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-list-inline-meta\plan.md`

## 文档治理与影响检查

- 使用 `$docs-governor` 的结论：
  - 当前仓 docs 树已受治理，无需补齐目录结构
  - 当前 workflow 的执行计划应维护在 worktree 根 `plan.md`
  - 当前 workflow 的完成归档应写入 `docs/change/2026-03-22_flow-list-inline-meta.md`
  - 归档完成后需要更新 `docs/change/README.md`
- Requirements impact：`none`
- Specs impact：`none`
- Related requirements：`none（当前仓无与该列表视觉微调对应的 requirements 叶子文档）`
- Related specs：`none（不变更接口、数据结构、调用契约或模块边界）`
- 变更归档路径：`docs/change/2026-03-22_flow-list-inline-meta.md`
- Lessons 路径：`none（当前预期不产生可复用事故教训）`

## 需求分析摘要

- 目标：
  - 让 `Flow` 本地项目列表的名称重新成为主视觉焦点
  - 让更新时间作为次级信息弱化展示
  - 让名称区和按钮区在正常宽度下同排显示
- 范围：
  - 必须做：名称使用主文本颜色
  - 必须做：时间使用灰色样式
  - 必须做：名称区与按钮区保持同排主布局
  - 必须做：继续不显示 `Flow ID` / `Project ID`
  - 不做：不改部署列表、不改元数据弹窗、不改 store、不改后端
- 验收标准：
  - 名称恢复为主文本样式
  - 时间为灰色弱化文本
  - 正常宽度下名称区与右侧按钮区同排，视觉更平衡
  - 名称为空时仍显示 `Untitled Project`
  - `Meta`、`Edit`、`Deploy`、`Delete` 行为不回归

## 架构设计摘要

- 采用方案：只在 [`frontend/src/pages/Flow.vue`](D:/project/MyFlowHub3/worktrees/MyFlowHub-Win-fix-flow-list-inline-meta/frontend/src/pages/Flow.vue) 拆分 `name` / `updatedAt` 节点并调整布局 class
- 选型理由：
  - 这是上轮列表收敛后的视觉修正，页面层调整即可
  - 布局与颜色都属于模板职责，不应扩散到 store
- 关键设计点：
  - 左侧名称区使用 `flex-1` / `min-w-0` 承担弹性空间，避免按钮区被挤压
  - 名称使用主文本样式，时间使用 `text-muted-foreground`
  - 继续使用 `Untitled Project` 作为空名称兜底
  - 不新增请求、不新增状态、不新增计算

## 可执行任务清单

- [x] `T1` 调整 Flow 列表项名称/时间样式与主行布局
- [x] `T2` 执行验证与 Code Review
- [x] `T3` 归档变更并更新索引

## 任务详情

### `T1` 调整 Flow 列表项名称/时间样式与主行布局

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-list-inline-meta`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-list-inline-meta\plan.md`
- 目标：
  - 拆分 `name` 与 `updatedAt` 的渲染节点
  - 让名称恢复主文本颜色，时间保持灰色
  - 优化左侧信息区与右侧按钮区的主行对齐
- 涉及模块 / 文件：
  - `frontend/src/pages/Flow.vue`
- Write set：
  - `frontend/src/pages/Flow.vue`
- 关键上下文引用：
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/i18n/messages/automation.ts`
- 依赖：无
- 验收条件：
  - 本地项目列表不再使用整行灰色文本承载名称与时间
  - 名称为主文本样式
  - 时间为灰色弱化样式
  - 左侧信息区和按钮区在正常宽度下同排
- 测试点：
  - 静态核对模板片段
  - Vue SFC 解析通过
- 回滚点：
  - 回退 `frontend/src/pages/Flow.vue`
- 风险与注意事项：
  - 保持窄宽度下的自然换行，不为了强制同排破坏响应式

### `T2` 执行验证与 Code Review

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-list-inline-meta`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-list-inline-meta\plan.md`
- 目标：
  - 完成最小验证并执行强制 Code Review
- 涉及模块 / 文件：
  - `frontend/src/pages/Flow.vue`
  - `plan.md`
- Write set：
  - 无新增实现写集；仅允许在必要时回填文档状态
- 关键上下文引用：
  - `frontend/src/pages/Flow.vue`
- 依赖：
  - `T1`
- 验收条件：
  - 目标模板可解析
  - 代码审查结论完整覆盖要求项
- 测试点：
  - 使用 `@vue/compiler-sfc` 解析 `frontend/src/pages/Flow.vue`
  - 文本检查确认不再把名称和时间包进同一个灰色段落
- 回滚点：
  - 回退 `T1` 的模板修改
- 风险与注意事项：
  - 当前任务为单文件 UI 微调，不做无关构建链路修复

### `T3` 归档变更并更新索引

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-list-inline-meta`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-list-inline-meta\plan.md`
- 目标：
  - 在 `docs/change/` 记录本次列表视觉修正，并更新归档索引
- 涉及模块 / 文件：
  - `docs/change/2026-03-22_flow-list-inline-meta.md`
  - `docs/change/README.md`
- Write set：
  - `docs/change/2026-03-22_flow-list-inline-meta.md`
  - `docs/change/README.md`
- 关键上下文引用：
  - `plan.md`
  - `docs/change/README.md`
- 依赖：
  - `T2`
- 验收条件：
  - 存在完整 change 文档
  - 已显式记录 requirement/spec impact
  - `docs/change/README.md` 可导航到本次归档
- 测试点：
  - 人工核对 change 文档字段完整性
  - 人工核对 README 索引链接有效
- 回滚点：
  - 删除本次归档文件并回退 README 更新
- 风险与注意事项：
  - 归档前必须再次显式执行 `$docs-governor` 检查

## 并行性预评估

- 结论：预计不使用子Agent
- 原因：
  - 当前实现写集只有 `frontend/src/pages/Flow.vue`，没有可安全并行拆分的独立实现任务
  - 3.2 阶段仍会再次执行一次正式并行性评估并记录结论

## 回归与验证步骤

1. 静态核对 `frontend/src/pages/Flow.vue` 的本地项目列表模板
2. 使用 `@vue/compiler-sfc` 解析目标 SFC
3. 如环境允许，使用浏览器查看列表主行观感
4. 完成 3.3 Code Review 后进入 4

## 执行结果

- `T1` 实现结果：
  - 已将名称与更新时间拆分为独立节点
  - 已让名称恢复主文本样式
  - 已让时间保持灰色弱化样式
  - 已通过 `flex-1` / `min-w-0` / `items-center` 优化左侧信息区与右侧按钮区的主行对齐
- `T2` 验证结果：
  - 文本核对通过：名称与时间不再共享同一个灰色段落
  - Vue SFC 解析通过：`frontend/src/pages/Flow.vue`
  - `npm run build` 未通过：失败原因是既有 `frontend/wailsjs` 生成物缺失，报错落在 `src/pages/Home.vue` 对 `../../wailsjs/go/main/App` 的解析，与本次改动无关

## Code Review 结论

- 需求覆盖：通过
  - 已覆盖“name 恢复主文本颜色”“时间保持灰色”“名称区与按钮区同排”“继续不显示 ID”
- 架构合理性：通过
  - 改动限定在页面模板层，未扩散到 store 或公共组件
- 性能风险：通过
  - 无新增请求、无重复计算、无额外 I/O；仅做模板节点与 class 调整
- 可读性与一致性：通过
  - 名称与时间语义拆分后更直观，颜色层级更符合页面现有设计语言
- 可扩展性与配置化：通过
  - 后续若要把时间改为相对时间或 badge，可直接在独立节点上扩展
- 稳定性与安全：通过
  - 不涉及接口、权限、持久化结构或敏感数据处理
- 测试覆盖情况：通过
  - 已完成目标文件语法校验与静态片段核对；完整前端构建受既有 `wailsjs` 缺失阻塞，已明确记录为环境残余问题
- 子Agent治理与审计：通过
  - 本轮未使用子Agent；原因已在计划文档中记录，未发生任务映射、文件所有权或集成冲突问题
