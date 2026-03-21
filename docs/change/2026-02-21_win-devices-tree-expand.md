# 2026-02-21 Win：Devices 树形子节点无法展开/显示 修复

## 背景 / 目标

用户反馈 `Session → Devices` 树形视图中：
- Root 行显示 `Children: N`，但列表里看不到子节点；
- 点击 `+/-` 也无法正确展开显示子节点行。

目标：
- Root 加载后能立刻渲染第一层子节点；
- 点击 `+/-` 能展开/收起，并可懒加载下一层；
- 不修改 wire/协议与后端接口，仅修复前端 store 的响应式更新问题。

## 具体变更内容

### 修改
- `MyFlowHub-Win/frontend/src/stores/devices.ts`
  - 将 root 节点与子节点由普通对象改为通过 `reactive(...)` 创建（保存为 Proxy）。
  - `nodeIndex` 存储 Proxy 节点引用，`toggle/retry/loadChildren` 对节点字段的修改均通过 Proxy 触发依赖更新。

## 任务映射（plan.md）

- DEV1 - 修复 store 响应式：树节点使用 reactive Proxy
- DEV2 - 验证（命令 + 手工）

## 关键设计决策与权衡

### 为什么需要 `reactive(...)`
问题根因是：root/child 节点创建为普通对象后，被赋值给 `state.root`（响应式对象）的属性；之后代码继续用“原始引用”修改节点字段（如 `root.children = ...`）。
这会绕过 Vue 的 Proxy setter，导致 `computed(() => flattenVisible(state.root))` 依赖未失效重算，从而出现：
- UI 其他部分因 `state.message` 等变更而刷新，但 `visibleNodes` 仍缓存旧结果，只包含 root；
- 导致“看起来 children 数量变了，但列表没展开”的错觉。

本次改为在节点创建阶段即返回 Proxy，并让后续所有节点字段修改都经由 Proxy 完成，从而保证 computed 依赖正确失效。

### 性能影响
- 不引入额外请求、轮询或重复计算。
- 仅将节点对象创建为响应式 Proxy（与树规模线性相关），符合本页“懒加载树”的交互设计。

## 测试与验证

### 命令验证（已执行）
- `GOWORK=off go test ./...`（通过）
- `GOWORK=off wails build -nopackage`（通过）

### 手工冒烟（需使用者确认）
1) Home：Connect → Register/Login
2) Session → Devices：
   - Root 自动加载后应立刻出现第一层 child 行；
   - 点击 child 的 `+`：应出现下一层（或提示 No children）；
   - 切换 Mode / Root Enter 重载：树刷新正确。

## 潜在影响与回滚方案

### 潜在影响
- 影响范围仅限 `Devices` 页面树节点的前端 store 行为；不涉及协议与后端。

### 回滚
- 回滚该变更提交即可恢复到原行为（但会重新出现“子节点不刷新”问题）。

