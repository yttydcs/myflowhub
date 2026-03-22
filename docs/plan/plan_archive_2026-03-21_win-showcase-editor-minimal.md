# Showcase 编辑器进一步简化计划

## 项目目标与当前状态
- 目标：继续简化 Showcase 编辑界面，使其更接近 Flow 的简洁编辑体验。
- 当前状态：`Showcase` 已具备列表页与独立编辑页；本轮聚焦编辑页进一步收敛交互复杂度，不调整后端和 viewer 运行逻辑，除非分析阶段确认有必要。

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`refactor/showcase-editor-minimal`
- Base：`main`
- Worktree：`D:\\project\\MyFlowHub3\\worktrees\\MyFlowHub-Win-showcase-editor-minimal`
- 当前阶段：`4. 归档变更（已完成，待用户确认是否结束 workflow）`

## 需求分析结论（阶段 1）
- 目标：继续压缩 Showcase 编辑页，只保留编辑主线，去掉与列表页重复的 screen 元信息编辑。
- 范围（必须）：
  - 移除编辑页中的 `Screen Name` 输入；
  - `Layout Mode / Max Columns / Min Column Width / Apply Layout` 改为顶部按钮触发的弹窗编辑；
  - 编辑页改为接近 Flow 的单屏编辑结构，避免页面级滚动；
  - 保留 widget 直接交互、右键菜单、保存/回退、新增 widget、两种布局模式。
- 范围（不做）：
  - 不改列表页、viewer、store schema、后端接口；
  - 不新增 screen 重命名入口。
- 使用场景：
  - 用户从 `Showcase Center` 打开独立编辑窗口，在固定窗口中直接编辑 widget 与布局；
  - 用户在中小窗口下仍能看到顶部工具栏与主编辑区，无需上下滚动页面。
- 输入 / 输出：
  - 输入：当前 screen 草稿、布局配置、窗口尺寸；
  - 输出：更极简的编辑页结构，保存后的 screen 数据结构不变。
- 边界异常：
  - `screenId` 缺失或 screen 不存在时，继续显示现有缺失态；
  - 切换到 `canvas_percent` 时，继续复用现有 canvas 初始化逻辑；
  - `saveDraft` 仍保留对空名称的防御式校验，但本轮不提供名称编辑入口。
- 验收标准：
  - 主界面不再展示 `Screen Name` 编辑；
  - 布局设置仅通过顶部按钮打开弹窗编辑；
  - 编辑窗口不出现页面级滚动，编辑区跟随窗口剩余空间自适应；
  - `columns` / `canvas_percent`、widget 直接操作、保存 / 回退均正常。
- 风险：
  - `columns` 模式在极端大量 widget 下仍可能出现内容密度过高，本轮不做复杂自动缩放；
  - 若只改页面不改窗口壳层，无法彻底消除滚动。

## 架构设计结论（阶段 2）
- 总体方案：
  - 通过路由 meta 标记 full-bleed window，让 `showcaseEditorWindow` 与 `flowEditorWindow` 共用 `AppShell` 的 `h-screen + overflow-hidden` 壳层；
  - `Showcase.vue` 改为 `header + flex-1 editor surface` 结构，主界面只保留必要工具条与编辑面板；
  - 布局编辑下沉为弹窗，继续调用现有 `saveScreenLayout`，不改持久化格式。
- 选型理由：
  - 仅改 `Showcase.vue` 无法根除外层滚动；
  - 使用 route meta 比在 `AppShell` 硬编码 route name 更可扩展。
- 模块职责：
  - `frontend/src/router/index.ts`：声明 full-bleed window 元信息；
  - `frontend/src/layout/AppShell.vue`：根据 route meta 决定窗口壳层尺寸策略；
  - `frontend/src/pages/Showcase.vue`：实现编辑页极简 UI、布局弹窗、自适应编辑区。
- 数据 / 调用流：
  - 打开编辑窗口 -> 读取 full-bleed route meta -> `Showcase.vue` 占满窗口剩余高度；
  - 顶部按钮打开布局弹窗 -> 修改布局字段 -> `saveScreenLayout` 校验并保存 -> 预览区重渲染。
- 错误与安全：
  - 继续复用现有输入校验、busy 锁与 toast 错误提示；
  - 弹窗关闭不自动提交。
- 性能与测试策略：
  - 不增加额外 I/O；
  - 通过 `flex-1 min-h-0` 避免多层 viewport 高度计算导致的布局抖动；
  - 重点验证无页面滚动、弹窗布局编辑、两种布局模式与 widget 直接交互。
- 可扩展性设计点：
  - full-bleed 窗口策略可复用到其它编辑器窗口；
  - 后续新增布局字段时只扩展弹窗，不污染主界面。

## 可执行任务清单
- [x] T1 明确本轮需求边界与验收标准
  - 目标：确认编辑器简化范围、保留能力与不做项。
  - 涉及模块 / 文件：需求分析文档、`frontend/src/pages/Showcase.vue`、相关样式与交互实现。
  - 验收条件：需求、边界、风险、验收标准明确，无阻塞问题。
  - 测试点：N/A
  - 回滚点：仅文档更新，可直接修改 `plan.md`
  - 依赖关系：无
  - 风险与注意事项：若涉及布局模式编辑交互变化，需要确认是否影响现有数据保存方式。
  - Owner：主 Agent
  - Write set：`plan.md`
  - 关键上下文引用：用户本轮界面简化要求、现有 `Showcase.vue` 结构

