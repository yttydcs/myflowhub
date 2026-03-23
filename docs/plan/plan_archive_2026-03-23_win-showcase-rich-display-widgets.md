# Plan - MyFlowHub-Win Showcase Rich Display Widgets

## Workflow Information

- Repo: `MyFlowHub-Win`
- Branch: `feat/win-showcase-rich-components`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-showcase-rich-components`
- Current Stage: `4 归档变更（已完成，等待 workflow 结束确认）`

## Stage Records

### Initialization

- `guide.md`: 已阅读，遵守 commit 中文说明、worktree 必须位于 `D:\project\MyFlowHub3\worktrees\`、优先使用 chrome-devtools 做界面验证的约束。
- base/worktree confirmation:
  - 控制面仓库：`D:\project\MyFlowHub3`，仅用于 workflow 编排与归档。
  - 实现仓库：`MyFlowHub-Win`。
  - 活跃执行 worktree：`D:\project\MyFlowHub3\worktrees\feat-win-showcase-rich-components`。
  - 参考文档：`D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`、`D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\topicbus.md`，本轮不计划修改。

### Stage 1 - Requirements Analysis

#### Goal

- 在现有 Win Showcase 基础上补充更丰富的展示组件，让 Screen 在不增加理解成本的前提下具备更强的“状态面板”表达力。
- 保持当前简洁风格，不把 Showcase 扩展成复杂仪表盘系统。

#### Scope

##### Must

- 在现有 `topic_button` / `var` 体系内扩展更丰富的展示能力，不引入新协议。
- 为 `var` widget 增加至少三种显式展示模式：
  - `metric`
  - `badge`
  - `progress`
- 保持既有 `display`、`slider`、`switch`、`topic_button` 行为兼容。
- Editor 预览与 Viewer 渲染结果一致，且同时支持 `columns` 与 `canvas_percent` 两种布局。
- 非法值、空值、非数值进度等异常场景必须优雅降级，不得破坏布局。

##### Optional

- 为展示模式增加轻量的语义色和辅助元信息，只要不引入视觉噪音。
- 适度复用当前 slider 范围配置，为 `progress` 提供上下界。

##### Out of Scope

- 不新增 Server / SDK / Proto 能力，也不改 TopicBus / VarStore 协议。
- 不做历史曲线、趋势图、聚合统计或多变量组合计算。
- 不重做 Showcase Center、Screen 布局模型或多窗口同步机制。

#### Use Cases

- 操作人员希望用更醒目的方式查看单个变量的当前值，而不是只看一行普通文本。
- 操作人员希望用紧凑 badge 快速判断状态值或布尔值。
- 操作人员希望用进度条查看数值变量在目标区间中的位置。
- 配置人员希望在编辑器里直接选择合适的展示模式，并在 Viewer 中得到一致呈现。

#### Functional Requirements

1. Widget 编辑表单必须允许为 `var` widget 选择新的展示模式。
2. `metric` 模式必须以更突出的数字/文本主值方式展示当前变量值。
3. `badge` 模式必须以紧凑状态标签方式展示当前变量值，并提供稳定的语义色规则。
4. `progress` 模式必须基于已配置区间展示当前数值进度，并在无法解析数值时回退到明确提示。
5. `auto` 模式必须保持现有推断规则，避免老配置出现静默行为变化。
6. 所有新增模式必须沿用现有订阅与刷新机制，不新增额外数据源。
7. 配置保存后重新打开 Editor / Viewer，新增模式必须稳定保留并正确渲染。

#### Non-functional Requirements

- 简洁性：
  - 保持当前卡片式、低噪音视觉风格。
  - 不引入重装饰、复杂图例或大段说明文字。
- 兼容性：
  - 旧配置无需迁移脚本即可继续加载。
  - 未识别 mode 必须回退到安全默认值。
- 可维护性：
  - Editor 与 Viewer 不应维护两套分叉的展示规则。
  - 新增模式应便于未来继续扩展，而不是继续堆积条件分支。
- 性能：
  - 不增加额外订阅。
  - 不引入高频保存、额外 I/O 或无意义的重复解析。

#### Inputs / Outputs

- Inputs:
  - 用户在 Showcase Editor 中选择的 `var` widget 展示模式与已有范围配置。
  - VarStore 当前订阅值快照。
- Outputs:
  - 持久化后的 Showcase 配置。
  - Editor / Viewer 中一致的 widget 展示结果。

#### Edge Cases

- `progress` 模式收到空值或非数值字符串。
- `progress` 的上下界非法或相等。
- `badge` 模式收到未知字符串、空字符串或布尔值。
- `metric` 模式在值缺失时不能出现布局塌陷。
- 旧配置里保存了未知 `mode`。

#### Acceptance Criteria

1. 用户可以在 Editor 中创建或编辑 `metric`、`badge`、`progress` 三种 `var` widget。
2. 保存后重新加载配置，三种模式都能被 Go 和前端正确规范化并保留。
3. Viewer 在 `columns` 与 `canvas_percent` 下都能正确渲染三种模式。
4. `progress` 模式在数值异常时显示明确降级态，不报错、不空白。
5. `npm run build` 与 `go test ./... -count=1` 通过。

#### Risks

- 新模式如果直接在 Editor 与 Viewer 中各自追加分支，后续易发生视觉和逻辑漂移。
- `progress` 复用 slider 配置时要明确哪些字段是“显示范围”而不是“交互行为”。
- badge 语义色若规则不稳定，会让同一值在不同窗口中呈现不一致。

#### Issue List

- 无

### Stage 2 - Architecture Design

#### Overall Solution

- 继续沿用现有 ShowcaseConfig / VarStore 数据模型，不引入新 widget kind。
- 将 `ShowcaseVarWidget.mode` 扩展为：
  - `auto`
  - `display`
  - `metric`
  - `badge`
  - `progress`
  - `slider`
  - `switch`
- 在 Go 与前端 store 中同时扩展 mode normalize，保证持久化兼容。
- 前端新增 Showcase 专用共享渲染组件，集中承载 var / topic widget 的卡片内容与新模式展示逻辑，再由 Editor / Viewer 复用。

#### Alternatives Considered

- 新增多个 widget kind（如 `var_metric` / `var_progress`）：
  - 放弃。会扩大 schema 和表单复杂度，且与现有 `var` 订阅逻辑重复。
- 只在 Viewer 做样式增强，不改配置模型：
  - 放弃。用户无法显式选择想要的展示组件，Editor / Viewer 会失配。
- 纯 CSS 改造现有 `display`：
  - 放弃。无法覆盖 badge / progress 等明确的语义差异。

#### Module Responsibilities

- `app_showcase.go`
  - 扩展 mode normalize，保持未知值回退和旧配置兼容。
- `app_showcase_test.go`
  - 补 mode normalize / 默认值 / 兼容性回归测试。
- `frontend/src/stores/showcase.ts`
  - 扩展类型定义、mode normalize、值格式化与数值/状态辅助判断。
- `frontend/src/components/showcase/*`
  - 新增共享 widget 渲染组件，统一 Editor / Viewer 的展示行为。
- `frontend/src/pages/Showcase.vue`
  - 扩展 widget 对话框模式选项，接入共享渲染组件。
- `frontend/src/windows/ShowcaseWindow.vue`
  - 复用共享渲染组件，保持与 Editor 预览一致。

#### Data / Call Flow

1. 用户在 Editor 中为 `var` widget 选择显示模式。
2. Editor 将模式与现有配置写入 screen draft。
3. 保存 draft 后，Go 执行 normalize 并持久化。
4. Viewer / Editor 收到 `showcase.config_changed` 后 reload 现有配置。
5. 共享渲染组件根据当前值和 mode 统一输出对应卡片形态。

#### Interface Drafts

- `type VarWidgetMode = "auto" | "display" | "metric" | "badge" | "progress" | "slider" | "switch"`
- 共享渲染组件输入草案：
  - `widget`
  - `surface: "editor" | "viewer"`
  - `connected`
  - `busy`
  - `selfNodeId`
  - 现有 send / switch / slider 行为透传
- `progress` 复用 `widget.var.slider.min/max` 作为显示区间；`step/throttleMs` 仅在 `slider` 交互模式下生效。

#### Error Handling and Safety

- 未识别 mode：
  - Go 与前端都回退为 `auto`。
- `progress` 无法解析数值：
  - 渲染为空态说明，不执行写操作。
- display-only 模式：
  - 不触发 `SetSimple` / `SendSimple`。
- 保持 `slider` / `switch` / `topic_button` 的发送校验与 ready-check 不变。

#### Performance and Testing Strategy

- 共享渲染组件复用同一套模式判断和展示逻辑，减少双份分支维护。
- 避免新增 watcher 或订阅；继续复用现有 `enter/leave` 生命周期。
- 验证策略：
  - Go：`go test ./... -count=1`
  - 前端：`npm run build`
  - 手工 / 冒烟：Editor 创建三种新模式，保存后在 Viewer 中验证一致性。

#### Extensibility Design Points

- 后续新增展示模式时，优先扩 mode enum 和共享渲染组件，而不是复制新的页面分支。
- 若未来需要更丰富的视觉参数，可在 `ShowcaseVarWidget` 下追加展示配置，而不必新增新的 widget kind。

#### Issue List

- 无

### Stage 3.1 - Planning

#### Project Goal and Current State

- 当前 Showcase 已支持：
  - `topic_button`
  - `var` 的 `auto/display/slider/switch`
  - `columns` / `canvas_percent`
  - Center / Editor / Viewer 分离
- 当前不足：
  - 纯展示组件只有单行 `display`
  - Editor 与 Viewer 在 widget 卡片上存在重复渲染逻辑
  - Showcase 缺少长期 requirements/specs 文档来约束新增展示模式

#### Docs Governance Routing Decision

- 使用 `$docs-governor` 校验计划文档路由和 requirements/specs 影响。
- Canonical destination:
  - 长期行为真相 -> `docs/requirements/showcase-display-widgets.md`
  - 长期技术约束 -> `docs/specs/showcase-display-widgets.md`
  - 执行控制面 -> worktree 根 `plan.md`
  - 完成结果 -> `docs/change/2026-03-23_win-showcase-rich-display-widgets.md`
- Requirements impact: `add`
- Specs impact: `add`
- Related requirements:
  - `docs/requirements/showcase-display-widgets.md`
- Related specs:
  - `docs/specs/showcase-display-widgets.md`
- Reference-only specs:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\topicbus.md`

#### Executable Task List

- [x] `SHW-1` 补齐 Showcase 展示组件的稳定 requirements/specs 与索引
- [x] `SHW-2` 扩展 Go / store 的展示模式 schema 与辅助判断
- [x] `SHW-3` 为 Editor / Viewer 接入共享渲染组件和新展示模式
- [x] `SHW-4` 执行验证并完成 Code Review
- [x] `SHW-5` 归档 `docs/change` 并更新索引

#### Task Details

##### `SHW-1` - 补齐稳定 requirements/specs 与索引

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-showcase-rich-components`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-showcase-rich-components\plan.md`
- Goal:
  - 为 Showcase 展示组件增强建立长期 requirements/specs 真相，并更新 `docs/requirements/README.md` / `docs/specs/README.md`。
- Files / Modules:
  - `docs/requirements/showcase-display-widgets.md`
  - `docs/specs/showcase-display-widgets.md`
  - `docs/requirements/README.md`
  - `docs/specs/README.md`
- Write Set:
  - `docs/requirements/*`
  - `docs/specs/*`
- Acceptance:
  - requirement/spec 文档完整记录目标、范围、验收、技术契约和错误边界
  - 索引可导航到新增文档
- Test Points:
  - 文档路径、交叉引用和 impact 记录正确
- Rollback:
  - 删除新增 requirement/spec 并回退对应索引修改

##### `SHW-2` - 扩展展示模式 schema 与 store 辅助判断

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-showcase-rich-components`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-showcase-rich-components\plan.md`
- Goal:
  - 在 Go 与前端 store 中支持 `metric` / `badge` / `progress` 三种模式，并补齐数值/状态解析辅助逻辑。
- Files / Modules:
  - `app_showcase.go`
  - `app_showcase_test.go`
  - `frontend/src/stores/showcase.ts`
- Write Set:
  - `app_showcase.go`
  - `app_showcase_test.go`
  - `frontend/src/stores/showcase.ts`
- Acceptance:
  - 新 mode 可保存、加载、normalize
  - 旧配置和未知 mode 兼容
  - 新辅助逻辑不改变现有 `auto` 推断语义
- Test Points:
  - `go test ./... -count=1`
  - TypeScript 构建验证
- Rollback:
  - 回退 schema / normalize / helper 改动

##### `SHW-3` - 接入共享渲染组件与新展示模式

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-showcase-rich-components`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-showcase-rich-components\plan.md`
- Goal:
  - 将新展示模式接入 Editor 预览和 Viewer，并保持简洁、一致的 widget 卡片视觉。
- Files / Modules:
  - `frontend/src/components/showcase/*`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/ShowcaseWindow.vue`
- Write Set:
  - `frontend/src/components/showcase/*`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/ShowcaseWindow.vue`
- Acceptance:
  - Editor 对话框可选择新模式
  - Editor / Viewer 渲染结果一致
  - `columns` / `canvas_percent` 下都不出现布局破坏
- Test Points:
  - `frontend/npm run build`
  - 手工检查三种模式在 Editor / Viewer 的显示与保存
- Rollback:
  - 回退共享渲染组件与页面接入

##### `SHW-4` - 验证与 Code Review

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-showcase-rich-components`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-showcase-rich-components\plan.md`
- Goal:
  - 执行本轮构建 / 测试，并按 workflow 清单做 code review。
- Files / Modules:
  - 本轮所有改动文件
- Write Set:
  - 限于 `SHW-1` ~ `SHW-3` 已授权写集
- Acceptance:
  - review 清单结论完整
  - 发现问题时返回对应任务修正
- Test Points:
  - `go test ./... -count=1`
  - `cd frontend && npm run build`
  - 如可行，`chrome-devtools` 冒烟验证
- Rollback:
  - 若 review 不通过，回到对应任务修正后重新验证

##### `SHW-5` - 归档 change 与索引

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-showcase-rich-components`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-showcase-rich-components\plan.md`
- Goal:
  - 使用 `$docs-governor` 记录本轮实现结果、impact、验证和回滚，并更新 `docs/change/README.md`。
- Files / Modules:
  - `docs/change/2026-03-23_win-showcase-rich-display-widgets.md`
  - `docs/change/README.md`
- Write Set:
  - `docs/change/*`
- Acceptance:
  - change 文档覆盖任务映射、验证结果、权衡与回滚
  - 索引已更新
- Test Points:
  - 文档路径、引用与 impact 结论一致
- Rollback:
  - 删除本轮 change 并回退索引修改

#### Dependencies

- `SHW-1` -> `SHW-2` -> `SHW-3` -> `SHW-4` -> `SHW-5`

#### Risks and Notes

- `progress` 需要明确“显示范围”与“交互 slider 配置”的边界，避免误导用户。
- 共享渲染组件不能吞掉 Editor 的拖拽、右键和 canvas resize 钩子。
- 本轮只做“更丰富但仍然简洁”的单值展示，不扩成图表系统。

#### Parallelism Assessment

- 本轮不派发子Agent。
- 原因：
  - `app_showcase.go`、`frontend/src/stores/showcase.ts`、共享渲染组件、`Showcase.vue`、`ShowcaseWindow.vue` 处于同一条高耦合关键路径。
  - 任务之间的 write set 难以彻底解耦，提前并行会放大 schema / UI 接口漂移风险。
  - 主Agent 需要持续统筹 Stage 1/2 决策、计划确认、代码集成与最终验收。

#### Issue List

- 无

### Stage 3.3 - Code Review

- 需求覆盖：通过
  - `metric` / `badge` / `progress` 已覆盖 Editor 配置、Go normalize、Viewer 渲染与兼容性。
- 架构合理性：通过
  - 沿用现有 `var` widget 模型，只扩 mode enum，并通过共享渲染组件避免双份逻辑。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 未新增订阅、未新增高频保存；新增计算均为组件内轻量字符串 / 数值派生。
- 可读性与一致性：通过
  - Editor / Viewer 的 widget 主体已收敛为共享组件，减少分叉。
- 可扩展性与配置化：通过
  - 后续新增 display-only mode 可继续复用同一 contract；未引入硬编码环境值。
- 稳定性与安全：通过
  - display-only 模式不触发写操作；未知 mode 仍回退为 `auto`。
- 测试覆盖情况：通过
  - Go 回归测试补齐；前端生产构建通过。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子Agent，关键路径由主Agent 完成并复核。

### Stage 4 - Change Archive

- Archive Path:
  - `docs/change/2026-03-23_win-showcase-rich-display-widgets.md`
- Requirements impact:
  - `updated`
- Specs impact:
  - `updated`
- Related requirements:
  - `docs/requirements/showcase-display-widgets.md`
- Related specs:
  - `docs/specs/showcase-display-widgets.md`
- Lessons needed:
  - `none`
- Index updates:
  - `docs/requirements/README.md`
  - `docs/specs/README.md`
  - `docs/change/README.md`

## Execution Results Summary

- `SHW-1`：已完成。新增 Showcase 展示组件的 requirements/specs，并更新索引。
- `SHW-2`：已完成。Go 与前端 store 已支持 `metric` / `badge` / `progress`。
- `SHW-3`：已完成。Editor / Viewer 已接入共享渲染组件，展示逻辑统一。
- `SHW-4`：已完成。`$env:GOWORK='off'; go test ./... -count=1`、`npm run build` 通过。
- `SHW-5`：已完成。变更已归档到 `docs/change/2026-03-23_win-showcase-rich-display-widgets.md`。

阻塞：否
已完成 Stage 4，等待用户确认是否结束 workflow
