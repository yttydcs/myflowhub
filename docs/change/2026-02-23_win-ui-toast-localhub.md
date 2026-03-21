# 2026-02-23 — Win：全局 Toast + 导航图标统一 + Local Hub 参数 + Devices 自身节点短路

## 变更背景 / 目标

本次变更聚焦于 `MyFlowHub-Win` 的体验一致性与可维护性：

1. **统一操作反馈**：将“页面某处显示一行 message”的提示方式，收敛为全局顶部 Toast（成功/信息/警告/错误）。
2. **统一导航视觉**：左侧导航 `short` 图标块固定为正方形圆角矩形，避免因布局压缩导致形变。
3. **Local Hub 参数配置**：为 `hub_server` 增加常用 flags 的配置入口，并支持高级参数（每行一个完整参数）。
4. **Devices 自身节点超时处理**：当 management 目标节点为 Win 自身时（`targetID == sourceID`），短路返回空列表，避免超时报错。

## 具体变更内容（新增 / 修改 / 删除）

### 前端（Vue）

- 新增全局 Toast
  - 新增 `frontend/src/stores/toast.ts`：Toast 队列（上限 5）、按级别自动消失、支持手动关闭。
  - 新增 `frontend/src/components/ToastHost.vue`：顶部固定层展示 Toast。
  - `frontend/src/layout/AppShell.vue` 挂载 `ToastHost`，确保主页面与 window layout 都可用。
- 导航图标块统一
  - `frontend/src/layout/AppShell.vue`：导航 `short` 块加 `shrink-0`/`leading-none`，固定方形视觉。
- 全仓 message 迁移至 Toast
  - `frontend/src/pages/*`、`frontend/src/windows/LogWindow.vue`：移除 `message.value` + 底部红字，改为 `toast.success/info/warn/errorOf`。
  - `frontend/src/stores/profile.ts`：加载/切换 profile 失败改为 toast。
  - `frontend/src/stores/devices.ts`：展开/重试等失败路径补充 toast（避免仅在节点行内显示错误）。
  - `frontend/src/stores/management.ts`：移除 `state.message`（页面成功提示已用 toast）。
- Local Hub 页面增强
  - `frontend/src/pages/LocalHub.vue`：新增 **Hub Params** 表单：
    - 常用字段：`node-id`、`parent`/`parent-enable`/`parent-reconnect`
    - Auth 字段：`auth-default-role`、`auth-default-perms`、`auth-node-roles`、`auth-role-perms`
    - 高级参数：`extraArgs`（每行一个完整参数；`#` 开头行忽略）
  - `SaveConfig` 现在保存整套配置（Listen + Params），并包含基础校验提示。

### 后端（Go / Wails Service）

- LocalHub 配置扩展与持久化
  - `internal/services/localhub/types.go`：扩展 `localhub.Config` 字段（与 UI 对齐）。
  - `internal/services/localhub/service.go`
    - 扩展配置读写 keys，持久化：node-id/parent/auth/extraArgs 等。
    - `SaveConfig` 增加输入校验（nodeId > 0；parent-enable 时 parent 必填；reconnect >= 0）。
    - `Start()` 组装 `hub_server` 启动参数：结构化 flags + `extraArgs`（按行切分，忽略空行与 `#` 注释行）。
- Management 自身节点短路
  - `internal/services/management/service.go`：`ListNodes` / `ListSubtree` 当 `sourceID == targetID` 直接返回 `{code:1,nodes:[]}`，避免 Win 作为叶子节点时等待超时。

## 对应 plan.md 任务映射

- UI1：导航图标块统一为方形圆角矩形
- UI2：全局 Toast 组件 + store
- UI3：全仓 message 提示迁移到 Toast（全覆盖）
- LH1：Local Hub：hub_server 参数配置（常用字段 + extra args）
- MGMT1：Devices：自身节点短路返回空列表

## 关键设计决策与权衡

- Toast 作为唯一操作反馈通道：
  - 优点：页面反馈更一致；失败信息更显眼；减少页面内“隐藏角落的红字”。
  - 权衡：高频动作（如 Flow undo/redo）会产生更多提示；通过自动消失 + 队列上限降低干扰。
- Local Hub 参数：
  - 结构化字段覆盖常用 flags，降低手写参数出错概率。
  - `extraArgs` 保留可扩展性（无需 UI 立即覆盖所有 flags），并允许覆盖同名 flags（按追加顺序“后者生效”）。
  - `parent-reconnect=0` 约定为“使用 server 默认值”，因此启动时不下发该 flag。
- Devices 自身节点短路：
  - 由于 Win 端不实现 management 入站（叶子节点定位），对自身发 management 请求会超时；短路符合预期且提升体验。

## 测试与验证方式 / 结果

- `GOWORK=off go test ./...`：通过（无测试文件）。
- `GOWORK=off wails build -nopackage`：通过（Windows/amd64）。

**冒烟建议**

1. 打开 Win，进入任意模块，触发一次失败（如未连接时点击需要连接的操作）→ 顶部 toast 展示错误。
2. Devices：将 Root 设置为自身 NodeID，Reload / Expand → 不再出现 `request timed out`，显示 Leaf/No children。
3. Local Hub：设置 `node-id`、启用 parent 并填写 `parent`，Start/Restart 后检查 hub_server 日志中 `node_id/parent` 是否生效。

## 潜在影响与回滚方案

- 影响：
  - UI 不再在页面底部显示 `message` 文本，提示全部集中在顶部 toast。
  - Local Hub 会新增若干持久化配置 key（旧配置仅有 host/port 仍可兼容）。
- 回滚：
  - 回滚对应提交（`git revert`），或直接回退分支合并即可。

