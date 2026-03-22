# Plan - Win：Devices 树形展开失效修复（可点击查看子节点）

> Workflow 目录：`d:\project\MyFlowHub3\worktrees\win-devices-tree-expand\`
> 目标仓库工作区：`d:\project\MyFlowHub3\worktrees\win-devices-tree-expand\MyFlowHub-Win\`
> 分支：`fix/win-devices-tree-expand`（基于 `main @ 3109091`）
>
> guide.md 约束：
> - commit 信息使用中文（前缀如 `fix` 可英文）
>
> 需求来源（用户反馈）：
> - “目前似乎无法点开查看子节点”（截图显示 Root 行显示 `Children: 1` 但列表未展开渲染子节点）

---

## 0. 当前进度（更新：2026-02-21）

- DEV1 已完成：节点对象改为 `reactive(...)` 创建，修复 computed 缓存不失效导致的“子节点不渲染”。
- DEV2 已完成：`GOWORK=off go test ./...`、`GOWORK=off wails build -nopackage` 通过。
- 代码提交：`fix: 修复 Devices 树子节点不刷新`（`97b5f30`）
- 待办：阶段 3.3 输出 Code Review；阶段 4 文档已生成（`docs/change/2026-02-21_win-devices-tree-expand.md`）；待合并到 `main` 并 push。

## 1. 项目目标与当前状态

### 1.1 目标
修复 `Session → Devices` 树形展示中“子节点无法显示/展开”的问题，使：
- Root 加载后能正确渲染第一层 child 行；
- 点击 `+/-` 可展开/收起；
- 子节点展开后能懒加载其 children 并渲染出来；
- 不改变 wire/协议与后端。

### 1.2 当前状态（事实 + 初步定位）
- 当前实现依赖 `visibleNodes = computed(() => flattenVisible(state.root))`。
- `devices.ts` 中 `root`/children 节点使用**普通对象**创建并通过本地变量修改（例如 `root.children = ...`）。
- 由于 Vue 计算属性依赖跟踪在 **Proxy** 上触发，而对普通对象的“脱离 Proxy 修改”可能不会触发 computed 失效，
  导致 UI 出现：节点行显示 `Children: N`（受其他 state 变化触发重渲染影响）但 `visibleNodes` 仍缓存旧值（只含 root），从而看不到 child 行。

---

## 2. 范围定义

### 必须（本 PR）
- 修复树节点的响应式更新，确保 children 加载后 `visibleNodes` 重新计算并渲染子节点。
- 修复节点展开/收起交互对 UI 的影响（展开后应出现下一层行）。
- 保持现有交互约束不变：
  - Mode 下拉切换 + 自动重载
  - Root Enter 重载
  - 重复节点允许展示但禁止继续展开
- 不修改 wire/协议与 Go 后端。

### 不做（本 PR）
- 不新增“展开全部/展开到 N 层”
- 不新增协议递归子树能力

---

## 3. 任务清单（Checklist）

### DEV0 - 基线确认
**验收**
- worktree：`worktrees/win-devices-tree-expand/MyFlowHub-Win`
- 分支：`fix/win-devices-tree-expand`
- `git status` 干净

---

### DEV1 - 修复 store 响应式：树节点使用 reactive Proxy
**目标**
- 让 root 与所有 tree 节点以 Vue `reactive(...)` 形式创建（或确保后续修改始终通过 Proxy 进行），使：
  - children 加载/赋值能触发 computed 失效
  - expanded/loading/error 等字段更新能即时刷新 UI
- `nodeIndex` 存储 Proxy 节点引用（用于 toggle/retry）。

**涉及文件**
- `frontend/src/stores/devices.ts`

**验收**
- Root 加载后，child 行能显示在列表中（不再仅显示 root）
- 点击 `+/-` 能正确展开/收起（可见层级变化）

**回滚点**
- revert 本提交

---

### DEV2 - 验证（命令 + 手工）
**命令**
- `GOWORK=off go test ./...`
- `GOWORK=off wails build -nopackage`

**手工冒烟**
1) Home：Connect → Register/Login
2) Session → Devices：
   - 自动加载 Root 后应立刻显示第一层 child 行
   - 点击某个 child 的 `+`：应出现下一层（或提示 No children）
   - 切换 Mode/Root Enter 重载：树刷新正确

---

### DEV3 - Code Review（阶段 3.3）
- 需求覆盖：子节点可见 + 可展开
- 架构：修复点聚焦 store；不引入额外耦合
- 性能：不引入轮询；不重复请求
- 稳定性：避免过期回包污染（epoch 仍生效）
- 测试：build + 冒烟

---

### DEV4 - 归档变更（阶段 4）
- `docs/change/2026-02-21_win-devices-tree-expand.md`
  - 背景/目标、变更内容、plan 映射、验证结果、回滚方案
