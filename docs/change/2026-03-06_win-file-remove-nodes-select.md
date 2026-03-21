# 2026-03-06 - Win：移除左侧 Nodes 顶部 Select 按钮

## 变更背景 / 目标
- 背景：
  - 左侧 `Nodes` 顶部存在 `Select` 按钮（浏览节点树选择入口）。
- 目标：
  1. 移除该 `Select` 按钮；
  2. 清理按钮对应的选择状态、函数和弹窗实现；
  3. 保留 `Add Remote Node` 弹窗内的 `Select` 功能不受影响。

## 具体变更内容（新增 / 修改 / 删除）

### 修改
- `frontend/src/pages/File.vue`
  - 删除 `browserNodePickerOpen` / `browserPickerTargetId` 状态；
  - 删除 `openBrowserNodePicker` / `onBrowserNodePicked` / `confirmBrowserNodePicker`；
  - 删除左侧 `Nodes` 顶部 `Select` 按钮；
  - 删除 `Select Node` 对应 Overlay 弹窗；
  - 左侧顶部操作区恢复为仅保留 `Add` 图标按钮。

### 保留
- `Add Remote Node` 弹窗中的 `Node ID + Select`（树选择回填）保持不变。

### 新增
- `docs/change/2026-03-06_win-file-remove-nodes-select.md`（本文档）。

### 删除
- 无文件删除。

## 对应 plan 任务映射
- `REMOVE-NODES-SELECT-1`：已完成。
- `REMOVE-NODES-SELECT-2`：已完成。
- `REMOVE-NODES-SELECT-3`：已完成。

## 关键设计决策与权衡（性能 / 扩展性）
- 采用“最小删除面”策略，仅删除 `Nodes` 顶部选择入口链路；
- 不触碰 Offer/AddNode 其它选择器，避免引入非目标回归；
- 删除后减少一套弹窗状态管理，页面状态更简单。

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
- 结果：失败（环境现状问题），`wailsjs` 生成文件缺失导致 `Home.vue` 依赖解析失败；非本次改动引入。

### 手工验证建议
1. 打开 `File` 页左侧 `Nodes` 面板，确认顶部无 `Select` 按钮，仅剩 `Add` 图标。
2. 点击 `Add` 打开 `Add Remote Node`，确认 `Node ID` 旁 `Select` 仍可用并可回填。

## Code Review 结论（阶段 3.3）
- 需求覆盖：通过。
- 架构合理性：通过（仅 UI 删除性改动）。
- 性能风险：通过（减少状态与渲染分支）。
- 可读性与一致性：通过（移除冗余分支，逻辑更聚焦）。
- 可扩展性与配置化：通过（未破坏现有选择组件复用链路）。
- 稳定性与安全：通过（未改协议与后端调用）。
- 测试覆盖情况：部分通过（Go 测试通过；前端 build 受环境依赖缺失阻塞）。

## 潜在影响与回滚方案

### 潜在影响
- 失去左侧快速树选切换浏览节点入口（符合本次需求）。

### 回滚方案
- 回滚以下文件可完整撤销：
  - `frontend/src/pages/File.vue`
  - `todo.md`
  - `docs/change/2026-03-06_win-file-remove-nodes-select.md`
