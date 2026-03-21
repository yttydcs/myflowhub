# Plan - Win：Devices 以树形结构展示（懒加载）

> Workflow 目录：`d:\project\MyFlowHub3\worktrees\win-devices-tree\`
> 目标仓库工作区：`d:\project\MyFlowHub3\worktrees\win-devices-tree\MyFlowHub-Win\`
> 分支：`feat/win-devices-tree`（基于本地 `repo/MyFlowHub-Win` 的 `main @ b497c40`）
>
> guide.md 约束：
> - commit 信息使用中文（前缀如 `feat` 可英文）
>
> 需求来源（用户确认）：
> - “Devices 能否以树形结构展示？”
> - 方案：B（懒加载树）
> - Mode 用下拉框切换；进入 tab 自动按模式加载
> - Root 修改后按 Enter 触发重载
> - 重复节点：允许重复展示，但不继续展开
> - 展开全部/展开到 N 层：后续再加

---

## 1. 项目目标与当前状态

### 1.1 目标
在 Win 应用 `Session → Devices` 页面，将节点/设备列表由平铺列表升级为**树形结构**展示（可展开/收起），并采用**懒加载**方式逐级查询直连子节点，以便联调阶段更直观地观察拓扑。

### 1.2 当前状态（事实）
- 已有 `Session → Devices` 页面（平铺列表）：
  - `frontend/src/pages/Devices.vue`
  - `frontend/src/stores/devices.ts`
- wire/协议现状（本 PR 不改）：
  - Root 加载可用：
    - `management.list_nodes` / `list_nodes_resp`
    - `management.list_subtree` / `list_subtree_resp`（注意：非递归全量子树，仅“直连 + 自身”）
  - 逐级展开时使用 `list_nodes` 获取“某节点的直连 children”

---

## 2. 范围定义

### 必须（本 PR）
1) UI 改为树形展示
   - Root 节点 + children 的层级展示
   - 节点可展开/收起（懒加载 children）
2) Mode 下拉框（两种）
   - `Direct`：Root 使用 `list_nodes`
   - `Subtree`：Root 使用 `list_subtree`（并在 UI 文案明确非递归；且过滤“自身”避免作为 child ）
   - 切换 mode：清空树并自动重新加载 Root
3) Root 输入与自动加载
   - Root 默认回落 `hubId`
   - Root 输入框按 **Enter** 触发 reload（避免输入过程频繁请求）
   - 进入 Devices 页面时，若已连接且已登录：自动加载 Root
4) 重复节点策略（用户确认）
   - 允许同一 `nodeId` 多处出现
   - 但一旦判定为“重复节点实例”：仍展示，但**禁止继续展开**
5) 错误处理与可观测性（前端侧）
   - 未连接/未登录/Root 非法：明确提示
   - Root/节点级请求失败：展示错误与重试按钮（节点失败不影响其他节点）

### 可选（本 PR 不做）
- 展开全部/展开到 N 层（需要并发与上限控制）
- 搜索/过滤/复制 nodeId 等增强

### 不做（本 PR）
- 不修改 wire/协议与后端实现
- 不新增“递归全量子树”能力

---

## 3. 任务清单（Checklist）

### DEV0 - 基线与隔离
**目标**
- 在独占 worktree 内完成实现；主仓 `repo/MyFlowHub-Win` 只做合并/推送。

**验收**
- worktree：`worktrees/win-devices-tree/MyFlowHub-Win`
- 分支：`feat/win-devices-tree`
- `git status` 干净

**回滚**
- 删除 worktree + 分支

---

### DEV1 - 重构 Devices store：树模型 + 懒加载 + 去重策略
**目标**
- 将现有平铺 `devices.ts` 升级为树形状态管理：
  - Root 加载（按 mode 使用 list_nodes/list_subtree）
  - 节点展开：统一用 list_nodes 拉直连 children
  - 缓存已加载 children，避免重复 I/O
  - 通过 `epoch/version` 防止切换 mode/root 后旧请求回包污染
  - 重复节点策略：重复实例 `duplicate=true` 且禁用展开

**涉及文件**
- 修改：`frontend/src/stores/devices.ts`

**验收**
- 未连接/未登录/Root 非法能给出明确错误
- 同一节点重复展开不会并发重复请求
- mode 切换/Root reload 后不会被旧请求覆盖

**测试点**
- 手工：切换 mode、快速输入 Root 并按 Enter、多次展开/收起验证状态稳定

**回滚点**
- revert 本提交

---

### DEV2 - 重构 Devices 页面：Mode 下拉框 + Enter reload + 树形渲染
**目标**
- 更新页面 UI/交互以匹配需求：
  - Mode 下拉框（Direct/Subtree）
  - Root 输入框按 Enter reload
  - 进入页面自动加载（连接+登录满足时）
  - 树形列表：缩进/折叠按钮/节点状态（loading/error/duplicate）
  - 节点级重试

**涉及文件**
- 修改：`frontend/src/pages/Devices.vue`
- （如需要拆组件）新增：`frontend/src/components/devices/*`

**验收**
- 进入 tab：自动加载 Root
- mode 切换：清空并自动重载
- Root Enter：触发重载
- 重复节点显示 “重复” 提示且不可展开

**回滚点**
- revert 本提交（含新增组件）

---

### DEV3 - 验证与冒烟
**命令**
- `cd MyFlowHub-Win && GOWORK=off go test ./...`
- `cd MyFlowHub-Win && GOWORK=off wails build -nopackage`

**手工冒烟**
1) 启动 server + win（可用 `d:\project\MyFlowHub3\scripts\run-dev.ps1`）
2) Home：Connect → Register/Login（得到 nodeId/hubId）
3) Session → Devices：
   - 默认 Direct：应自动加载 Root 并展示第一层 children
   - 展开任意节点：拉取并显示下一层
   - 切换 Subtree：清空并重载（非递归提示正确）
   - Root 输入后按 Enter：重新加载

**回滚点**
- revert 分支提交

---

### DEV4 - Code Review（阶段 3.3）
逐项审查并输出结论（通过 / 不通过）：
- 需求覆盖（树形/懒加载/模式切换/Enter reload/重复节点策略）
- 架构合理性（store 边界清晰，UI 与数据职责分离）
- 性能风险（避免请求风暴、避免重复计算、缓存策略正确）
- 可读性与一致性（命名、结构、文案口径一致）
- 稳定性与安全（输入校验、错误处理、过期回包保护）
- 测试覆盖（命令级 + 手工冒烟）

---

### DEV5 - 归档变更（阶段 4）
**目标**
- 在 workflow 根目录创建 `docs/change/` 并新增归档文档。

**文件**
- `docs/change/2026-02-21_win-devices-tree.md`

**必须包含**
- 变更背景/目标
- 具体变更（新增/修改/删除）
- 与本 plan 的任务映射
- 关键决策与权衡（尤其性能/防环/缓存）
- 测试与验证方式/结果
- 潜在影响与回滚方案

---

## 4. 依赖关系、风险与注意事项
- 依赖：Management Wails binding 已存在（`ManagementService.ListNodesSimple/ListSubtreeSimple`）。
- 风险：
  - 当前环境 GitHub `push` 可能失败（网络不通）；需在网络恢复后补 push。
  - `list_subtree` 非递归，“Subtree”在 UI 中必须明确为“direct + self”，避免误解为全量子树。
- 注意：
  - 不引入自动轮询；所有加载由用户操作或进入页面触发。