- [x] T2 完成架构设计与交互方案
  - 目标：定义顶部按钮 + 弹窗编辑布局模式、移除名称编辑、取消页面内部滚动的实现方案。
  - 涉及模块 / 文件：`frontend/src/pages/Showcase.vue`、`frontend/src/layout/AppShell.vue`、`frontend/src/router/index.ts`
  - 验收条件：模块职责、数据流、错误处理、性能与测试策略明确。
  - 测试点：N/A
  - 回滚点：仅文档更新，可直接修改 `plan.md`
  - 依赖关系：T1
  - 风险与注意事项：需避免破坏当前两种布局模式及直接编辑 widget 的能力。
  - Owner：主 Agent
  - Write set：`plan.md`
  - 关键上下文引用：`frontend/src/layout/AppShell.vue`、`frontend/src/router/index.ts`、`frontend/src/pages/Showcase.vue`

- [x] T3 实现编辑器极简化改造
  - 目标：按确认方案修改页面结构与交互，并让编辑窗口获得 full-bleed 壳层。
  - 涉及模块 / 文件：`frontend/src/layout/AppShell.vue`、`frontend/src/router/index.ts`、`frontend/src/pages/Showcase.vue`
  - 验收条件：名称输入移除；布局模式改为顶部按钮触发弹窗；页面编辑区无内部滚动并自适应窗口；保留直接操作 widget。
  - 测试点：
    - Vue SFC 语法解析；
    - 关键文本检查：主界面不再出现 `Screen Name`；
    - 路由 / 壳层检查：`showcaseEditorWindow` 进入 full-bleed 窗口布局；
    - 冒烟：布局弹窗、保存布局、保存草稿、columns / canvas 预览结构仍可渲染。
  - 回滚点：
    - 回滚 `frontend/src/pages/Showcase.vue`
    - 回滚 `frontend/src/layout/AppShell.vue`
    - 回滚 `frontend/src/router/index.ts`
  - 依赖关系：T2
  - 风险与注意事项：需控制样式收缩，避免小窗口下工具区遮挡主编辑区。
  - Owner：主 Agent
  - Worktree 路径：`D:\\project\\MyFlowHub3\\worktrees\\MyFlowHub-Win-showcase-editor-minimal`
  - Plan 路径：`D:\\project\\MyFlowHub3\\worktrees\\MyFlowHub-Win-showcase-editor-minimal\\plan.md`
  - Write set：
    - `frontend/src/pages/Showcase.vue`
    - `frontend/src/layout/AppShell.vue`
    - `frontend/src/router/index.ts`
  - 关键上下文引用：
    - `frontend/src/windows/FlowEditorWindow.vue`
    - `frontend/src/layout/AppShell.vue`
    - `frontend/src/router/index.ts`
  - 实施结果：
    - 已移除主界面 `Screen Name` 编辑；
    - 已新增顶部 `Layout` 按钮与布局弹窗；
    - 已将 `showcaseEditorWindow` 接入 full-bleed window；
    - 已将编辑页改为 `header + flex-1 preview` 结构，消除页面级滚动。

- [x] T4 进行 Code Review 与归档
  - 目标：审查需求覆盖、架构合理性、风险与验证结果，并输出变更归档。
  - 涉及模块 / 文件：`plan.md`、`docs/change/*`、已修改代码文件
  - 验收条件：Review 通过，生成归档文档。
  - 测试点：核对实现与文档一致性。
  - 回滚点：按任务粒度回退相应文档与代码文件。
  - 依赖关系：T3
  - 风险与注意事项：需保留与本轮无关的已有脏改动。
  - Owner：主 Agent
  - Worktree 路径：`D:\\project\\MyFlowHub3\\worktrees\\MyFlowHub-Win-showcase-editor-minimal`
  - Plan 路径：`D:\\project\\MyFlowHub3\\worktrees\\MyFlowHub-Win-showcase-editor-minimal\\plan.md`
  - Write set：
    - `plan.md`
    - `docs/change/*`
  - 关键上下文引用：
    - T3 实现结果
    - 本 workflow 的需求 / 架构结论
  - 产出：
    - Review 结论：通过；
    - 归档文档：`docs/change/2026-03-21_win-showcase-editor-minimal.md`

## 并行与所有权
- 当前判断：本轮核心改动集中在 `Showcase.vue`，且 `AppShell.vue` / `router/index.ts` 与其高度耦合，写集重叠且需要统一验证窗口壳层行为；因此不适合安全拆分给子 Agent。进入 `3.2` 时已再次评估，如无新增独立任务，继续由主 Agent 单独执行。

## 验证结果
- `npm ci`：通过
- Vue SFC 解析：通过
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/layout/AppShell.vue`
- `npm run build`：未通过
  - 原因：现有 Wails 绑定缺失，失败点为 `src/pages/Home.vue` 依赖 `../../wailsjs/go/session/SessionService`，与本轮改动无关

## Code Review 结论（阶段 3.3）
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：部分通过（受既有 Wails bindings 问题影响）
- 子Agent治理与审计：通过（本轮未使用子 Agent）

## 阶段 4 归档结果
- 已新增：`docs/change/2026-03-21_win-showcase-editor-minimal.md`
- 当前状态：可进入 workflow 结束确认
