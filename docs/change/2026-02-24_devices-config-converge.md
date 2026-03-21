# 2026-02-24 Devices 集成 Config 编辑并移除 Management

## 变更背景 / 目标
当前前端同时存在 `Devices` 与 `Management` 两个入口，能力存在重复（尤其是节点 Config 的查看与编辑）。本次变更将 Config 编辑能力收敛到 `Devices`，并移除 `Management`，同时按信息架构调整导航位置。

目标：
1) 在 `Devices` 的节点列表行右侧提供“编辑（Config）”入口，弹窗查看/编辑该节点 Config；
2) 删除 `Management` 页面、导航入口与路由，避免重复维护；
3) 将 `Devices` 移动到左侧导航 `Operations` 分组，并放在 `File Console` 上方。

## 具体变更内容

### 新增
- `frontend/src/pages/Devices.vue`
  - 节点行右侧新增 `Edit` 按钮（`@click.stop`，避免触发行点击打开 NodeInfo）。
  - 新增 Config Overlay（全屏遮罩 + 中央面板），展示 Config 列表并支持 `Refresh / Close`。
  - 新增二级 Overlay：单条 Config 编辑（Key 只读、Value 可编辑、Save 写回）。

### 修改
- `frontend/src/layout/AppShell.vue`
  - 移除导航项 `Management`。
  - 将 `Devices` 从 `Session` 分组移动到 `Operations` 分组，并放在 `File Console` 之前（`flatNav` 顺序随 `navGroups` 自动同步）。
- `frontend/src/router/index.ts`
  - 移除 `/management` 路由与页面 import。

### 删除
- `frontend/src/pages/Management.vue`

## Plan.md 任务映射
- T2：Devices：新增 Config 编辑入口（行右侧按钮 + Config Overlay）
- T3：移除 Management 模块（路由 + 导航 + 页面文件）
- T4：导航重排：Devices 移到 Operations 并位于 File Console 上方
- T5：验收与回归（构建 + 冒烟）

## 关键设计决策与权衡
1) **复用 `useManagementStore` 的 Config 逻辑**
   - `Devices` 直接复用 `frontend/src/stores/management.ts` 的 `selectNode/refreshConfig/setConfig`，避免重复实现 bindings 调用、状态管理与并发时序保护（`selectedNodeId` guard）。
2) **Overlay 统一交互模型**
   - Config 采用现有 `Overlay` 组件；二级编辑弹窗同样使用 `Overlay`，确保 Esc 仅关闭最上层、Backdrop 行为一致。
3) **性能权衡**
   - Config value 仍按 key 逐条 `ConfigGetSimple` 并发加载（沿用原 `Management` 行为）。当 keys 很多时可能出现瞬时请求峰值；如后续出现卡顿/压力，可再引入并发上限或搜索过滤（需回到计划阶段补任务）。

## 测试与验证方式 / 结果

### 构建验证（通过）
- 生成 bindings + 完整构建：在 worktree 根目录执行：
  - `$env:GOWORK='off'; wails build -nopackage`
- 前端单独构建（需先生成 bindings）：在 `frontend/` 执行：
  - `npm run build`

### 手工冒烟建议
1) 打开 `Devices` 页面，确认每行右侧出现 `Edit` 按钮。
2) 点击节点行：仍打开 NodeInfo；点击 `Edit`：仅打开 Config Overlay。
3) Config Overlay 中点击 `Edit` 修改 value，点击 `Save` 后列表 value 更新并提示成功。
4) 导航侧边栏：`Management` 不再出现；`Devices` 位于 `Operations` 分组，且在 `File Console` 上方。

## 潜在影响与回滚方案

### 潜在影响
- `/management` 路由被彻底删除（无兼容 redirect），历史书签会失效。

### 回滚方案
- 按任务粒度 revert（建议顺序：恢复 router/nav → 恢复 `Management.vue` → 回退 `Devices.vue` Config Overlay 相关改动）。

