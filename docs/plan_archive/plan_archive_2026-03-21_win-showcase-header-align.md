# Win Showcase 头部对齐 Flow

## 项目目标与当前状态
- 目标：将 Showcase Editor / Viewer 的窗口头部对齐到更接近 Flow 的单行结构，上方一行承载名称、状态和控制按钮，下方直接进入编辑或展示内容。
- 当前状态：
  - `frontend/src/pages/Showcase.vue` 的编辑窗口头部目前分成两层，内容区还保留 `Editing Surface / Layout` 说明块。
  - `frontend/src/windows/ShowcaseWindow.vue` 的 viewer 头部仍是两层信息结构，且窗口骨架与 editor / flow 不一致。

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`refactor/showcase-header-align`
- Base：`main`
- Worktree：`D:\\project\\MyFlowHub3\\worktrees\\MyFlowHub-Win-showcase-header-align`
- 当前阶段：`4 归档变更（待用户确认是否结束 workflow）`

## 需求分析结论（阶段 1）
- 目标：把 `Showcase Editor` 和 `Showcase Viewer` 都收敛成与 `Flow` 更接近的窗口结构。
- 必须范围：
  - 顶部只保留一行主 header。
  - `Editor` 顶部左侧显示 screen 名称与状态信息，右侧收纳按钮排。
  - `Viewer` 顶部也改为单行 header，下面直接进入展示内容；没有按钮的部分不强行补。
  - 移除 `Editor` 内容区上方额外的 `Editing Surface / Layout` 说明块。
- 可选范围：无。
- 不做：
  - 不新增 viewer 控制逻辑。
  - 不修改 Showcase store、路由、保存逻辑、widget 交互。
  - 不改变 columns / canvas 两种布局本身。
- 使用场景：打开独立编辑窗口或展示窗口时，第一屏先看到单行头部，其余空间直接给编辑/展示区。
- 功能需求：
  - Editor / Viewer 的顶部结构统一收敛。
  - Editor 的按钮区进入 header 右侧。
  - 内容区不再保留多余说明性副 header。
- 非功能需求：
  - 变更最小化。
  - 不增加额外滚动。
  - 保持小窗口下的可换行适配。
- 输入输出：
  - 输入：现有 Showcase Editor / Viewer Vue 模板。
  - 输出：更贴近 Flow 的单行头部与更直接的内容区。
- 边界异常：
  - `screenMissing`、`!loaded`、断连等状态保持现有逻辑。
  - 窄宽度下允许 header 信息和按钮自然换行，避免重叠。
- 验收标准：
  - Editor / Viewer 头部都收敛为单行结构。
  - Editor 的按钮排进入 header 右侧，不再单独占一层。
  - Editor 内容区不再出现说明性副 header。
  - 页面主交互不回归。
- 风险：
  - 信息项压缩到同一 header 后，如果排列不当，会在小宽度下显得拥挤。
- 问题清单：无。
- 阻塞：否。

## 架构设计结论（阶段 2）
- 总体方案：只改 `frontend/src/pages/Showcase.vue` 和 `frontend/src/windows/ShowcaseWindow.vue` 的模板层，必要时清理一个不再使用的 `computed`。
- Editor 方案：
  - header 改为单层 `justify-between`。
  - 左侧聚合 `screen name + connection + unsaved + layout + widget count + time`。
  - 右侧保留现有按钮排。
  - 删除内容区顶部 `Editing Surface / Layout` 说明块，让预览面板直接顶到内容区顶部。
- Viewer 方案：
  - 采用与 editor / flow 类似的窗口骨架：`section + header + content`。
  - 头部左侧聚合 `screen name + connection + id/self/hub`，无按钮时右侧留空。
- 数据 / 调用流：保持不变，仍使用现有 `showcase` store、`sessionStore` 和 route query。
- 接口草案：无新增接口。
- 错误与安全：不新增错误路径；现有加载中、screen 缺失、连接态处理保持不变。
- 性能与测试策略：
  - 仅模板收口和少量死代码清理，无新增运行时开销。
  - 通过文本搜索、静态编译校验验证头部结构变更。
- 可扩展性设计点：
  - 后续继续精简 Showcase 头部时，可集中在同一 header 区域调整。
