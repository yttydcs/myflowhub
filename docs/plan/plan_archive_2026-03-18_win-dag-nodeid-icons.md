# Plan - Win：DAG 节点 ID 自动生成 + 工具栏图标化

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`feat/win-dag-nodeid-icons`
- Worktree：`d:\project\MyFlowHub3\repo\MyFlowHub-Win\worktrees\feat-win-dag-nodeid-icons`
- Base：`main`

## 项目目标与当前状态
- 目标：
  - 新增 DAG 节点时自动生成不重复的 Node ID（大小写敏感）；
  - 允许在右侧 Node Detail 修改 Node ID，并自动同步更新 edges 引用；
  - 将 Flow Editor 顶部工具条 + Add/Remove Node/Edge 统一改为图标按钮，并使用 Tooltip 展示功能说明（风格对齐 Showcase 顶部）。
- 当前状态：
  - Add Node 需要手动输入 Node ID，且不提供自动生成；
  - Node Detail 的 Node ID 输入框 disabled，无法改名；
  - Flow Editor 顶部与 Add/Remove Node/Edge 使用文本按钮，视觉密度较高。

## 范围
- 必须：
  - 自动生成 Node ID（避免重复）并支持手动修改（重复校验，大小写敏感）；
  - 改名时自动更新所有 `edges[].from/to`（旧 ID → 新 ID）；
  - 变更应纳入 undo/redo 历史；
  - 工具条与 Add/Remove Node/Edge 改为 icon + tooltip，并保留可访问性文本（`sr-only`）。
- 可选：
  - Tooltip 文案包含快捷键提示（例如 Save: `Ctrl+S`）。
- 不做：
  - 不改协议/服务端；
  - 不调整 Flow 的执行语义与节点类型（local/exec）；
  - 不做 UI 大改版（仅对齐已有 Showcase toolbar 组件风格）。

## 可执行任务清单（Checklist）

### WIN-DAG-1 - 自动生成 Node ID（Add Node 预填）
- 目标：
  - 打开 Add Node 弹窗时，自动填入一个唯一 Node ID（默认形如 `n1/n2/...`）。
- 涉及模块 / 文件：
  - `frontend/src/stores/flow.ts`（新增 `suggestNodeId()`）
  - `frontend/src/pages/Flow.vue`（`openAddNodeDialog()` 使用建议 ID）
- 验收条件：
  - 直接点 Add 可成功新增节点；连续新增不会重复。
- 测试点：
  - 新建草稿，连续 Add Node 3 次，检查 ID 唯一；
  - 手动改成重复 ID，Add 时能提示失败。
- 回滚点：
  - 回退 `suggestNodeId()` 与对应 UI 改动。

### WIN-DAG-2 - 支持改名 Node ID（同步更新 edges + history）
- 目标：
  - 在 Node Detail 中允许编辑 Node ID；改名时自动更新所有引用该节点的边。
- 涉及模块 / 文件：
  - `frontend/src/stores/flow.ts`（新增 `renameNodeId(oldId, newId)`）
  - `frontend/src/pages/Flow.vue`（Node Detail 增加可编辑输入与提交流程）
- 验收条件：
  - 改名后 edges 的 `from/to` 被正确替换；
  - 改名会进入历史快照，undo/redo 可回退/恢复。
- 测试点：
  - 创建 3 节点 + 2 条边；改名中间节点；确认两条边同步更新；
  - undo/redo 验证改名与边更新可回退/恢复；
  - 重复 ID 被拒绝且不会破坏现有状态。
- 回滚点：
  - 回退改名逻辑；恢复 Node ID disabled。

### WIN-DAG-3 - Flow 工具条与节点操作按钮图标化（Tooltip）
- 目标：
  - 将 Flow Editor 顶部工具条与 Add/Remove Node/Edge 改为 icon 按钮，并使用 Tooltip 展示功能说明。
- 涉及模块 / 文件：
  - `frontend/src/pages/Flow.vue`
- 验收条件：
  - Tooltip 正常显示；禁用态逻辑不变；功能行为与原文本按钮一致。
- 测试点：
  - Refresh/New/Undo/Redo/AutoLayout/Save/Run/Status；
  - Add/Remove Node/Edge；
  - 键盘快捷键仍可用（如 `Ctrl+S`、`Ctrl+Z`）。
- 回滚点：
  - 回退到原文本按钮布局。

### WIN-DAG-4 - 最小构建与人工验收
- 目标：
  - 证明前端能正常构建，且核心交互可用。
- 涉及模块 / 文件：
  - `frontend/` 构建产物（不提交）
- 验收条件：
  - `frontend` 构建成功；关键交互按上述测试点通过。
- 测试点：
  - `cd frontend && npm run build`
- 回滚点：
  - 清理本地构建产物；回退相关提交。

### WIN-DAG-5 - Code Review（强制）
- 目标：
  - 审查需求覆盖、架构合理性、稳定性与可维护性。
- 涉及模块 / 文件：
  - 本 workflow 全部改动文件
- 验收条件：
  - 形成逐项通过/不通过结论。
- 测试点：
  - Review 清单齐全。
- 回滚点：
  - 若不通过，回到对应任务修订。

### WIN-DAG-6 - 归档变更（强制）
- 目标：
  - 输出可交接、可审计的变更说明。
- 涉及模块 / 文件：
  - `docs/change/2026-03-18_win-dag-nodeid-icons.md`
- 验收条件：
  - 文档包含背景/目标、具体改动、任务映射、设计权衡、测试结果、影响与回滚。
- 测试点：
  - 文档完整可复现。
- 回滚点：
  - 回退归档文档。

## 依赖关系
- `WIN-DAG-1` → `WIN-DAG-2` → `WIN-DAG-3` → `WIN-DAG-4` → `WIN-DAG-5` → `WIN-DAG-6`

## 风险与注意事项
- Node ID 改名必须同步更新 edges，否则会造成图显示/选择异常；
- commitHistory 基于 snapshot diff：需要避免在 UI 层产生“输入框显示值与 state 不一致”的状态；
- 图标按钮需保留 `sr-only` 文本，避免可访问性退化。

