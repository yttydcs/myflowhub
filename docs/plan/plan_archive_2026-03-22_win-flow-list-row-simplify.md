# Win Flow 列表简化 Workflow Plan

## 项目目标与当前状态

- 目标：
  - 调整 `Flow` 页本地项目列表展示，不再在列表中显示 `Flow ID` 和 `Project ID`
  - 将项目名称与最后更新时间合并到同一行，并使用灰色信息样式展示
- 当前状态：
  - 独占分支已创建：`feat/flow-list-row-simplify`
  - 独占 worktree 已创建：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-flow-list-row-simplify`
  - 已完成需求分析与架构设计，确认本轮只改 Win 前端 `Flow` 页本地项目列表
  - 已识别关键边界：当前标题回退值是 `flowId`，若不处理会继续在列表中显示流程 ID

## Workflow 信息

- 仓库：`MyFlowHub-Win`
- 分支：`feat/flow-list-row-simplify`
- Base：`main`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-flow-list-row-simplify`
- 当前阶段：`4 归档变更（已完成，等待用户确认是否结束 workflow）`
- 计划文档：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-flow-list-row-simplify\plan.md`

## 文档治理与影响检查

- 使用 `$docs-governor` 的结论：
  - 当前仓 docs 树已受治理，无需补齐目录结构
  - 当前 workflow 的执行计划应维护在 worktree 根 `plan.md`
  - 当前 workflow 的完成归档应写入 `docs/change/2026-03-22_flow-list-row-simplify.md`
  - 归档完成后需要更新 `docs/change/README.md`
- Requirements impact：`none`
- Specs impact：`none`
- Related requirements：`none（当前仓无与该视觉收敛对应的 requirements 叶子文档）`
- Related specs：`none（不变更接口、数据结构、调用契约或模块边界）`
- 变更归档路径：`docs/change/2026-03-22_flow-list-row-simplify.md`
- Lessons 路径：`none（当前预期不产生可复用事故教训）`

## 需求分析摘要

- 目标：
  - 简化 `Flow` 页本地项目列表的信息密度
  - 保留项目名称和更新时间，去掉技术 ID
- 范围：
  - 必须做：移除 `Flow ID` 行
  - 必须做：移除 `Project ID` 行
  - 必须做：将 `name` 与 `updatedAt` 合并为同一行
  - 必须做：该行使用灰色文本风格
  - 不做：不改部署列表、不改编辑器窗口、不改 store 数据结构、不改后端接口
- 验收标准：
  - 本地项目列表中不再出现 `Flow ID`、`Project ID`
  - 名称和更新时间显示在同一行，视觉上存在明确空隔
  - 当项目名为空时，不再回退显示 `flowId`
  - 列表上的 `Meta`、`Edit`、`Deploy`、`Delete` 行为不回归

## 架构设计摘要

- 采用方案：只在 [`frontend/src/pages/Flow.vue`](D:/project/MyFlowHub3/worktrees/MyFlowHub-Win-feat-flow-list-row-simplify/frontend/src/pages/Flow.vue) 调整模板渲染
- 选型理由：
  - 这是纯展示收敛，不应把格式化逻辑下沉到 store
  - 单文件改动可以最小化影响范围，便于审计与回滚
- 关键设计点：
  - 使用现有国际化文案 `Untitled Project` 作为空名称兜底，避免暴露 `flowId`
  - 使用现有设计 token `text-muted-foreground`，避免硬编码颜色值
  - 不新增请求、不新增状态、不新增计算

## 可执行任务清单

- [x] `T1` 调整 Flow 本地项目列表展示
- [x] `T2` 执行验证与 Code Review
- [x] `T3` 归档变更并更新索引

## 任务详情

### `T1` 调整 Flow 本地项目列表展示

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-flow-list-row-simplify`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-flow-list-row-simplify\plan.md`
- 目标：
  - 修改本地项目列表项标题区，只保留名称与更新时间
  - 移除列表中的 `Flow ID` 和 `Project ID`
  - 消除 `name || flowId` 的回退展示
- 涉及模块 / 文件：
  - `frontend/src/pages/Flow.vue`
- Write set：
  - `frontend/src/pages/Flow.vue`
- 关键上下文引用：
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/i18n/messages/automation.ts`
- 依赖：无
- 验收条件：
  - `Flow ID`、`Project ID` 文案不再出现在本地项目列表模板
  - 名称与更新时间位于同一行
  - 该行使用灰色文本样式
  - 名称为空时显示 `Untitled Project`
- 测试点：
  - 静态核对模板渲染片段
  - Vue SFC 解析通过
- 回滚点：
  - 回退 `frontend/src/pages/Flow.vue`
- 风险与注意事项：
  - 只改本地项目列表，不误伤部署列表区域

### `T2` 执行验证与 Code Review

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-flow-list-row-simplify`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-flow-list-row-simplify\plan.md`
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
  - 文本检查确认列表不再渲染 `Flow ID` 和 `Project ID`
- 回滚点：
  - 回退 `T1` 的模板修改
- 风险与注意事项：
  - 当前任务为单文件 UI 变更，不做无关构建链路修复

### `T3` 归档变更并更新索引

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-flow-list-row-simplify`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-flow-list-row-simplify\plan.md`
- 目标：
  - 在 `docs/change/` 记录本次 UI 收敛变更，并更新归档索引
- 涉及模块 / 文件：
  - `docs/change/2026-03-22_flow-list-row-simplify.md`
  - `docs/change/README.md`
- Write set：
  - `docs/change/2026-03-22_flow-list-row-simplify.md`
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

1. 静态核对 `frontend/src/pages/Flow.vue` 中本地项目列表模板
2. 使用 `@vue/compiler-sfc` 解析目标 SFC
3. 完成 3.3 Code Review 后进入 4

## 执行结果

- `T1` 实现结果：
  - 已删除本地项目列表中的 `Flow ID` 与 `Project ID` 显示
  - 已将 `name` 与 `updatedAt` 合并为同一行灰色文本
  - 已将空名称兜底从 `flowId` 改为 `Untitled Project`
- `T2` 验证结果：
  - 文本核对通过：本地项目列表模板中不再渲染 `Flow ID` / `Project ID`
  - Vue SFC 解析通过：`frontend/src/pages/Flow.vue`
  - `npm run build` 未通过：失败原因是既有 `frontend/wailsjs` 生成物缺失，报错落在 `src/pages/Home.vue` 对 `../../wailsjs/go/main/App` 的解析，与本次改动无关

## Code Review 结论

- 需求覆盖：通过
  - 已覆盖“隐藏流程 ID / 项目 ID”“名称与更新时间同一行”“灰色样式”“空名称不回退 flowId”
- 架构合理性：通过
  - 改动限定在页面模板层，未把展示策略错误下沉到 store
- 性能风险：通过
  - 无新增请求、无重复计算、无额外 I/O、无列表级性能回退点
- 可读性与一致性：通过
  - 使用现有 `t(...)` 与 `text-muted-foreground` 设计 token，模板收敛后更直接
- 可扩展性与配置化：通过
  - 数据层未耦合 UI 规则，后续若做列表密度切换仍可只改页面层
- 稳定性与安全：通过
  - 不涉及接口、权限、持久化结构或敏感数据处理
- 测试覆盖情况：通过
  - 已完成目标文件语法校验与静态渲染点核对；完整前端构建受既有 `wailsjs` 缺失阻塞，已明确记录为环境残余问题
- 子Agent治理与审计：通过
  - 本轮未使用子Agent；原因已在计划文档中记录，未发生任务映射、文件所有权或集成冲突问题
