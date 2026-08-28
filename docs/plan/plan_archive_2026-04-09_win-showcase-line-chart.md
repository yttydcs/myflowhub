# Plan - MyFlowHub-Win Showcase Line Chart Widget

## Workflow Information
- Repo: `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win`
- Branch: `feat/showcase-chart-widgets`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`:
  - workspace root `D:\project\MyFlowHub3\guide.md` 已阅读
  - 遵守 `AGENTS.md` 与 `$m-autoflow` 约束：实现只在 `worktrees/` 中进行，commit 信息使用中文，计划确认前不进入编码
- base/worktree confirmation:
  - 控制面 worktree：`D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets`
  - Win 实现 worktree：`D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win`
  - 控制面主仓 `D:\project\MyFlowHub3` 当前存在用户自己的未提交改动，仅用于 workflow 编排和后续归档
  - `repo/MyFlowHub-Win` 主仓当前存在用户自己的未提交改动（`go.mod`、`myflowhub-mcp.exe`），保持控制面只读
  - 本轮唯一执行实现 worktree：`D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win`
  - 本轮只修改 `MyFlowHub-Win`，控制面 worktree 仅用于 Stage 4 归档同步

### Stage 1 - Requirements Analysis
#### Goal
- 在现有 Showcase `var` widget 基础上补充折线图展示能力。
- 折线图只依赖现有实时变量快照，在前端内存中保留采样历史，不新增后端历史查询或数据持久化。
- 用户可在界面上调整显示时间范围和采样粒度，同时保持当前 Showcase 的轻量、卡片式风格。

#### Scope
- Must:
  - 为 `var` widget 增加新的显式展示模式：`line_chart`
  - 折线图基于当前 app session 内收到的变量实时快照生成历史曲线
  - Editor 中可以配置折线图模式，Viewer 中可以一致渲染
  - 用户可调整显示时间范围和粒度
  - 折线图在 `columns` 与 `canvas_percent` 两种布局下都保持可读
  - 空值、非数值、样本不足、时间窗内无数据时必须明确降级，不得报错或破坏布局
  - 现有 `display`、`metric`、`badge`、`progress`、`slider`、`switch`、`topic_button` 行为不得回退
- Optional:
  - 在图表卡片上补充轻量统计信息，如最新值或采样点数，只要不破坏简洁风格
  - 为时间范围和粒度提供少量稳定预设，避免自由输入导致配置失控
- Not in scope:
  - 不实现饼图、多变量图表、堆叠图或聚合型 dashboard
  - 不新增后端历史接口、数据库持久化或磁盘缓存
  - 不支持查询“当前 session 之外”的历史日期数据
  - 不重做 Showcase Center、布局模型、多窗口同步或变量订阅机制

#### Use Cases
- 操作人员希望在 Showcase 中直接看到某个数值变量最近一段时间的变化趋势。
- 操作人员希望切换查看近 15 分钟、1 小时、6 小时、24 小时等不同时间窗口。
- 操作人员希望在高频变量下把显示粒度调整为 10 秒、1 分钟、5 分钟等，以减少噪声。
- 配置人员希望在 Editor 中预览折线图效果，并在 Viewer 中获得一致结果。

#### Functional Requirements
1. `var` widget 编辑器必须允许用户选择 `line_chart` 模式。
2. `line_chart` 必须只对数值变量绘制曲线；当前值无法解析为数值时显示明确降级态。
3. `line_chart` 必须支持可调时间范围和粒度。
4. 时间范围和粒度配置必须保存到 Showcase 配置中；采样数据本身不得持久化。
5. 折线图必须随实时变量更新追加前端内存样本，并按当前显示配置重新计算可见序列。
6. 若时间范围内无有效样本、样本不足两点或当前 session 尚未积累足够数据，界面必须给出可理解的提示。
7. `auto` 模式必须保持既有推断规则，不自动切换到 `line_chart`。
8. 现有交互型模式 `slider` / `switch` 的写回语义不得被折线图逻辑影响。

#### Non-functional Requirements
- 兼容性：
  - 旧 Showcase 配置可以直接加载
  - 未识别 mode 必须回退为 `auto`
  - 缺失或非法图表配置必须回退到安全默认值
- 简洁性：
  - 继续保持 Showcase 的轻量卡片视觉，不引入重型图表面板
  - 控制项和图表信息量保持克制
- 可维护性：
  - 折线图渲染与序列计算尽量集中，避免 Editor / Viewer 双份逻辑漂移
  - 样本采集与裁剪逻辑需要明确边界，避免内存无限增长
- 性能：
  - 不新增额外订阅种类或后端 I/O
  - 样本缓存必须有 retention 上限和裁剪策略
  - 可见序列聚合应按需计算，避免无意义重复遍历

#### Inputs / Outputs
- Inputs:
  - `VarStore` 现有实时快照流
  - `ShowcaseVarWidget` 当前配置
  - 用户选择的显示时间范围和粒度
- Outputs:
  - 持久化后的 Showcase widget 配置（包含 `line_chart` 配置）
  - 当前 app session 内的前端内存样本序列
  - Editor / Viewer 一致的折线图渲染结果

#### Edge Cases
- 变量当前值为空、非数值或频繁在数值/非数值之间切换
- 时间范围大于当前 session 已积累的样本时长
- 粒度大于时间范围，或粒度配置非法
- 高频更新导致样本过多，需要明确裁剪上限
- 样本全部相同导致 y 轴跨度为 0
- 屏幕切换、Viewer 重开或 layout 改变时，图表控件与历史采样不能造成页面异常

#### Acceptance Criteria
1. 用户可以在 Editor 中创建或编辑 `line_chart` 模式的 `var` widget。
2. 保存后重新打开 Editor 或 Viewer，`line_chart` 模式及其显示时间范围、粒度配置能够保留。
3. 折线图在 `columns` 与 `canvas_percent` 下都能正常显示，不破坏布局。
4. 折线图只在前端内存中保留样本，不新增后端历史持久化。
5. 非数值、无样本和样本不足场景有明确降级提示，不报错。
6. 相关自动化验证通过：`go test ./... -count=1`、前端定向 Vitest、`npm run build`。

#### Risks
- 当前稳定 requirement/spec 明确把趋势图列为超范围，本轮必须同步更新文档，否则后续会再次产生认知冲突。
- 样本缓存如果按 widget 而不是按变量 key 管理，容易产生重复内存占用和逻辑分叉。
- “日期”在无后端历史查询前提下只能解释为“当前 session 内的相对时间窗”，如果用户期望任意自然日查询，需要重新开需求。

#### Issue List
- 约定：本轮“显示的日期”实现为相对时间窗和时间轴标签，不支持跨 session 的任意历史日期查询。

### Stage 2 - Architecture Design
#### Overall Solution
- 继续沿用现有 `ShowcaseVarWidget.mode` 扩展方案，在现有 `var` widget 上新增 `line_chart`，而不是发明新的 widget kind。
- 在 Go 与前端 store 中同时扩展持久化模型和 normalize 逻辑，使 `line_chart` 配置可以稳定保存和回读。
- 将图表采样数据严格限定为前端内存态：
  - 基于现有 `upsertSnapshot` 实时更新
  - 仅在前端保存样本数组
  - 不进入 `showcase.config`
- 通过共享渲染组件输出折线图，保持 Editor / Viewer 的一致性。
- 使用轻量 SVG 路径绘制折线图，避免引入新的图表依赖。

#### Alternatives Considered
- 新增独立 widget kind（如 `var_line_chart`）：
  - 放弃。会扩大 schema 和表单结构，且重复现有 `var` 的订阅与权限语义。
- 引入第三方 chart 库：
  - 放弃。当前需求只需要简单折线图，增加依赖和样式体系不划算。
- 只做前端临时图表、不改 Go schema：
  - 放弃。`ShowcaseConfig` 回存时无法稳定保留 `line_chart` 及其显示配置。
- 支持绝对日期选择：
  - 放弃。当前没有后端历史数据源，只能在 session 内做相对时间窗。

#### Module Responsibilities
- `docs/requirements/showcase-display-widgets.md`
  - 更新长期需求，纳入 `line_chart`、时间范围、粒度和“样本不持久化”的稳定口径
- `docs/specs/showcase-display-widgets.md`
  - 更新模式枚举、配置模型、采样边界和性能约束
- `app_showcase.go`
  - 扩展 `ShowcaseVarWidget` 持久化模型与 normalize
- `app_showcase_test.go`
  - 补齐 `line_chart` 配置保留与 normalize 回归测试
- `frontend/src/stores/showcase.ts`
  - 扩展类型定义、默认值、normalize
  - 维护内存样本缓存、时间窗裁剪和粒度聚合
  - 暴露折线图可见序列和降级状态
- `frontend/src/components/showcase/*`
  - 复用或拆分共享渲染组件，集中处理折线图 SVG 和图表 controls
- `frontend/src/pages/Showcase.vue`
  - 扩展 widget 编辑表单，允许配置 `line_chart`、时间范围和粒度
- `frontend/src/windows/ShowcaseWindow.vue`
  - 复用共享渲染组件，保持 Viewer 一致性
- `frontend/src/i18n/messages/showcase.ts`
  - 补充新增文案

#### Data / Call Flow
1. 用户在 Editor 中将某个 `var` widget 设为 `line_chart`，并配置时间范围、粒度。
2. 保存 `ShowcaseConfig` 时，Go 与前端 normalize 保留 `line_chart` 配置。
3. `varpool.changed` 或相关实时更新到达时，Showcase store 将数值样本写入前端内存历史缓存。
4. 渲染折线图时，store 根据 widget 的时间范围和粒度聚合可见样本序列。
5. 共享渲染组件用 SVG 输出折线图，并显示空态、无数值态或样本不足态。

#### Interface Drafts
- `type VarWidgetMode = "auto" | "display" | "metric" | "badge" | "progress" | "line_chart" | "slider" | "switch"`
- `ShowcaseVarWidget.chart` 草案：
  - `rangeMs`
  - `bucketMs`
- 前端内存样本结构草案：
  - `timestamp`
  - `value`
- 默认配置假设：
  - 默认时间范围：`1h`
  - 默认粒度：`1m`
  - 预设时间范围：`15m` / `1h` / `6h` / `24h`
  - 预设粒度：`10s` / `1m` / `5m` / `15m` / `1h`
  - 前端 retention 上限：按最大支持时间窗裁剪，不把样本持久化到磁盘或配置

#### Error Handling and Safety
- 未识别 mode：
  - Go 与前端统一回退到 `auto`
- 未识别或非法图表配置：
  - 回退到默认 `rangeMs` / `bucketMs`
- 非数值样本：
  - 不用于绘制曲线，并在无有效序列时展示明确提示
- 样本过旧：
  - 在前端裁剪掉，不影响其它 widget
- `line_chart` 保持只读展示：
  - 不调用 `SetSimple`、`SendSimple` 或 TopicBus 发送路径

#### Performance and Testing Strategy
- 样本缓存按变量 key 存储并统一裁剪，避免同一变量被多个 widget 重复缓存。
- 聚合逻辑按当前时间窗和粒度计算，避免无上限全量保留。
- 验证策略：
  - Go：`go test ./... -count=1`
  - 前端定向测试：为 store 的样本裁剪/聚合和共享渲染组件补 Vitest
  - 前端构建：`npm run build`
  - 手工验证：运行 Wails 后检查 Editor / Viewer 的折线图显示和控件切换

#### Extensibility Design Points
- 后续如果需要面积图、柱状图或更丰富趋势展示，优先扩展共享图表配置，而不是新增新的 widget kind。
- 将样本聚合逻辑放在 store 或独立 helper 中，避免组件内直接散落数值处理。
- `line_chart` 仅作为显式模式，不改 `auto`，为后续更多展示模式留出空间。

#### Issue List
- 若用户后续要求自然日选择器或更长历史跨度，需要新增后端历史数据源，不能在本方案内强行扩展。

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 Showcase 已支持：
  - `topic_button`
  - `var` 的 `auto/display/metric/badge/progress/slider/switch`
  - `columns` / `canvas_percent`
  - Center / Editor / Viewer 分离
- 当前不足：
  - 没有趋势类展示模式
  - `ShowcaseWidgetCardContent.vue` 只有单值展示逻辑
  - 当前 requirement/spec 与本次图表需求冲突，必须同步更新
- 本轮目标：
  - 为 Showcase 增加轻量 `line_chart` 模式
  - 支持时间范围和粒度配置
  - 数据点只保留在前端内存，不做后端历史持久化

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验路由与 impact
- Canonical destination:
  - 长期行为真相 -> `docs/requirements/showcase-display-widgets.md`
  - 长期技术契约 -> `docs/specs/showcase-display-widgets.md`
  - 执行控制面 -> worktree 根 `plan.md`
  - 完成结果 -> `docs/change/YYYY-MM-DD_win-showcase-line-chart.md`
- Requirements impact: `updated`
- Specs impact: `updated`
- Related requirements:
  - `docs/requirements/showcase-display-widgets.md`
- Related specs:
  - `docs/specs/showcase-display-widgets.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
- Related lessons:
  - `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets\docs\lessons\frontend-worktree-wailsjs-missing.md`

#### Executable Task List
- [x] `LCH-1` 更新 Showcase requirement/spec，纳入折线图和前端内存采样边界
- [x] `LCH-2` 扩展 Go / 前端配置模型与 normalize，支持 `line_chart` 与图表配置
- [x] `LCH-3` 在 Showcase store 中实现内存样本采集、裁剪和粒度聚合
- [x] `LCH-4` 接入共享折线图渲染与 Editor / Viewer 配置控件
- [x] `LCH-5` 补测试并完成 build / review 验证
- [x] `LCH-6` 归档 `docs/change` 并同步控制面索引

#### Task Details
##### `LCH-1` - 更新稳定 requirement/spec
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win\plan.md`
- Goal:
  - 将 `line_chart`、时间范围、粒度和“样本不持久化”写入稳定文档
- Files / Modules:
  - `docs/requirements/showcase-display-widgets.md`
  - `docs/specs/showcase-display-widgets.md`
- Write Set:
  - `docs/requirements/showcase-display-widgets.md`
  - `docs/specs/showcase-display-widgets.md`
- Acceptance:
  - requirement/spec 不再与本轮需求冲突
  - 长期边界清楚写明：仅 session 内内存历史、无后端历史查询
- Test Points:
  - impact 记录准确，交叉引用正确
- Rollback:
  - 回退上述两份文档到当前主线版本

##### `LCH-2` - 扩展持久化模型与 normalize
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win\plan.md`
- Goal:
  - 在 Go 与前端类型中支持 `line_chart` 模式和图表配置持久化
- Files / Modules:
  - `app_showcase.go`
  - `app_showcase_test.go`
  - `frontend/src/stores/showcase.ts`
- Write Set:
  - `app_showcase.go`
  - `app_showcase_test.go`
  - `frontend/src/stores/showcase.ts`
- Acceptance:
  - `line_chart` 配置可保存、加载、normalize
  - 旧配置与未知配置安全回退
- Test Points:
  - `go test ./... -count=1`
  - 定向前端类型 / normalize 测试
- Rollback:
  - 回退 schema 和 normalize 改动

##### `LCH-3` - 实现前端内存样本采集与聚合
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win\plan.md`
- Goal:
  - 基于现有快照流维护折线图样本缓存，并输出时间窗内的聚合序列
- Files / Modules:
  - `frontend/src/stores/showcase.ts`
  - 如有必要，新增 `frontend/src/lib/showcaseChart.ts`
- Write Set:
  - `frontend/src/stores/showcase.ts`
  - `frontend/src/lib/showcaseChart.ts`
- Acceptance:
  - 样本只保留在内存
  - 样本缓存有上限和裁剪
  - 时间窗和粒度切换能得到稳定可见序列
- Test Points:
  - Vitest 覆盖样本裁剪、bucket 聚合、空态和非法配置回退
- Rollback:
  - 回退样本缓存与 helper 改动

##### `LCH-4` - 接入共享折线图渲染和配置控件
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win\plan.md`
- Goal:
  - 在 Editor / Viewer 中统一渲染折线图，并提供时间范围和粒度控件
- Files / Modules:
  - `frontend/src/components/showcase/ShowcaseWidgetCardContent.vue`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/ShowcaseWindow.vue`
  - `frontend/src/i18n/messages/showcase.ts`
- Write Set:
  - `frontend/src/components/showcase/*`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/ShowcaseWindow.vue`
  - `frontend/src/i18n/messages/showcase.ts`
- Acceptance:
  - Editor 可选 `line_chart`
  - Viewer 与 Editor 预览一致
  - 时间范围和粒度控件可用
  - 布局在 `columns` / `canvas_percent` 下不回退
- Test Points:
  - 定向组件测试
  - `npm run build`
- Rollback:
  - 回退共享渲染组件与页面表单改动

##### `LCH-5` - 验证与 Code Review
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win\plan.md`
- Goal:
  - 执行本轮自动化验证，并按 Stage 3.3 清单做 code review
- Files / Modules:
  - 限于 `LCH-1` ~ `LCH-4` 的写集
- Write Set:
  - 不新增写集；若 review 暴露问题，仅回写既有授权文件
- Acceptance:
  - review 清单完整
  - build / tests 给出明确结论
- Test Points:
  - `go test ./... -count=1`
  - `cd frontend && npm run test -- <targeted files>`
  - `cd frontend && npm run build`
  - 如运行会话可用，再做 `chrome-devtools` 冒烟
- Rollback:
  - 若 review 不通过，返回对应任务修正后重新验证

##### `LCH-6` - 归档 change 与控制面同步
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets-win\plan.md`
- Goal:
  - 使用 `$m-docs` 记录实现结果、测试、影响和回滚，并在控制面 worktree 完成归档同步
- Files / Modules:
  - `docs/change/YYYY-MM-DD_win-showcase-line-chart.md`
  - `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets\docs/change/*`
  - `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets\docs/plan/*`
  - `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets\plan.md`
- Write Set:
  - `docs/change/*`
  - 控制面 worktree 的 `docs/change/*`、`docs/plan/*`、`plan.md`
- Acceptance:
  - change 文档覆盖任务映射、测试、权衡、回滚
  - 控制面索引与历史入口同步完成
- Test Points:
  - 文档路径和引用有效
  - 控制面索引更新符合逆序追加规则
- Rollback:
  - 删除本轮 change 归档并回退控制面索引改动

#### Dependencies
- `LCH-1` -> `LCH-2` -> `LCH-3` -> `LCH-4` -> `LCH-5` -> `LCH-6`

#### Risks and Notes
- 本轮的核心实现假设是：
  - 图表配置会持久化到 Showcase 配置
  - 图表样本只保留在前端内存
  - “日期”解释为相对时间窗，不是任意历史日期查询
- 如果你希望时间范围和粒度是自由输入，而不是稳定预设，需要在 3.1 重新确认校验策略和 UI 复杂度。
- 当前落地语义是：
  - widget 对话框保存默认 `rangeMs` / `bucketMs` 到 Showcase 配置
  - 卡片内 `Range` / `Granularity` 仅调节当前显示窗口，不写回配置

#### Parallelism Assessment
- 本轮不派发子Agent。
- 原因：
  - `app_showcase.go`、`frontend/src/stores/showcase.ts`、共享渲染组件、`Showcase.vue`、`ShowcaseWindow.vue` 在同一条高耦合关键路径上
  - 图表配置、样本缓存和 UI 渲染需要统一接口，拆分并行会放大 schema / UI 漂移风险
  - 主Agent 需要连续统筹 requirement/spec 更新、实现假设、测试策略和 Stage 4 归档

### Stage 3.2 - Implementation Record
- `LCH-1`
  - 已更新 `docs/requirements/showcase-display-widgets.md` 与 `docs/specs/showcase-display-widgets.md`，明确 `line_chart`、时间范围/粒度、仅前端 session 内存采样、无后端历史查询。
- `LCH-2`
  - 已更新 `app_showcase.go`、`app_showcase_test.go`、`frontend/src/stores/showcase.ts`，扩展 `line_chart` 模式、`chart` 配置与 normalize。
- `LCH-3`
  - 已新增 `frontend/src/lib/showcaseChart.ts` 与测试，集中处理图表配置归一化、样本 retention、bucket 聚合。
  - 已在 `frontend/src/stores/showcase.ts` 基于现有 `upsertSnapshot()` 采集内存样本，并按变量 key 复用历史。
- `LCH-4`
  - 已更新 `frontend/src/components/showcase/ShowcaseWidgetCardContent.vue`、`frontend/src/pages/Showcase.vue`、`frontend/src/i18n/messages/showcase.ts`。
  - Widget 对话框可配置并持久化默认图表范围/粒度；卡片内可临时调整显示窗口与粒度。

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - `line_chart` 已接入 Editor / Viewer，共享渲染与降级态完整。
- 架构合理性：通过
  - 选择扩展 `var` mode 而非新增 widget kind；聚合逻辑集中在 `frontend/src/lib/showcaseChart.ts`。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 无新增后端 I/O；样本序列按 `24h` 和 `4096` 上限裁剪；聚合仅基于当前 widget 可见窗口计算。
- 可读性与一致性：通过
  - Go / 前端 normalize 语义对齐，组件与 store 的职责边界清楚。
- 可扩展性与配置化：通过
  - 图表配置独立为 `chart { rangeMs, bucketMs }`，后续扩更多趋势模式时可复用。
- 稳定性与安全：通过
  - 未识别 mode 回退 `auto`；非法范围/粒度回退安全默认值；非数值/无样本/样本不足均明确降级。
- 测试覆盖情况：通过
  - `$env:GOWORK='off'; go test ./... -count=1`
  - `npm run test -- src/lib/showcaseChart.test.ts src/components/showcase/ShowcaseWidgetCardContent.test.ts`
  - `npm run build`
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 未派发子Agent。

### Stage 4 - Archive Record
- 使用 `$m-docs` 完成 change 路由、requirements/specs impact 复核与索引同步。
- Requirements impact: `updated`
- Specs impact: `updated`
- Lessons impact: `none`
- 相关归档：
  - `docs/change/2026-04-09_win-showcase-line-chart.md`
  - `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets\docs\change\2026-04-09_win-showcase-line-chart.md`
  - `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets\docs\plan\plan_archive_2026-04-09_win-showcase-line-chart.md`

#### Issue List
- 无新增阻塞；实现、验证和归档均已完成，等待你决定是否结束当前 workflow。

阻塞：否
3.2 / 3.3 / 4 已完成
等待是否结束 workflow
