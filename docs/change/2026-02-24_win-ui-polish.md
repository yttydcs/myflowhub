# 变更说明：Win 前端 UI 收敛（Devices hover/root + 页面头部移除 + 导航图标化）

## 变更背景 / 目标
你提出以下 4 个 UI 问题需要统一修复与收敛：
1) `Devices` 节点列表缺少明显 hover 反馈；
2) `Devices` 树中“所有节点都像 root”（root 标记错误）；
3) 页面顶部存在 “Session / Devices / 说明文案” 的 header 区块，你要求 **整块移除**（无实际意义）；
4) 左侧导航（tab 选项）使用两位首字母（`HM/DV/...`），你要求改为 **图标**（并确认使用图标库方案）。

本次变更目标：
- 提升 `Devices` 列表的可交互感（hover 清晰）；
- 修复 root 标记逻辑，仅真实 root 显示；
- 移除顶部说明性 header，并迁移其上的操作控件到页面内合适位置，保证功能不丢失；
- 左侧导航图标化，保留现有激活态/非激活态色彩策略。

## 具体变更内容（新增 / 修改 / 删除）

### 修改：`Devices`（hover + root 标记 + 控件迁移）
- `frontend/src/pages/Devices.vue`
  - 节点行 hover 增强：增加 `hover:bg-muted/60`、`hover:border-*`、`hover:shadow-sm`，并为展开按钮补齐 hover 反馈。
  - root 标记修复：从 `node.key.startsWith('root:')` 改为 `depth === 0`，避免“root key 前缀传染到所有子节点”导致误判。
  - 移除顶部 header 区块；将原 header 的 `Identity / Mode / Root / Reload` 控件迁移到 `Nodes` 卡片 header 区。

### 修改：全局移除页面顶部说明 header（有则移除；无则跳过）
- `frontend/src/pages/File.vue`
  - 移除顶部 header；将 `Tasks / Settings / Refresh` 迁移到文件列表主卡片 header 区。
- `frontend/src/pages/Flow.vue`
  - 移除顶部 header；将 `Executor + Refresh/New/Undo/Redo/AutoLayout/Save/Run/Status` 迁移到 `Flow Editor` 卡片 header 区。
- `frontend/src/pages/LocalHub.vue`
  - 移除顶部 header；将 `Supported/Unsupported badge + Reload snapshot` 合并到 `Latest Release` 卡片 header 区，并保留原 `Refresh`。
- `frontend/src/pages/Logs.vue`
  - 移除顶部 header；将 `Lines` 与 `Open Window` 迁移到日志卡片 header 区。
- `frontend/src/pages/Management.vue`
  - 移除顶部 header；将 `Target + List Direct/Subtree` 迁移到 `Nodes` 卡片 header 区，并保留列表模式信息展示。
- `frontend/src/pages/Debug.vue`
  - 移除顶部 header；将 `Session` 状态 pill 迁移到 `Connection` 卡片 header 区。

### 修改：左侧导航两字母缩写 → 图标
- `frontend/src/layout/AppShell.vue`
  - `NavItem.short` 改为 `NavItem.icon`（Vue Component）。
  - 导航方块渲染从两字母文本改为 `<component :is="item.icon" class="h-5 w-5" />`。

### 依赖变更
- `frontend/package.json` / `frontend/package-lock.json`
  - 新增依赖：`lucide-vue-next`

## 对应 plan.md 任务映射
- `plan.md`
  - T2：Devices（hover + root 修复 + 顶部 header 移除与控件迁移）
  - T3：全局移除顶部说明 header，并迁移控件
  - T4：左侧导航图标化（`lucide-vue-next`）
  - T5：验收与构建

## 关键设计决策与权衡（性能 / 扩展性）
1) **root 判断改为 `depth === 0`**
   - 解决根因：子节点 key 会继承父 key 前缀，导致 `startsWith('root:')` 误判整棵树。
   - 优点：不改 store/key 结构，改动面小、风险低、语义清晰。
2) **顶部 header 整块移除，但控件不丢**
   - 采取“就近迁移”：把原 header 上的操作控件迁移到该页面第一块相关卡片/列表 header 区。
   - 好处：减少无意义占位，操作更贴近其作用对象。
3) **导航图标库选型：`lucide-vue-next`**
   - 优点：风格统一、按需 import、SVG 继承当前 `text-*` 色彩方案，适配现有 active/tone 策略。
   - 代价：引入一个新的前端依赖（已锁定在 lockfile 中）。

## 测试与验证方式 / 结果
已执行：
- `GOWORK=off wails build -nopackage` ✅（包含 bindings 生成 + 前后端编译链路）
- `cd frontend && npm run build` ✅

建议人工冒烟（UI）：
1) `Devices`：hover 明显；仅 root 行显示 `(root)`；控件（Mode/Root/Reload）可用；节点详情弹窗仍可开关。
2) `File/Flow/LocalHub/Logs/Management/Debug`：顶部 header 已移除；原控件仍可用且布局合理。
3) 左侧导航：图标显示正常；激活态/非激活态对比清晰。

## 潜在影响与回滚方案
### 潜在影响
- 页面整体信息密度上升（顶部大标题消失），对“首次进入页面需要说明”的用户不友好；但符合你“移除无意义说明”的明确需求。
- Flow 页的操作按钮从页面顶部移入 `Flow Editor` 卡片 header，位置变化需要适应。

### 回滚方案
- 回滚本次实现 commit `b64b5fa` 即可整体撤销。
- 若仅回滚图标：revert `AppShell.vue` + 移除 `lucide-vue-next` 依赖即可。

