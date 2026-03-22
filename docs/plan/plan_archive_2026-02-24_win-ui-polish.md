# Plan - Win 前端：Devices 列表交互增强 + 页面顶部文案移除 + 导航图标化

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`refactor/win-ui-polish`
- Worktree：`d:\project\MyFlowHub3\worktrees\win-ui-polish\MyFlowHub-Win`
- Base：`main`（当前 HEAD：`d8351af`）
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）

## 背景（问题清单与用户需求）
你提出 4 个 UI 问题需要统一修复：
1) `Devices` 的节点列表缺少明显 hover 反馈（行看起来“不像可点/可操作”）。
2) `Devices` 树中“似乎所有节点都被标记为了 root”（root 标记错误）。
3) 各页面顶部存在类似：
   - `Session` / `Devices` / “Query the management plane ...”
   这类说明性 header 区块，你要求 **整块移除**（无实际意义）。
4) 左侧导航（tab 选项）当前使用 2 个首字母（`HM/DV/...`），你要求改为 **图标**（并确认使用图标库方案）。

## 目标
1) `Devices`：节点行 hover 更明显，且不影响展开按钮与点击行为。
2) `Devices`：仅真实 root 节点显示 `(root)`，其余节点不显示。
3) 全局：移除页面顶部“模块说明 header 区块”，并迁移原 header 上的控制按钮/输入框到页面内合适位置，保证功能不丢失。
4) 左侧导航：2 字母缩写替换为图标，保留现有激活态/非激活态色彩策略与可读性。

## 范围与约束
- 必须：
  - 覆盖所有存在“顶部说明 header 区块”的页面；不存在则跳过（按你的确认）。
  - 不改协议/后端/业务行为，仅做 UI/交互与信息结构收敛。
  - 图标使用图标库（你已确认）。
- 不做：
  - 不重做整体布局风格（不做大规模 UI redesign）。
  - 不要求把移动端顶部“圆点导航”也改成图标（你未提出；如后续需要再开新任务）。

## 关键决策（已确认）
- D1：顶部 header 区块移除策略 = **C（整块移除）**
- D2：覆盖范围 = **B（全局；没有则跳过）**
- D3：tab 选项指代 = **左侧导航两字母块**
- D4：图标方案 = **B（引入图标库）**

## 根因分析（可审计）
### `Devices` “全部节点都像 root”
- 现状：UI 使用 `node.key.startsWith('root:')` 判断 root。
- 但子节点 key 由父 key 拼接（`${parentKey}/${nodeId}`），会导致所有后代 key 都以 `root:` 开头，从而误判为 root。
- 修复策略：root 标记基于 **深度**（`depth === 0`）或基于 `devicesStore.state.root?.key` 精确判断。

---

## 3.1) 计划拆分（Checklist）

### T1. Workspace 准备（已完成）
- 目标：创建独占 worktree 与专业分支，确保不在 `repo/` 里做实现改动。
- 验收：
  - `git status -sb` 显示在 `refactor/win-ui-polish` 且工作区干净。
- 回滚点：`git worktree remove` + `git worktree prune` + 删除分支（若未推送）。

### T2. Devices：root 标记修复 + hover 反馈增强 + 顶部 header 移除与控件迁移（已完成）
- 目标：
  - root 标记仅出现在真实 root 行。
  - 节点行 hover 更明显（背景/边框/阴影更清晰），且不影响按钮点击。
  - 移除顶部 header 区块；将原 header 的 `Identity/Mode/Root/Reload` 控件迁移到 “Nodes” 卡片 header 区。
- 涉及文件：
  - `frontend/src/pages/Devices.vue`
- 验收：
  - 仅 `depth === 0` 的行显示 `(root)`。
  - hover 可见（肉眼明显）且不导致点击误触/按钮不可用。
  - 顶部 `Session/Devices/说明` 整块消失，但 `Mode/Root/Reload` 等功能仍可用。
- 测试点：
  - Root 输入 Enter 仍能触发 reload；Mode 切换仍能 reload。
  - duplicate 节点按钮仍禁用，hover 不误导为可展开。
- 回滚点：revert `Devices.vue`。

### T3. 全局：移除页面顶部说明 header 区块（有则移除；无则跳过）（已完成）
- 目标：删除各页面顶部的说明性 header 区块，并将该区块中的操作控件迁移到页面内第一块相关卡片/列表的 header 区，避免功能丢失。
- 预期涉及文件（以扫描结果为准）：
  - `frontend/src/pages/File.vue`（迁移：Tasks/Settings/Refresh）
  - `frontend/src/pages/Flow.vue`（迁移：Executor + Refresh/New/Undo/Redo/AutoLayout/Save/Run/Status）
  - `frontend/src/pages/LocalHub.vue`（迁移：Supported badge + Reload snapshot）
  - `frontend/src/pages/Logs.vue`（迁移：Lines + Open Window）
  - `frontend/src/pages/Management.vue`（迁移：Target + List Direct/Subtree）
  - `frontend/src/pages/Debug.vue`（迁移：Session pill）
- 验收：
  - 每个页面顶部 header 区块完全移除。
  - 原 header 上的控件在页面内仍可访问、可操作，且布局不突兀。
- 测试点：
  - `File`：Tasks/Settings/Refresh 可用；且不依赖顶部 header 存在。
  - `Flow`：常用按钮仍可操作，布局不拥挤到不可用。
- 回滚点：逐文件 revert。

### T4. 左侧导航：两字母缩写替换为图标（图标库方案）（已完成）
- 目标：将 `AppShell.vue` 侧边栏导航项从 `HM/DV/...` 替换为对应模块图标；保持激活态/非激活态色彩与对比度。
- 方案：
  1) 引入依赖：`lucide-vue-next`
  2) `NavItem` 增加 `icon` 字段，移除/废弃 `short`
  3) 模板中渲染 `<component :is="item.icon" class="h-5 w-5" />`
- 涉及文件：
  - `frontend/package.json` / `frontend/package-lock.json`
  - `frontend/src/layout/AppShell.vue`
- 验收：
  - 左侧导航不再显示两字母缩写，改为图标。
  - 激活态图标颜色/背景符合现有高亮策略；非激活态保留 `tone` 色彩。
- 回滚点：revert 依赖与 `AppShell.vue`。

### T5. 验收与构建（本地可执行）（已完成）
- 目标：保证构建通过，且关键页面无明显回归。
- 验收命令：
  - `cd frontend && npm run build`
  -（建议）`GOWORK=off wails build -nopackage`
- 执行记录（本次）：
  - `GOWORK=off wails build -nopackage` ✅
  - `cd frontend && npm run build` ✅
- 人工冒烟：
  - `Devices`：hover、root 标记、控件迁移、节点详情弹窗
  - `File/Flow/LocalHub/Logs/Management/Debug`：顶部 header 已移除且控件可用
  - 左侧导航：图标显示正常

---

## 依赖关系
- T4（图标库）依赖 `frontend` 依赖安装与 lock 更新。
- T2/T3 为纯页面调整，可并行，但建议按 “先 Devices 再全局页面” 执行便于回归对照。

## 风险与注意事项
- 顶部 header 移除后，部分页面的操作栏可能变拥挤：需要逐页微调布局（`flex-wrap`、分组、按钮 size）。
- 新增依赖需确保 tree-shake：仅按需 import 图标组件，避免无意引入整包。
