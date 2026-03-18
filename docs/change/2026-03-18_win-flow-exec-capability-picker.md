# 变更背景 / 目标

- 背景：`flow` 的 `exec` 节点要求严格指定 `target + method`，但在 Win 编辑器中完全手填容易出错。
- 目标：保持执行语义不变（仍为定点调用），新增“能力选择器”辅助用户从可用能力列表回填节点配置。

# 具体变更内容（新增 / 修改 / 删除）

## 1) FlowService 增加 exec 能力查询接口

- 文件：`repo/MyFlowHub-Win/internal/services/flow/service.go`
- 新增：
  - `ExecCapQuery(...)`
  - `ExecCapQuerySimple(...)`
- 行为：
  - 通过 `exec.cap_query`（SubProto=7）查询能力路由；
  - 统一复用 Win 侧错误包装，返回 `CapQueryResp` 给前端。

## 2) Flow Store 新增能力查询与回填

- 文件：`repo/MyFlowHub-Win/frontend/src/stores/flow.ts`
- 新增状态：
  - `execCapabilities`
  - `execCapabilitiesLoading`
- 新增动作：
  - `queryExecCapabilities(methodPrefix)`：查询能力列表；
  - `applyExecCapability(key)`：把所选能力回填到当前 `exec` 节点（写入 `method` 与 `target=provider_node`）。

## 3) Flow 页面新增能力选择器 UI

- 文件：`repo/MyFlowHub-Win/frontend/src/pages/Flow.vue`
- 在 `Node Detail` 的 `exec` 区域新增：
  - `Load Capabilities` 按钮；
  - capability 下拉；
  - `Use Selected Capability` 回填按钮。

# 对应 plan 任务映射

- Task A：后端能力查询接口
  - `repo/MyFlowHub-Win/internal/services/flow/service.go`
- Task B：前端 store 接口与状态
  - `repo/MyFlowHub-Win/frontend/src/stores/flow.ts`
- Task C：编辑器 UI 交互
  - `repo/MyFlowHub-Win/frontend/src/pages/Flow.vue`

# 关键设计决策与权衡

- 保持 `exec` 严格定点调用：仅做“配置辅助”，不改运行时 `target` 强制语义。
- 查询默认按“当前 Method 前缀”过滤，兼顾可发现性与结果规模。
- 回填只写 `method + target`，不做自动执行策略，避免引入不确定性。

# 测试与验证方式 / 结果

- `repo/MyFlowHub-Win`：`go test ./...`（通过）
- `repo/MyFlowHub-Win/frontend`：`npm run build`（通过）

# 潜在影响与回滚方案

- 潜在影响：
  - 能力查询依赖 `exec.cap.query` 权限，若权限不足会返回失败提示。
  - UI 查询结果取决于当前 executor 视角（网络拓扑变化会影响可见能力）。
- 回滚方案：
  - 回退 `FlowService.ExecCapQuery*` 接口；
  - 回退 Flow Store / 页面中的 capability picker 相关状态与按钮。

# 实施例外说明

- 本轮在现有多仓主工作区直接实施并提交（未新建独占 worktree）。
- 归档文档已直接保存在全局 `docs/change/`，用于对齐后续计划与发布审计。