- 备选方案对比：
  - 保留双层 header，只压缩间距：改动小，但仍不符合“上面一行”的目标，不采用。
  - 改成单行 header，并删除内容区副说明：更符合 Flow 风格，采用。
- 问题清单：无。
- 阻塞：否。

## 可执行任务清单
- [x] T1 明确本轮 Showcase 头部对齐目标
  - 目标：确定 Editor / Viewer 都要收敛到单行 header，并明确保留与移除项。
  - 涉及模块 / 文件：`plan.md`、`frontend/src/pages/Showcase.vue`、`frontend/src/windows/ShowcaseWindow.vue`
  - 验收条件：需求边界、风险、验收标准明确，无阻塞问题。
  - 测试点：N/A
  - 回滚点：修改 `plan.md`
  - 依赖关系：无

- [x] T2 完成实现方案设计
  - 目标：明确 Editor / Viewer 的头部布局对齐方案，以及 Editor 内容区副 header 的去留。
  - 涉及模块 / 文件：`plan.md`、`frontend/src/pages/Showcase.vue`、`frontend/src/windows/ShowcaseWindow.vue`
  - 验收条件：模板调整点、保留信息和验证策略明确。
  - 测试点：N/A
  - 回滚点：修改 `plan.md`
  - 依赖关系：T1

- [x] T3 实现 Showcase 头部对齐与内容区收口
  - 目标：完成 Editor / Viewer 头部单行化，并移除 Editor 内容区说明性副 header。
  - 涉及模块 / 文件：`frontend/src/pages/Showcase.vue`、`frontend/src/windows/ShowcaseWindow.vue`
  - 验收条件：头部结构与 Flow 风格更接近；Editor 按钮排在 header 右侧；Viewer 下面直接进入展示内容。
  - 测试点：文本搜索、静态编译或构建验证。
  - 回滚点：回退上述两个文件。
  - 依赖关系：T2

- [x] T4 进行 Code Review 与归档
  - 目标：输出 review 结论并归档本轮结构对齐改动。
  - 涉及模块 / 文件：`plan.md`、`docs/change/*`
  - 验收条件：review 通过，归档文档完整。
  - 测试点：实现与文档一致性。
  - 回滚点：删除新增文档并回退代码。
  - 依赖关系：T3

## 并行与所有权
- 当前判断：本轮改动集中在两个 Vue 文件，目标强耦合且都属于同一套头部结构收口，拆分子 Agent 容易导致样式方向不一致，因此由主 Agent 串行完成。

## Code Review（阶段 3.3）
- 需求覆盖：通过。
  - `frontend/src/pages/Showcase.vue` 已收敛为单行 header，左侧展示名称与状态，右侧展示按钮排。
  - `frontend/src/windows/ShowcaseWindow.vue` 已改为单行 header + 内容区骨架。
  - Editor 内容区上方的 `Editing Surface / Layout` 说明块已移除。
- 架构合理性：通过。
  - 改动仅限模板层与一个不再使用的 `computed` 清理，没有侵入 store、路由或业务逻辑。
- 性能风险：通过。
  - 仅减少静态 DOM 与无用计算，无新增 I/O、订阅或重复计算。
- 可读性与一致性：通过。
  - Editor / Viewer 都采用相近的窗口骨架，结构更统一，维护成本更低。
- 可扩展性与配置化：通过。
  - 头部信息与按钮区已集中到单个 header 区域，后续继续收口或增减状态项时改动面更集中。
- 稳定性与安全：通过。
  - 未修改保存逻辑、权限逻辑、数据结构、widget 编辑行为或 viewer 运行行为。
- 测试覆盖情况：有条件通过。
  - `@vue/compiler-sfc` 静态解析 `src/pages/Showcase.vue` 与 `src/windows/ShowcaseWindow.vue`：通过。
  - 文本搜索确认 `currentLayoutSummary`、`Editing Surface`、旧 Showcase 标题文案已不存在：通过。
  - `npm run build`：未通过；失败点是现有 `src/pages/TopicBus.vue` 无法解析 `../../wailsjs/go/main/App`，与本次改动无关。
- 子Agent治理与审计：通过。
  - 本轮未使用子Agent；原因是任务集中于两个强耦合模板文件，拆分没有收益。
