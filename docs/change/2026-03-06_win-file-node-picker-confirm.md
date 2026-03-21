# 2026-03-06 - Win：File 节点选择交互优化（图标 Add + Select 弹窗确认）

## 变更背景 / 目标
- 背景：
  - 左侧 `Nodes` 顶部 `Add` 为文本按钮，与其余图标操作不统一；
  - 节点树选择存在“点击即回填”风险，展开时容易误触；
  - 选择弹窗存在冗余文案与底部 `Selected target` 展示。
- 目标：
  1. 左侧 `Nodes` 顶部 `Add` 改为图标；
  2. 左侧新增 `Select` 按钮并复用树形选择；
  3. `Select Target Node` 改为“点击仅选中，Confirm 才回填”；
  4. 移除冗余文案与 `Selected target`；
  5. 右上角 `Close` 改无边框图标按钮。

## 具体变更内容（新增 / 修改 / 删除）

### 修改
- `frontend/src/pages/File.vue`
  - 左侧 `Nodes` 顶部：
    - `Add` 按钮改为 icon 按钮；
    - 新增 `Select` 按钮，打开节点树选择弹窗。
  - Offer 节点选择流程：
    - `onOfferTargetPicked` 改为只更新暂存选中值；
    - 新增 `confirmOfferTargetPicker`，点击 `Confirm` 后回填 `Target Node ID`。
  - 新增左侧节点选择流程：
    - `browserNodePickerOpen/browserPickerTargetId` 状态；
    - `openBrowserNodePicker/onBrowserNodePicked/confirmBrowserNodePicker`；
    - Confirm 后切换浏览节点；远端节点若未保存则自动加入 saved nodes。
  - 两个节点选择弹窗都改为：
    - 右上角无边框图标关闭按钮（`X`）；
    - 底部 `Cancel/Confirm` 操作区；
    - 移除 `Click a node in the tree to apply it.` 文案。

- `frontend/src/components/file/OfferNodeTreePicker.vue`
  - 移除底部 `Selected target` 文本展示，仅保留树选择主体。

### 新增
- `docs/change/2026-03-06_win-file-node-picker-confirm.md`（本文档）。

### 删除
- 无文件删除。

## 对应 plan 任务映射
- `NODE-PICKER-1`：完成（Nodes 顶部按钮区改造）。
- `NODE-PICKER-2`：完成（Offer 选择改 Confirm 回填）。
- `NODE-PICKER-3`：完成（移除文案/selected target，Close 图标化）。
- `NODE-PICKER-4`：完成（左侧 Select 弹窗确认并切换节点）。
- `NODE-PICKER-5`：完成（回归验证、Code Review、归档）。

## 关键设计决策与权衡（性能 / 扩展性）
- 误触控制优先：
  - 树节点点击仅更新“暂存选择”，由 `Confirm` 执行最终写入，避免展开树时误触回填。
- 复用优先：
  - 左侧节点选择复用 `OfferNodeTreePicker`，避免重复维护树加载逻辑。
- 性能：
  - 维持按需加载子节点策略，不增加额外轮询或全量请求。
- 扩展性：
  - 选择弹窗模式可继续复用于其他“输入 + 选择器”场景。

## 测试与验证方式 / 结果

### 后端测试
```powershell
$env:GOWORK='off'; go test ./...
```
- 结果：通过。

### 前端构建
```powershell
cd frontend
npm install
npm run build
```
- 结果：失败（环境现状问题），`wailsjs` 生成文件缺失导致模块解析失败；非本次改动引入。

### 手工验证建议
1. 打开 `File` 页面，确认左侧 `Nodes` 顶部 `Add` 为图标，且出现 `Select` 按钮。
2. 点击左侧 `Select`，在树里选择节点后点击 `Confirm`，确认当前浏览节点切换。
3. 打开 `Send Offer`，点击 `Target Node ID` 右侧 `Select`，树中点击节点不应立即回填。
4. 在弹窗点击 `Confirm` 后，`Target Node ID` 输入框回填节点 ID。
5. 校验 `Target Node ID` 为本地节点或非法输入时，发送仍给出正确提示。

## Code Review 结论（阶段 3.3）
- 需求覆盖：通过。
- 架构合理性：通过（仅 UI 层改造，复用既有树组件）。
- 性能风险：通过（无新增高频 I/O，懒加载保持不变）。
- 可读性与一致性：通过（状态与事件命名清晰，弹窗行为一致）。
- 可扩展性与配置化：通过（节点选择流程可复用）。
- 稳定性与安全：通过（回填前保留节点合法性校验）。
- 测试覆盖情况：部分通过（Go 测试通过；前端 build 受环境依赖缺失阻塞）。

## 潜在影响与回滚方案

### 潜在影响
- 新增左侧 `Select` 选择远端节点时会写入 saved nodes（避免下次丢失），可能改变节点列表顺序表现。

### 回滚方案
- 回滚以下文件可完整撤销：
  - `frontend/src/pages/File.vue`
  - `frontend/src/components/file/OfferNodeTreePicker.vue`
  - `todo.md`
  - `docs/change/2026-03-06_win-file-node-picker-confirm.md`
