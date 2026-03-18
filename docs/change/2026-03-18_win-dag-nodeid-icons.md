# 2026-03-18 Win：DAG 节点 ID 自动生成 + 工具栏图标化

## 变更背景 / 目标
Flow DAG 编辑器当前存在两类体验问题：
- 新增节点需要手动输入 `Node ID`，容易重复、也增加了创建成本；
- Flow Editor 顶部与节点操作区按钮较多，文本按钮占用空间大，界面密度偏高。

本次目标：
- 新增节点时自动生成一个不重复的 `Node ID`（大小写敏感），同时仍允许用户手动修改（并检测重复）；
- 在右侧 Node Detail 允许修改 `Node ID`，并在改名时自动同步更新所有连线引用（`edges[].from/to`）；
- 将 Flow Editor 顶部工具条 + Add/Remove Node/Edge 统一改为图标按钮，并用 Tooltip 提示功能（对齐 Showcase 顶部交互风格）。

## 具体变更内容
### 1) 新增节点自动生成 Node ID
- 新增 `suggestNodeId(prefix?: string)`：扫描当前节点集合生成唯一 ID（默认 `n1/n2/...`）。
- Add Node 弹窗打开时自动预填建议 ID，用户仍可在弹窗内手动修改；最终仍由 `addNode()` 做唯一性校验。

### 2) 支持 Node ID 改名（同步更新 edges + history）
- 新增 `renameNodeId(oldId, newId)`：
  - 校验非空、去首尾空格、唯一性（大小写敏感）；
  - 改名成功后，批量替换 `edges[].from/to` 中对旧 ID 的引用；
  - 写入 history，确保 undo/redo 可回退/恢复。
- Node Detail 的 Node ID 输入框改为可编辑，`blur/Enter` 时提交改名；失败则回滚输入并 toast 提示。

### 3) Flow 工具条与节点操作按钮图标化（Tooltip）
- Flow Editor 顶部按钮改为 icon + tooltip（Refresh/New/Undo/Redo/Auto Layout/Save/Run/Status）。
- Add/Remove Node/Edge 也改为 icon + tooltip，并保留 `sr-only` 文本确保可访问性。

## 对应 plan.md 任务映射
- WIN-DAG-1：自动生成 Node ID（Add Node 预填）
- WIN-DAG-2：支持改名 Node ID（同步更新 edges + history）
- WIN-DAG-3：工具条与节点操作按钮图标化（Tooltip）
- WIN-DAG-4：最小构建与人工验收

## 关键设计决策与权衡
- **改名逻辑放在 store 而非 UI**：保证“节点改名 + edges 同步 + history”原子化，避免 UI 漏改造成图状态不一致。
- **ID 生成策略选择 `n{number}` 递增**：简单、可预期，且可通过 `prefix` 扩展未来不同命名风格。
- **大小写敏感**：保持与现有 `addNode()` / `selectNodeById()` 等一致的精确匹配语义。

## 测试与验证方式 / 结果
### 构建验证
- 在 worktree 内执行 `wails generate module` 生成 `frontend/wailsjs`（生成产物未纳入提交）
- `cd frontend && npm ci && npm run build`：通过

### 人工验收要点
- 连续点击 Add Node：弹窗内 Node ID 自动预填且不重复；
- 连接多条边后改名节点：相关边的 `from/to` 自动更新；
- undo/redo：可回退/恢复改名与边更新；
- 工具条与 Add/Remove Node/Edge：图标按钮 tooltip 正常，禁用态与行为与原一致。

## 潜在影响与回滚方案
### 潜在影响
- 改名会同步更新 edges，历史 status 视图可能暂时显示“旧 ID 的上一次运行状态”（取决于保存/运行时机），不影响实际执行。

### 回滚方案
- 直接回退本次提交；
- 或恢复 Node Detail 的 Node ID 为只读，并回退工具栏 icon 化改动。

