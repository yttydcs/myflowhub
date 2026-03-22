# Win Showcase 顶部按钮图标化

## 项目目标与当前状态
- 目标：将 Showcase 编辑窗口顶部仍带文字的按钮改为图标按钮，原文字通过 tooltip 展示，进一步贴近 Flow 的工具栏风格。
- 当前状态：
  - `frontend/src/pages/Showcase.vue` 顶部按钮中，`Refresh Vars`、`Add Event`、`Add Var` 已是图标按钮。
  - `Save`、`Revert`、`Layout`、`Open Viewer` 仍是文字按钮。
  - `frontend/src/windows/ShowcaseWindow.vue` 当前没有顶部控制按钮。

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`refactor/showcase-header-icon-actions`
- Base：`main`
- Worktree：`D:\\project\\MyFlowHub3\\worktrees\\MyFlowHub-Win-showcase-header-icon-actions`
- 当前阶段：`4 归档变更（待用户确认是否结束 workflow）`

## 需求分析结论（阶段 1）
- 目标：把 Showcase 编辑窗口顶部按钮统一图标化，按钮文字改由 tooltip 承载。
- 必须范围：
  - `frontend/src/pages/Showcase.vue` 顶部 `Save`、`Revert`、`Layout`、`Open Viewer` 改成图标按钮。
  - 为这些按钮补充 tooltip，并保留可访问的 `sr-only` 文案。
  - 保持现有按钮顺序、禁用逻辑和点击行为不变。
- 可选范围：无。
- 不做：
  - 不新增 viewer 顶部按钮。
  - 不修改 Showcase store、保存逻辑、layout 弹窗逻辑、viewer 打开逻辑。
  - 不改页面其余区域和列表页按钮。
- 使用场景：用户在 Showcase 编辑窗口顶部进行保存、回退、编辑布局、打开 viewer 等操作时，看到与 Flow 更一致的图标工具栏。
- 功能需求：
  - 文字按钮图标化后，原含义仍能通过 tooltip 和无障碍文本读取。
  - 按钮变体和可点击状态仍与现有行为一致。
- 非功能需求：
  - 变更最小化。
  - 不降低可访问性。
  - 不影响顶部 toolbar 的换行适配。
- 输入输出：
  - 输入：当前 `Showcase.vue` 头部按钮区。
  - 输出：图标化的工具栏按钮。
- 边界异常：
  - `screenMissing` 时顶部按钮区仍按现有逻辑隐藏。
  - `busy`、`dirty` 等禁用条件保持不变。
- 验收标准：
  - `Save`、`Revert`、`Layout`、`Open Viewer` 不再在按钮正文中显示文字。
  - 对应 tooltip 可展示原动作名称。
  - 顶部按钮点击行为与禁用逻辑无回归。
- 风险：
  - 若缺少合适的 `sr-only` 文案，会降低可访问性。
- 问题清单：无。
- 阻塞：否。

## 架构设计结论（阶段 2）
- 总体方案：只修改 `frontend/src/pages/Showcase.vue` 的头部按钮模板和相关 icon import，不调整脚本行为。
- 设计要点：
  - 复用 Flow 编辑器现有的 icon + tooltip 工具栏模式。
  - `Save` 使用 `Save` icon。
  - `Revert` 使用 `Undo2` icon。
  - `Layout` 继续使用 `Settings2` icon，但移除按钮正文文字。
  - `Open Viewer` 继续使用 `ExternalLink` icon，但移除按钮正文文字。
- 数据 / 调用流：保持不变，仍调用现有 `saveDraft`、`revertDraft`、`openLayoutDialog`、`openShowcaseWindow`。
- 接口草案：无新增接口。
- 错误与安全：不新增错误路径；保留原按钮禁用条件。
- 性能与测试策略：
  - 仅模板与 import 微调，无新增运行时开销。
  - 通过静态文本搜索和 Vue SFC 解析验证按钮正文已去文字且模板结构正确。
