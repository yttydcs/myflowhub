# 2026-02-21 - Win：Devices 树形展示（懒加载）

## 变更背景 / 目标
联调阶段需要更直观地查看当前会话下的节点/设备拓扑。此前 `Session → Devices` 仅提供平铺列表（Direct/Subtree），难以体现层级关系。

本次变更将 Devices 页面升级为**树形结构**展示，并采用**懒加载**方式逐级查询直连子节点，避免一次性请求过多数据，同时对重复节点进行防环处理。

## 具体变更内容

### 修改 - `MyFlowHub-Win/frontend/src/stores/devices.ts`
- 新增树形状态模型 `DeviceTreeNode`：
  - `expanded/loading/error/children` 等节点状态
  - `duplicate=true` 的重复节点实例禁止继续展开（允许展示但不递归）
- 新增懒加载流程：
  - Root 加载：按 Mode 调用 `ListNodesSimple`（Direct）或 `ListSubtreeSimple`（Subtree）
  - 节点展开：统一调用 `ListNodesSimple` 获取该节点的直连 children
- 增加性能与稳定性保护：
  - 已加载 children 缓存，重复展开不重复请求
  - 节点级 `loading` 防止并发重复请求
  - `epoch` 防止切换 mode / reload root 后旧请求回包污染新树

### 修改 - `MyFlowHub-Win/frontend/src/pages/Devices.vue`
- UI 从平铺列表改为树形展示（缩进 + 展开/收起）
- Mode 改为下拉框：
  - `Direct` / `Subtree (direct + self)`
  - 切换 mode 自动清空并重载 Root
- Root 输入改为 Enter 触发重载
- 进入 Devices tab：
  - 若已连接且已登录：自动按当前 mode 加载 Root
- 节点级错误与重试：
  - 展开失败时显示错误，并提供 Retry
- 文案强调：
  - Subtree 非递归（当前实现为“直连 + 自身”）

## 与计划任务映射（plan.md）
- DEV0：完成（worktree 隔离）
- DEV1：完成（store：树模型 + 懒加载 + 去重策略）
- DEV2：完成（页面：Mode 下拉 + Enter reload + 树形渲染）
- DEV3：完成（验证：go test + wails build；手工冒烟见下）
- DEV4：完成（Code Review：通过）
- DEV5：完成（本归档文档）

## 关键设计决策与权衡
1) **不改 wire/协议**
   - Root 仍使用现有 `list_nodes` / `list_subtree`；逐级展开统一用 `list_nodes`
   - 优点：落地快、风险低；满足联调工具定位
2) **重复节点策略：允许展示但不继续展开**
   - 用全树 `seenNodeIDs` 做重复判定：首次出现可展开，后续出现标记 `duplicate` 并禁用展开
   - 优点：防止环导致无限递归；实现简单可控
   - 代价：在“同一 nodeId 多父节点挂载”的拓扑里，只能展开首次出现的实例（符合当前需求确认）
3) **性能与稳定性**
   - 懒加载避免请求风暴；缓存 children 避免重复 I/O
   - `epoch` 处理“切换 mode / Root 重载”与“旧请求回包”竞争，保证 UI 不被过期数据污染

## 测试与验证方式 / 结果

### 命令验证（Windows）
- `GOWORK=off go test ./...`：通过（仓库无单测文件）
- `GOWORK=off wails build -nopackage`：通过，生成 `build/bin/myflowhub-win.exe`

### 手工冒烟（建议步骤）
1) 启动 server + win（可用 `d:\\project\\MyFlowHub3\\scripts\\run-dev.ps1`）
2) Home：Connect → Register/Login（得到 nodeId/hubId）
3) Session → Devices：
   - 进入 tab 自动加载 Root（默认回落 hubId）
   - 展开任意节点：能加载并显示下一层
   - 切换 mode：树清空并自动重载 Root
   - Root 输入后按 Enter：重载 Root
   - 出现重复节点：可见“Duplicate”，但不可继续展开

## 潜在影响与回滚方案
- 影响范围：仅影响 Win 前端的 Devices 页面与其 store；不影响 wire/后端。
- 回滚：
  - revert 分支提交即可：
    - `feat(win): Devices 树形 store（懒加载/防重复展开）`
    - `feat(win): Devices 树形界面（Mode 下拉/Enter 重载）`

