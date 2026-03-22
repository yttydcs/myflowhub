# Plan - Win 前端：Devices 集成 Config 编辑 + 移除 Management + 导航重排

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`refactor/win-devices-config`
- Worktree：`d:\project\MyFlowHub3\worktrees\win-devices-config\MyFlowHub-Win`
- Base：`main`（当前 worktree HEAD：`409f078`）
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）
- 运行与验收（建议）：
  1) `frontend` 构建：`cd frontend && npm run build`
  2) Win 构建（含 bindings）：`$env:GOWORK='off'; wails build -nopackage`

## 背景（问题与用户需求）
你反馈：`Devices` 与 `Management` 的能力有一定重复，希望收敛为一个入口。

具体需求：
1) 在 `Devices` 的节点列表每一行右侧新增“编辑”按钮；
2) 点击后弹出内容类似 `Management` 的 `Config` 面板（key 列表、value 展示、单条编辑保存）；
3) 移除 `Management`（你确认：**路由也删除**，不做 redirect）；
4) 将 `Devices` 从左侧导航的原分组移动到 `Operations` 分组，并放在 `File Console` 上面。

## 目标
1) `Devices` 内即可完成目标节点的 Config 查看与编辑（覆盖现有 `Management` 右侧 Config 面板能力）。
2) 删除 `Management` 页面入口与路由，避免模块重复与维护成本。
3) 导航结构更符合使用路径：`Devices` 与 `File Console` 相邻，且 `Devices` 在上。

## 范围与约束
- 必须：
  - 复用现有 `ManagementService` bindings（不改后端/协议）。
  - “编辑按钮”位置：节点行右侧（已确认）。
  - `Management` route 删除（已确认）。
  - `Devices` 导航移动分组到 `Operations`，并调整排序（已确认）。
- 不做：
  - 不为 `#/management` 做兼容 redirect（按你确认的决策）。
  - 不引入新的后端能力；不重写 Devices 树查询。

## 决策（已确认）
- D1：编辑按钮位置 = `Devices` 节点行右侧（A）
- D2：移除 `Management` 路由 = 彻底删除（A）
- D3：`Devices` 导航调整 = 移动到 `Operations`，位于 `File Console` 之前（A）

## 设计要点（实现约束）
- Config 能力复用策略（最小化变更）：
  - 复用现有 `frontend/src/stores/management.ts` 的 Config 逻辑（`selectNode/refreshConfig/setConfig`），避免再写一套重复请求与状态管理。
  - `Management` 页面移除后，该 store 仍作为“管理面（management plane）能力封装”被 `Devices` 使用。
- UI 结构：
  - 在 `Devices` 内新增一个 Config Overlay（全屏遮罩 + 中央面板），展示 Config 列表与 Refresh/Close。
  - 单条配置编辑仍使用二级 Overlay（与原 `Management` 的 Edit Config 弹窗一致）。
  - 交互安全：按钮需 `@click.stop`，避免触发节点行点击打开 NodeInfo。

---

## 3.1) 计划拆分（Checklist）

### T1. Workspace 准备（已完成）
- 目标：创建独占 worktree 与专业分支，确保不在 `repo/` 里做实现改动。
- 验收：
  - `git status -sb` 显示 `refactor/win-devices-config` 且工作区干净（或仅包含本 workflow 的变更）。
- 回滚点：`git worktree remove` + `git worktree prune` + 删除分支（若未推送）。

### T2. Devices：新增 Config 编辑入口（行右侧按钮 + Config Overlay）
- 目标：
  - 每个节点行右侧新增 “Edit/编辑” 按钮；
  - 点击后打开 Config Overlay，展示该节点配置列表（keys + values），并支持单条 Edit/Save。
- 涉及文件（预期）：
  - `frontend/src/pages/Devices.vue`
  - （复用）`frontend/src/stores/management.ts`（尽量不改；如需补充 loading/并发控制再评估）
- 验收：
  - 点击节点行仍打开 NodeInfo；点击 Edit 仅打开 Config Overlay（不触发 NodeInfo）。
  - Config Overlay 能显示 key 列表，并逐步填充 value（与原 Management 行为一致）。
  - Edit Config：保存成功后列表 value 更新，并 toast 提示成功；失败 toast 错误。
- 测试点：
  - 快速切换不同节点打开 Config，旧请求不会污染新节点（依赖 store 内 `selectedNodeId` guard）。
  - Esc 行为符合 Overlay 栈：仅关闭最上层（已有全局 Overlay 机制）。
- 回滚点：revert `Devices.vue` 中新增 Overlay 相关改动。

### T3. 移除 Management 模块（路由 + 导航 + 页面文件）
- 目标：删除 `Management` 页面入口，避免与 `Devices` 能力重复。
- 涉及文件（预期）：
  - `frontend/src/router/index.ts`（移除 `/management` route 与 import）
  - `frontend/src/layout/AppShell.vue`（移除导航项）
  - `frontend/src/pages/Management.vue`（删除文件）
- 验收：
  - 左侧导航不再显示 Management；
  - 路由表中不再存在 `/management`；
  - `frontend` build / `wails build` 通过。
- 回滚点：revert router/nav，并恢复页面文件。

### T4. 导航重排：Devices 移到 Operations 并位于 File Console 上方
- 目标：左侧导航结构调整符合你的信息架构需求。
- 涉及文件：
  - `frontend/src/layout/AppShell.vue`
- 验收：
  - `Devices` 出现在 `Operations` 分组，且排序在 `File Console` 之前；
  - 移动端顶部导航（`flatNav`）顺序同步变化。
- 回滚点：revert `AppShell.vue` 的 navGroups 变更。

### T5. 验收与回归（构建 + 冒烟）
- 目标：保证改动无编译/运行时回归。
- 验收命令：
  - `cd frontend && npm run build`
  - `$env:GOWORK='off'; wails build -nopackage`
- 冒烟：
  - `Devices`：树加载/展开正常；NodeInfo 仍可用；Config Overlay 可打开并编辑保存；
  - 导航：Management 消失；Devices 位置在 File Console 上方。
- 回滚点：逐 commit revert（建议按 T2/T3/T4 拆分 commit）。

---

## 依赖关系
- 建议顺序：T2（先把能力迁入 Devices）→ T3（移除 Management）→ T4（导航重排）→ T5（验收）。

## 风险与注意事项
- Config keys 数量较多时，逐 key 并发 `ConfigGet` 可能造成瞬时请求峰值：本次先沿用现有实现；如出现压力/卡顿再加并发上限与搜索过滤（需要回到 3.1 补充任务）。
- 删除 `/management` 不做 redirect：历史书签会失效（你已确认接受）。

