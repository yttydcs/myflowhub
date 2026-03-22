# Plan - Win：在 Session 下新增“设备查询”界面（Devices）

> Workflow 目录：`d:\project\MyFlowHub3\worktrees\win-session-devices\`
> 目标仓库工作区：`d:\project\MyFlowHub3\worktrees\win-session-devices\MyFlowHub-Win\`（分支：`feat/win-session-devices`）
>
> guide.md 约束：
> - commit 信息使用中文（前缀如 `feat` 可英文）
>
> 需求来源（用户确认）：
> - “给 Win 添加一个查询设备的界面，tab 选项放在 Session 下面即可。”

---

## 1. 项目目标与当前状态

### 1.1 目标
在 Win 应用左侧导航的 **Session** 分组下新增一个入口（Devices/设备查询），用于在连接并登录后，快速查询当前网络视角的“设备/节点列表”，用于联调与验证。

### 1.2 当前状态（事实）
- Win 当前已有：
  - Session/Home（连接 + 注册/登录 + 身份快照）：`frontend/src/pages/Home.vue`
  - Management（nodes + config 编辑）：`frontend/src/pages/Management.vue`
- 协议侧已有可用动作（wire 不改）：
  - `management.list_nodes` / `management.list_nodes_resp`
  - `management.list_subtree` / `management.list_subtree_resp`
- 当前 `list_subtree` 的实现语义（事实）：返回“目标节点的直连 nodes + 目标节点自身”，**不是递归全量子树**（见 `MyFlowHub-SubProto/management/action_nodes.go`）。

---

## 2. 范围定义

### 必须（本 PR）
1) 新增页面与路由：
   - 新增 `frontend/src/pages/Devices.vue`
   - 路由 `GET /devices`（hash 路由）并在左侧导航的 Session 分组下增加入口
2) 新增前端 store（不复用 Management store，避免状态互相污染）：
   - `frontend/src/stores/devices.ts`
   - 调用 Wails binding：`ManagementService.ListNodesSimple` / `ListSubtreeSimple`
3) UI 功能：
   - Target Node 输入（默认回落到 hubId）
   - 两个按钮：List Direct（list_nodes）、List Subtree（list_subtree）
   - 列表展示：nodeId + hasChildren
   - 错误提示与空状态提示

### 可选（本 PR，如不扩大改动则不做）
- “点击节点 → 一键把 Target 填为该 nodeId” 的快捷操作（提升体验）

### 不做（本 PR）
- 不改 wire/协议（SubProto/Action/JSON schema 保持不变）
- 不新增“递归全量子树”能力（当前 `list_subtree` 非递归，若需要另起 workflow）
- 不做 UI 自动化测试（以命令级 build + 手工冒烟为主）

---

## 3. 任务清单（Checklist）

### DEV0 - 基线确认
**目标**
- 在独占 worktree 内完成改动；主仓仅做合并与推送。

**验收**
- 分支：`feat/win-session-devices`
- `git status` 干净

**回滚**
- 删除 worktree + 分支即可

---

### DEV1 - 增加路由与导航入口（Session 分组）
**目标**
- 增加 `/devices` 路由与 nav item，使其出现在 Session 分组下。

**涉及文件**
- `frontend/src/router/index.ts`
- `frontend/src/layout/AppShell.vue`

**验收**
- 左侧导航出现 “Devices”（或“设备查询”）入口
- 点击可进入页面且标题/副标题正确

**回滚点**
- revert 本提交

---

### DEV2 - 新增 Devices store（只负责设备列表）
**目标**
- 独立 store 承载设备查询状态与调用逻辑（不复用 Management store）。

**涉及文件**
- 新增：`frontend/src/stores/devices.ts`

**验收**
- 未登录时给出明确错误（需要 nodeId/hubId）
- list_nodes / list_subtree 可正确更新 `state.nodes`

**回滚点**
- revert 本提交

---

### DEV3 - 新增 Devices 页面（Session 下的设备查询 UI）
**目标**
- 提供可用、可理解的设备查询界面（Direct/Subtree 两种模式）。

**涉及文件**
- 新增：`frontend/src/pages/Devices.vue`

**验收**
- 可输入 target node id
- 点击按钮后展示 nodes
- 空列表/错误信息提示正确

**回滚点**
- revert 本提交

---

### DEV4 - 验证与冒烟
**命令**
- `cd MyFlowHub-Win && GOWORK=off go test ./...`
- `cd MyFlowHub-Win && GOWORK=off wails build -nopackage`

**手工冒烟**
1) 启动 server + win（可用 `d:\project\MyFlowHub3\scripts\run-dev.ps1`）
2) Home：Connect → Register/Login（得到 nodeId/hubId）
3) Session → Devices：
   - List Direct：能看到直连节点
   - List Subtree：能看到 “直连节点 + 自身”（与实现语义一致）

**回滚点**
- revert 分支提交

---

### DEV5 - Code Review（阶段 3.3）
逐项输出结论（通过/不通过）：
- 需求覆盖
- 架构合理性（store 隔离、无跨模块污染）
- 性能风险（避免频繁轮询、避免不必要解析）
- 可读性与一致性
- 稳定性与安全（输入校验、错误提示）
- 测试覆盖情况（命令级 + 手工冒烟）

---

### DEV6 - 归档变更（阶段 4）
**目标**
- 在当前 workflow 根目录创建 `docs/change/` 并新增归档文档。

**文件**
- `docs/change/2026-02-21_win-session-devices.md`

**必须包含**
- 变更背景/目标
- 具体变更（新增/修改文件）
- 与本 plan 的任务映射
- 关键决策与权衡
- 测试与验证方式/结果
- 潜在影响与回滚方案

---

## 4. 风险与注意事项
- `list_subtree` 当前语义非递归，UI 文案需避免误导为“全量子树”。
- 该页面属于“验证/联调工具”，避免引入定时轮询；以手动刷新为主。