- 可扩展性设计点：
  - 头部按钮模式进一步对齐 Flow，后续若继续压缩 toolbar，可以在同一区域延续 icon-only 设计。
- 备选方案对比：
  - 保留文字按钮，仅补 tooltip：不满足图标化目标，不采用。
  - 改为图标按钮并保留 tooltip 与 `sr-only`：满足目标且可访问性更完整，采用。
- 问题清单：无。
- 阻塞：否。

## 可执行任务清单
- [x] T1 明确顶部按钮图标化边界
  - 目标：确认只处理 Showcase 编辑窗口头部文字按钮，viewer 无按钮则忽略。
  - 涉及模块 / 文件：`plan.md`、`frontend/src/pages/Showcase.vue`
  - 验收条件：需求边界和验收标准明确，无阻塞问题。
  - 测试点：N/A
  - 回滚点：修改 `plan.md`
  - 依赖关系：无

- [x] T2 完成实现方案设计
  - 目标：确定按钮对应 icon、tooltip 文案和可访问性策略。
  - 涉及模块 / 文件：`plan.md`、`frontend/src/pages/Showcase.vue`
  - 验收条件：模板修改点明确，行为保持不变。
  - 测试点：N/A
  - 回滚点：修改 `plan.md`
  - 依赖关系：T1

- [x] T3 实现 Showcase 头部按钮图标化
  - 目标：完成 `Save`、`Revert`、`Layout`、`Open Viewer` 的图标按钮化与 tooltip 替换。
  - 涉及模块 / 文件：`frontend/src/pages/Showcase.vue`
  - 验收条件：头部按钮全部使用图标；tooltip 展示动作名称；行为无回归。
  - 测试点：静态搜索、Vue SFC 解析。
  - 回滚点：回退 `frontend/src/pages/Showcase.vue`。
  - 依赖关系：T2

- [x] T4 进行 Code Review 与归档
  - 目标：输出 review 结论并归档本轮按钮图标化改动。
  - 涉及模块 / 文件：`plan.md`、`docs/change/*`
  - 验收条件：review 通过，归档文档完整。
  - 测试点：实现与文档一致性。
  - 回滚点：删除新增文档并回退代码。
  - 依赖关系：T3

## 并行与所有权
- 当前判断：本轮改动集中在单个 Vue 文件，目标单一且写集唯一，不适合拆分子 Agent，由主 Agent 串行完成。

## Code Review（阶段 3.3）
- 需求覆盖：通过。
  - `Save`、`Revert`、`Layout`、`Open Viewer` 已改为图标按钮。
  - 原按钮文案已通过 tooltip 和 `sr-only` 文本保留。
  - `Refresh Vars`、`Add Event`、`Add Var` 维持原图标按钮模式，没有被破坏。
- 架构合理性：通过。
  - 改动只落在 `frontend/src/pages/Showcase.vue` 模板层和 icon import，不影响任何业务逻辑。
- 性能风险：通过。
  - 仅替换静态按钮内容，无新增计算、I/O 或订阅。
- 可读性与一致性：通过。
  - 工具栏风格更接近 `FlowEditorWindow.vue` 的 icon + tooltip 模式。
- 可扩展性与配置化：通过。
  - 后续若继续压缩或重排 Showcase 头部按钮，可继续在同一模式上演进。
- 稳定性与安全：通过。
  - 保留原禁用条件、点击处理和 tooltip 语义，没有改变保存、回退、布局弹窗或打开 viewer 的行为。
- 测试覆盖情况：有条件通过。
  - 静态搜索确认头部目标按钮不再以文字正文形式渲染：通过。
  - `@vue/compiler-sfc` 解析 `src/pages/Showcase.vue`：通过。
  - `npm run build`：未通过；失败点是现有 `src/pages/Home.vue` 无法解析 `../../wailsjs/go/session/SessionService`，与本次改动无关。
- 子Agent治理与审计：通过。
  - 本轮未使用子Agent；原因是任务集中于单个文件，拆分没有收益。
