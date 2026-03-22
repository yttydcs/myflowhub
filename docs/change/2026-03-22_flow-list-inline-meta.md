# 2026-03-22 Flow 列表同排与时间弱化

## 变更背景 / 目标

- 背景：上一轮 `Flow` 列表收敛后，名称与更新时间被放进同一个灰色文本段落里，名称的主视觉层级被削弱，同时左侧信息区与右侧按钮区的主行观感仍然不够稳定。
- 目标：恢复名称的主文本层级，让更新时间保持灰色弱化，并优化列表项左右布局，使名称区与按钮区在正常宽度下更自然地保持同排。

## 文档治理与影响检查

- Requirements impact: `none`
- Specs impact: `none`
- Related requirements: `none`
- Related specs: `none`
- Lessons impact: `none`
- Index updates:
  - 已更新 `docs/change/README.md`
  - 未更新 `docs/README.md`，原因是 docs 顶层拓扑未变化
  - 未更新 `docs/plan/README.md`，原因是当前 workflow 计划仍以 worktree 根 `plan.md` 为准

## 具体变更内容（新增 / 修改 / 删除）

### 修改

- `frontend/src/pages/Flow.vue`
  - 将名称与更新时间拆分为两个独立节点。
  - 将名称恢复为主文本样式 `font-semibold`。
  - 将更新时间保持为灰色弱化文本 `text-muted-foreground`。
  - 将项目列表主行从 `items-start` 调整为 `items-center`。
  - 为左侧信息区补充 `min-w-0 flex-1`，提升与右侧按钮区的同排稳定性。
  - 为右侧按钮区补充 `justify-end`，让按钮区在宽屏下收口更整齐。
- `plan.md`
  - 更新当前 workflow 的任务状态、验证结果与 Code Review 结论。

### 未修改

- 未恢复 `Flow ID` 或 `Project ID` 列表展示。
- 未修改 `flowProjects` store、后端接口、数据结构或更新时间生成逻辑。
- 未修改部署列表、元数据弹窗和编辑器窗口。

## 对应 plan.md 任务映射

- `T1` 调整 Flow 列表项名称/时间样式与主行布局：完成。
- `T2` 执行验证与 Code Review：完成。
- `T3` 归档变更并更新索引：完成。

## 关键设计决策与权衡

- 只改模板和布局 class，不改 store。
  - 原因：本次是纯视觉微调，状态层不应承载排版责任。
  - 收益：改动集中，回滚简单，不扩大耦合面。
- 将名称与时间拆分为独立节点，而不是继续拼接单个字符串。
  - 原因：只有拆分节点，才能让名称保持主色、时间保持灰色，并为后续细化留出扩展位。
  - 收益：结构更清晰，样式职责更稳定。
- 左侧信息区使用 `flex-1` 与 `min-w-0`。
  - 原因：保证按钮区有稳定的右侧空间，同时避免长名称把主行布局顶坏。
  - 收益：宽屏同排观感更好，窄屏仍可自然换行。
- 性能说明：
  - 本次变更为纯模板与 class 调整，不新增请求、不新增循环、不新增计算、不新增 I/O。
  - 渲染复杂度与上一轮相比保持常量级。

## 测试与验证方式 / 结果

- 文本核对：
  - 检查 `frontend/src/pages/Flow.vue` 的本地项目列表模板。
  - 结果：名称与时间已拆分为独立节点，名称不再使用灰色文本。
- Vue SFC 解析：
  - 命令：使用 `@vue/compiler-sfc` 解析 `frontend/src/pages/Flow.vue`
  - 结果：通过。
- 依赖准备：
  - 命令：`npm ci`
  - 结果：通过。
- 前端构建：
  - 命令：`npm run build`
  - 结果：失败。
  - 失败原因：`src/pages/Home.vue` 仍无法解析 `../../wailsjs/go/main/App`，属于当前仓既有 `frontend/wailsjs` 生成物缺失问题，与本次列表行布局修正无关。

## Code Review 结论

- 需求覆盖：通过。名称已恢复主文本层级，时间保持灰色，列表继续隐藏技术 ID。
- 架构合理性：通过。改动限定在页面模板层，没有扩散到 store 或接口层。
- 性能风险：通过。纯渲染层微调，无额外请求、计算或 I/O。
- 可读性与一致性：通过。名称和时间语义分离，布局类名比单行拼接更清晰。
- 可扩展性与配置化：通过。后续若要单独增强时间展示，可直接在独立节点上扩展。
- 稳定性与安全：通过。未触碰业务调用、权限、存储或输入处理。
- 测试覆盖情况：通过。目标文件语法校验和静态核对完成；完整构建受既有环境问题阻塞，已记录残余风险。
- 子Agent治理与审计：通过。本轮未使用子Agent。

## 潜在影响与回滚方案

- 潜在影响：
  - 本地项目列表的名称会更醒目，时间弱化为次级信息。
  - 长名称在狭窄宽度下可能与时间分成两行，但按钮区仍会保持可用。
- 回滚方案：
  - 回退 `frontend/src/pages/Flow.vue`
  - 回退 `plan.md`
  - 删除 `docs/change/2026-03-22_flow-list-inline-meta.md`
  - 回退 `docs/change/README.md`

## 子Agent执行轨迹

- `T1` → `主Agent` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-list-inline-meta` → `frontend/src/pages/Flow.vue` → 验收通过
- `T2` → `主Agent` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-list-inline-meta` → `plan.md` → 验收通过
- `T3` → `主Agent` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-list-inline-meta` → `docs/change/2026-03-22_flow-list-inline-meta.md`, `docs/change/README.md` → 验收通过
