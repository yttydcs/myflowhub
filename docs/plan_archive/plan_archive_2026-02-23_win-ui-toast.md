# Plan - Win：全局 Toast + 导航图标统一 + Local Hub 参数 + Devices 自身节点短路

> Workflow 目录：`d:\project\MyFlowHub3\worktrees\win-ui-toast\`
>
> 本 workflow 独占：
> - Worktree：`MyFlowHub-Win`：`d:\project\MyFlowHub3\worktrees\win-ui-toast\MyFlowHub-Win`
> - Branch：`feat/win-ui-toast`
> - Plan：`d:\project\MyFlowHub3\worktrees\win-ui-toast\plan.md`（本文件）
> - Change docs：`d:\project\MyFlowHub3\worktrees\win-ui-toast\docs\change\`
>
> guide.md 约束：
> - commit 信息使用中文（前缀如 `feat`/`fix` 可英文）

---

## 1. 项目目标与当前状态

### 1.1 目标
在 `MyFlowHub-Win` 中完成以下收敛：

1) **全局 Toast（顶部提示）**
- 统一全仓错误提示方式：失败/成功/警告/信息均通过 Toast 展示。
- 逐步替换页面内“某处一行 message 文本”。

2) **导航图标块样式统一**
- 左侧导航每个 item 的 “short” 图标块统一为固定尺寸的正方形圆角矩形（避免因 flex/shrink 或字体导致形变）。

3) **Local Hub 支持 hub_server 参数配置**
- 常用字段表单（可校验）+ 高级 extra args（每行一个完整参数），并持久化保存。
- 启动/重启时参数生效（命令行 flags）。

4) **Devices：Win 自身节点超时短路**
- 当 management 目标节点为 Win 自身（`targetID == sourceID`）时，直接返回空节点列表（OK），防止 UI 报错与超时等待。

### 1.2 当前状态
- 多数页面失败提示使用 `message.value = "..."` 并在页面底部渲染一行红字，反馈不够显眼。
- 左侧导航图标块（`item.short`）存在视觉不一致（尤其 Local Hub）。
- Local Hub 当前仅支持 `addr(host/port)`，缺少 `hub_server` 其他 flags 的配置入口。
- Win 自身节点不实现 management 入站，因此 `management list_nodes` 对 `targetID=自身` 会超时（需短路为叶子节点）。

---

## 2. 范围定义

### 必须（本 workflow）
- Toast：实现全局 Toast 组件 + store，并把 **所有页面/窗口** 的 message 提示迁移到 Toast（你已确认 2.B）。
- Nav：统一左侧导航图标块为固定方形圆角矩形（含窄屏顶部 nav 标签）。
- Local Hub：支持以下常用参数，并提供高级 extra args：
  - `-node-id`
  - `-parent`
  - `-parent-enable`
  - `-parent-reconnect`
  - `-auth-default-role`
  - `-auth-default-perms`
  - `-auth-node-roles`
  - `-auth-role-perms`
- Devices：management `ListNodes*` / `ListSubtree*` 对 `sourceID==targetID` 返回 `code=1,nodes=[]`。

### 可选（本 workflow 若不增加复杂度）
- Toast：支持复制详情/展开错误（长错误可折叠）。
- Toast：区分不同 default duration（success/info 短，warn/error 长）。

### 不做（本 workflow）
- 不改 wire/协议。
- 不将 Win 内嵌 server（仍为 sidecar 模式）。
- 不做 Local Hub “优雅停止控制面”（保持当前 stop 策略）。

---

## 3. 任务清单（Checklist）

### DEV0 - 基线确认
**目标**
- worktree/branch 正确；工作区干净。

**涉及**
- `d:\project\MyFlowHub3\worktrees\win-ui-toast\MyFlowHub-Win`

**验收**
- `git status` 无未提交改动
- branch 为 `feat/win-ui-toast`

**回滚点**
- 不适用

---

### UI1 - 导航图标块统一为方形圆角矩形
**目标**
- 左侧 nav item 的 short 图标块固定为正方形圆角矩形，避免被压缩或随内容变化。
- 窄屏（顶部横向 nav pills）保持一致视觉（点/圆点可保持，但整体对齐统一）。

**涉及文件（预计）**
- `MyFlowHub-Win/frontend/src/layout/AppShell.vue`

**验收**
- Local Hub / 其它项图标块尺寸一致（正方形，圆角一致）。

**测试点**
- 宽屏/窄屏切换（调整窗口宽度）。

**回滚点**
- revert 本提交

---

### UI2 - 全局 Toast 组件 + store
**目标**
- 新增 `toast store`：支持 `success/warn/error/info`，可关闭，自动消失（按级别区分 duration）。
- 新增 `ToastHost`：固定在顶部（AppShell 内），不遮挡主内容过多。

**涉及文件（预计）**
- `MyFlowHub-Win/frontend/src/stores/toast.ts`（新增）
- `MyFlowHub-Win/frontend/src/components/ToastHost.vue`（新增）
- `MyFlowHub-Win/frontend/src/layout/AppShell.vue`（挂载 ToastHost）

**验收**
- 任意页面触发 toast 能正常显示/消失/关闭。

**性能注意**
- toast 队列默认限制条数（例如最多 5 条，超出丢弃最旧）。

**回滚点**
- revert 本提交

---

### UI3 - 全仓 message 提示迁移到 Toast（全覆盖）
**目标**
- 将所有页面/窗口中 `message.value` + 底部红字提示迁移为 `toast.*()`。
- 页面内仅保留必要的“状态区域”（例如 listMessage 这种模块状态可保留），但操作失败必须 toast。

**涉及文件（预计）**
- `MyFlowHub-Win/frontend/src/pages/*.vue`
- `MyFlowHub-Win/frontend/src/windows/*.vue`
- 相关 store（如需要）

**验收**
- 常见失败路径（未连接/未登录/请求超时/下载失败/保存失败）均以 toast 顶部提示出现。

**测试点**
- `Home`：Connect 失败、Login 失败
- `Devices`：Load root 失败、Expand 失败
- `Local Hub`：Refresh/Install/Start 失败
- `File/Flow/...`：至少覆盖 1-2 个失败路径冒烟

**回滚点**
- revert 本提交

---

### LH1 - Local Hub：hub_server 参数配置（常用字段 + extra args）
**目标**
- 扩展 `LocalHubService` 配置：
  - 常用字段（结构化）：node-id/parent/.../auth-*
  - 高级：extra args（每行一个完整参数字符串，原样拼接到命令行）
- 配置持久化保存；Start/Restart 使用新参数。

**涉及文件（预计）**
- `MyFlowHub-Win/internal/services/localhub/types.go`
- `MyFlowHub-Win/internal/services/localhub/service.go`
- `MyFlowHub-Win/frontend/src/pages/LocalHub.vue`

**验收**
- 修改参数后 Restart 生效（从 localhub log 或 Snapshot 显示的启动参数可验证）。
- 校验：
  - node-id > 0
  - parent-enable=true 时 parent 必填
  - parent-reconnect >= 1（或允许 0 代表默认）

**安全注意**
- extra args 仅作用于本机启动命令；UI 提示“高级参数需谨慎”。

**回滚点**
- revert 本提交

---

### MGMT1 - Devices：自身节点短路返回空列表
**目标**
- 在 `ManagementService.ListNodes/ListSubtree` 入口短路：`sourceID==targetID` 返回 `code=1,nodes=[]`。

**涉及文件**
- `MyFlowHub-Win/internal/services/management/service.go`

**验收**
- `Devices` 展开到自身节点不再出现 timeout toast/报错，显示为 Leaf/No children。

**回滚点**
- revert 本提交

---

### DEV3 - Code Review（阶段 3.3）
逐项输出结论（通过/不通过）：
- 需求覆盖
- 架构合理性（Toast 集中；Local Hub 参数边界清晰；自身节点短路符合“叶子节点”设定）
- 性能风险（toast 队列/重渲染；避免频繁 setInterval 等）
- 可读性与一致性
- 可扩展性（后续加 dialog/复制详情/更多 flags）
- 稳定性与安全（参数校验、非回环提示不弱化）
- 测试覆盖情况（go test + wails build + 冒烟）

---

### DEV4 - 归档变更（阶段 4）
**目标**
- 在本 workflow 根目录 `docs/change/` 下新增变更文档：
  - `YYYY-MM-DD_win-ui-toast-and-localhub.md`
- 内容按规范写齐（背景/目标、变更内容、任务映射、权衡、验证、回滚）。

---

## 4. 依赖关系与注意事项
- `UI3` 依赖 `UI2`（toast 先落地再迁移）。
- `LH1` 与 `UI3` 有局部冲突风险（同改 `LocalHub.vue`），执行顺序建议：`UI2 → UI1 → UI3(先通用) → LH1(补 LocalHub 页字段) → UI3(再扫尾) → MGMT1`。
- 由于要求“全仓迁移 toast”，建议分多次 commit（每个模块一到两个提交），便于回滚与 review。

