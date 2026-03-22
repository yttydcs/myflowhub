# Win 卡片头收敛 Workflow Plan

## 项目目标与当前状态

- 目标：遍历 `MyFlowHub-Win` 的页面与独立窗口，收敛卡片头里的“小标题 + 大标题 + 可选提示语”结构，只保留主标题和原有提示语。
- 当前状态：
  - 独占分支已创建：`fix/win-card-header-simplify`
  - 独占 worktree 已创建：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-card-header-simplify`
  - 已完成需求分析与架构设计，确认本轮覆盖 `frontend/src/pages/**` 与 `frontend/src/windows/**` 中命中的卡片头
  - 已确认边界：没有提示语的卡片不补；字段 `label`、列表项内部元信息、已是单行 window header 的区域不纳入本轮

## Workflow 信息

- 仓库：`MyFlowHub-Win`
- 分支：`fix/win-card-header-simplify`
- Base：`main`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-card-header-simplify`
- 当前阶段：`4 归档变更（已完成，等待用户确认是否结束 workflow）`
- 计划文档：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-card-header-simplify\plan.md`

## 文档治理与影响检查

- 使用 `$docs-governor` 的结论：
  - 本次 workflow 计划落点为 worktree 根 `plan.md`
  - 本次完成后的归档落点为 `docs/change/2026-03-22_win-card-header-simplify.md`
  - 当前仓已有受治理的 `docs/README.md`、`docs/plan/README.md`、`docs/change/README.md`，无需重建 docs 树
- Requirements impact：`none`
- Specs impact：`none`
- Related requirements：`none（当前仓无与“卡片头视觉收敛”对应的稳定 requirements 叶子文档）`
- Related specs：`none（不变更技术契约、接口或数据结构）`
- 计划归档路径：`docs/plan/plan_archive_2026-03-22_win-card-header-simplify.md`
- 变更归档路径：`docs/change/2026-03-22_win-card-header-simplify.md`
- Lessons 路径：`none（预期不产生复用性事故/教训文档）`

## 需求分析摘要

- 目标：
  - 将页面与窗口里的卡片级头部从“顶部小标题 + 主标题 + 可选提示语”收敛为“主标题 + 可选提示语”
  - 保持卡片右侧的 badge、tab、button、统计信息不变
- 范围：
  - 必须做：遍历命中的页面/窗口卡片头并统一收口
  - 必须做：覆盖 `frontend/src/pages/**` 与 `frontend/src/windows/**`
  - 不做：字段 label、内部列表项的细粒度元标签、已有单行 window 顶栏
- 当前命中文件：
  - `frontend/src/pages/Home.vue`
  - `frontend/src/pages/Devices.vue`
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/pages/LocalHub.vue`
  - `frontend/src/pages/Permissions.vue`
  - `frontend/src/pages/Presets.vue`
  - `frontend/src/pages/Settings.vue`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/pages/VarPool.vue`
  - `frontend/src/pages/File.vue`
  - `frontend/src/pages/Debug.vue`
  - `frontend/src/windows/FileTasks.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/windows/LogWindow.vue`
  - `frontend/src/windows/TopicBusWindow.vue`
- 验收标准：
  - 目标卡片头不再出现顶部小标题
  - 原主标题保留
  - 原提示语存在则保留，不存在则不新增
  - 右侧操作区和状态区不回归
  - 模板解析通过

## 架构设计摘要

- 采用方案：新增一个共享 `CardHeader` 组件，专门承接卡片级标题区；页面首屏头部继续由 `PageHero` 负责
- 选型理由：
  - `PageHero` 针对页面级头部，直接复用到卡片会让组件语义过重
  - `CardHeader` 更适合承接 `title / description / actions` 的卡片级结构
  - 组件化能降低 17 个文件中重复模板的维护成本
- 备选方案：
  - 逐文件直接删除 `<p>` 小标题
  - 不采用原因：范围大、重复模板多，后续难以继续维护一致性

## 可执行任务清单

- [x] `T1` 新增共享卡片头组件
- [x] `T2` 收敛页面类卡片头
- [x] `T3` 收敛窗口类卡片头
- [x] `T4` 验证并执行 Code Review
- [x] `T5` 归档变更并更新索引

## 任务详情

### `T1` 新增共享卡片头组件

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-card-header-simplify`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-card-header-simplify\plan.md`
- 目标：新增 `CardHeader.vue`，统一渲染卡片级主标题、可选提示语与右侧操作区
- 涉及模块 / 文件：
  - `frontend/src/components/CardHeader.vue`
- Write set：
  - `frontend/src/components/CardHeader.vue`
- 关键上下文引用：
  - `frontend/src/components/PageHero.vue`
  - `frontend/src/pages/Home.vue`
  - `frontend/src/pages/VarPool.vue`
  - `frontend/src/windows/TopicBusWindow.vue`
- 依赖：无
- 验收条件：
  - 可渲染主标题
  - 可选渲染提示语
  - 支持 `actions` slot
  - 不耦合任何 store、route 或业务状态
- 测试点：
  - Vue SFC 解析
- 回滚点：
  - 删除 `CardHeader.vue`
- 风险与注意事项：
  - 只统一卡片头部，不把卡片外层容器也一并抽成通用组件，避免改动过大

### `T2` 收敛页面类卡片头

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-card-header-simplify`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-card-header-simplify\plan.md`
- 目标：将页面中的命中卡片头替换为 `CardHeader` 或等价统一结构
- 涉及模块 / 文件：
  - `frontend/src/pages/Home.vue`
  - `frontend/src/pages/Devices.vue`
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/pages/LocalHub.vue`
  - `frontend/src/pages/Permissions.vue`
  - `frontend/src/pages/Presets.vue`
  - `frontend/src/pages/Settings.vue`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/pages/VarPool.vue`
  - `frontend/src/pages/File.vue`
  - `frontend/src/pages/Debug.vue`
- Write set：
  - 上述页面文件
- 关键上下文引用：
  - `frontend/src/components/PageHero.vue`
  - `frontend/src/router/index.ts`
- 依赖：
  - `T1`
- 验收条件：
  - 页面卡片头顶部小标题被移除
  - 原有主标题和提示语保留
  - 原 badge、button、tab 仍在头部区域可用
- 测试点：
  - 代码核对目标片段不再包含对应小标题 `<p>`
  - Vue SFC 解析
- 回滚点：
  - 回退对应页面模板改动
- 风险与注意事项：
  - 不误改表单字段 `label`
  - 不误改列表项内部的统计元信息

### `T3` 收敛窗口类卡片头

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-card-header-simplify`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-card-header-simplify\plan.md`
- 目标：将独立窗口中命中的业务卡片头统一收口
- 涉及模块 / 文件：
  - `frontend/src/windows/FileTasks.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/windows/LogWindow.vue`
  - `frontend/src/windows/TopicBusWindow.vue`
- Write set：
  - 上述窗口文件
- 关键上下文引用：
  - `frontend/src/components/CardHeader.vue`
  - `frontend/src/windows/ShowcaseWindow.vue`（对照已收敛的单行 window 顶栏，确认不在本轮范围）
- 依赖：
  - `T1`
- 验收条件：
  - 命中的窗口卡片头不再出现顶部小标题
  - 原标题、提示语、操作按钮/统计信息保持
- 测试点：
  - 代码核对目标片段不再包含对应小标题 `<p>`
  - Vue SFC 解析
- 回滚点：
  - 回退对应窗口模板改动
- 风险与注意事项：
  - `FlowEditorWindow` 里部分区域是表单抽屉和方法选择面板，需只改卡片头，不改字段标签

### `T4` 验证并执行 Code Review

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-card-header-simplify`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-card-header-simplify\plan.md`
- 目标：完成静态验证、构建验证与强制 Code Review
- 涉及模块 / 文件：
  - `frontend/**`
  - `plan.md`
- Write set：
  - 无新增实现写集，仅允许补充必要的文档状态
- 依赖：
  - `T1`
  - `T2`
  - `T3`
- 验收条件：
  - 目标文件模板可解析
  - `npm run build` 如失败，需明确是否为既有环境问题
  - Code Review 全项给出通过/不通过结论
- 测试点：
  - `npm ci`
  - `npm run build`
  - `@vue/compiler-sfc` 解析改动文件
- 回滚点：
  - 按任务回退对应模板改动
- 风险与注意事项：
  - 当前仓已知存在 `wailsjs` 生成物缺失风险，若复现，需显式记录为残余风险

### `T5` 归档变更并更新索引

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-card-header-simplify`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-card-header-simplify\plan.md`
- 目标：完成 `docs/change` 归档并更新本仓 docs 索引
- 涉及模块 / 文件：
  - `docs/change/2026-03-22_win-card-header-simplify.md`
  - `docs/change/README.md`
- Write set：
  - 上述文档文件
- 依赖：
  - `T4`
- 验收条件：
  - 存在完整 change 文档
  - 已记录 requirement/spec impact
  - `docs/change/README.md` 能导航到本次归档
- 测试点：
  - 人工核对归档字段与索引链接
- 回滚点：
  - 删除本次 change 文档并回退索引更新
- 风险与注意事项：
  - 归档前必须再次显式执行 `$docs-governor` 检查

## 并行性评估

- 结论：本轮不使用子Agent
- 原因：
  - 目标页面和窗口会共享同一个 `CardHeader` 组件，文件写集与集成顺序高度耦合
  - 当前执行策略要求获得显式用户授权后才可委派子Agent，本轮未获得该授权

## 回归与验证步骤

1. 在 `frontend/` 目录执行 `npm ci`
2. 执行 `npm run build`
3. 使用 `@vue/compiler-sfc` 解析所有改动过的 Vue 文件
4. 静态核对目标卡片头已无顶部小标题
5. 完成强制 Code Review 后进入 `4`

## 执行结果

- 文本扫描：
  - `rg -n "text-xs font-semibold uppercase tracking-\[0\.3em\]" frontend/src/pages frontend/src/windows`
  - 结果：无命中
- 模式复核：
  - `rg -U -n '<p class="text-xs font-semibold uppercase tracking-\[0\.[23]em\] text-muted-foreground">[\s\S]{0,200}<h[123]' frontend/src/pages frontend/src/windows`
  - 结果：无命中
- Vue SFC 解析：
  - 使用 `@vue/compiler-sfc` 解析 `CardHeader.vue` 与所有变更文件
  - 结果：通过
- `npm ci`
  - 结果：通过
- `npm run build`
  - 结果：失败
  - 失败原因：`src/pages/Home.vue` 解析 `../../wailsjs/go/main/App` 失败，属于既有 `wailsjs` 生成绑定缺失问题，与本轮卡片头收敛改动无关

## Code Review 结论

- 需求覆盖：通过
- 架构合理性：通过
- 性能风险：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：静态验证通过，完整构建受既有环境问题阻塞
- 子Agent治理与审计：通过（本轮未使用子Agent）
