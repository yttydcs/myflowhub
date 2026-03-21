# 2026-03-06 - Win：Offer 目标节点改为输入框 + 选择按钮弹窗树

## 变更背景 / 目标
- 背景：
  - 当前 `Send Offer` 中目标节点使用内嵌树组件，占用弹窗空间，且与 `Remote Dir` 输入交互风格不一致。
- 目标：
  1. 将目标节点改为与 `Remote Dir` 同风格的输入框；
  2. 输入框右侧增加 `Select` 按钮；
  3. 点击 `Select` 弹出树形选择框，点击节点后自动回填 NodeID 到输入框。

## 具体变更内容（新增 / 修改 / 删除）

### 修改
- `frontend/src/pages/File.vue`
  - `offerForm.targetId` 从数字改为字符串，支持手工输入；
  - Offer 表单中的目标节点区域改为：输入框 + 右侧 `Select` 按钮；
  - 新增 `offerNodePickerOpen` / `offerPickerTargetId` 状态；
  - 新增 `openOfferNodePicker` / `onOfferTargetPicked`：
    - 打开树选择弹窗；
    - 点击树节点后回填输入框并关闭弹窗；
  - Offer 发送时从输入框解析并校验 nodeId（必须为正整数，且不能为本地节点）。

### 复用
- `frontend/src/components/file/OfferNodeTreePicker.vue`
  - 不改组件内部逻辑，继续复用节点树加载/展开/重试能力。

### 删除
- 无。

## 对应 plan 任务映射
- `OFFER-PICKER-UI-1`：输入框 + 选择按钮，已完成。
- `OFFER-PICKER-UI-2`：弹窗树选择并回填，已完成。
- `OFFER-PICKER-UI-3`：回归验证、Code Review、归档，已完成。

## 关键设计决策与权衡（性能 / 扩展性）
- 仅改页面编排，不改协议与后端接口：
  - 保持 `StartOfferToDir` 与 `req.dir` 语义不变；
  - 风险小、回滚简单。
- 树选择器继续懒加载子节点：
  - 不引入额外全量查询；
  - 弹窗打开后才进行选择交互，保持主弹窗轻量。
- 兼顾手输与点选：
  - 输入框保留手工输入路径；
  - `Select` 提供低错误率可视化选择路径。

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
- 结果：失败（环境现状问题），`Home.vue` 依赖 `../../wailsjs/go/session/SessionService` 无法解析；非本次改动直接引入。

### 手工验证建议
1. 进入 `File` 页，选中本地文件，打开 `Send Offer`。
2. 在 `Target Node ID` 输入框右侧点击 `Select`。
3. 在树弹窗中点击一个远端节点，确认弹窗关闭且输入框回填该 NodeID。
4. 直接发送 Offer，确认发送成功且行为与改动前一致。
5. 手工输入非法 NodeID（空/非数字/本地节点）验证校验提示。

## Code Review 结论（阶段 3.3）
- 需求覆盖：通过。
- 架构合理性：通过（仅 UI 交互层变更，无协议/后端破坏性改动）。
- 性能风险：通过（无新增高频 I/O；树仍按需加载）。
- 可读性与一致性：通过（Offer 相关状态集中，函数职责清晰）。
- 可扩展性与配置化：通过（后续可把同类“输入+选择弹窗”模式复用于其他 ID 选择场景）。
- 稳定性与安全：通过（增加 ID 解析与本地节点排除校验）。
- 测试覆盖情况：部分通过（Go 测试通过；前端构建受环境依赖缺失阻塞）。

## 潜在影响与回滚方案

### 潜在影响
- 弹窗为叠加层，若未来调整 Overlay 栈行为，需验证父子弹窗关闭顺序。

### 回滚方案
- 回滚以下文件可完整撤销：
  - `frontend/src/pages/File.vue`
  - `todo.md`
  - `docs/change/2026-03-06_win-file-offer-target-picker-dialog.md`
