# Plan - Management Node Display Name

## Project Goal And Current State

- Goal:
  - 为设备管理引入节点显示名能力，并在 Win `Devices` 中展示与编辑。
  - 让 `hubruntime` 支持持久化默认配置层，使 `node.display_name` 等配置可跨重启保留。
- Current state:
  - Win `Devices` 树和节点详情仍直接展示 `node_id`。
  - `MyFlowHub-Proto` 的 `management.NodeInfo` 没有 `display_name` 字段。
  - `MyFlowHub-SubProto management config_set` 仅调用运行时 `Set`，不知道目标节点是否支持持久化。
  - `MyFlowHub-Server/hubruntime` 仍使用纯内存 `MapConfig` 作为运行时配置。

## Workflow Information

- Current stage: `4`
- Primary repo: `MyFlowHub-Win`
- Primary branch: `feat/management-node-display-name`
- Base branch: `main`
- Primary worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-management-node-display-name`
- Related worktrees:
  - `MyFlowHub-Proto`
    - Branch: `feat/management-node-display-name`
    - Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-feat-management-node-display-name`
    - Responsibility: protocol schema for `list_nodes`
  - `MyFlowHub-SubProto`
    - Branch: `feat/management-node-display-name`
    - Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-feat-management-node-display-name`
    - Responsibility: management handler behavior, display name 回传, optional persistence hook
  - `MyFlowHub-Server`
    - Branch: `feat/management-node-display-name`
    - Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-feat-management-node-display-name`
    - Responsibility: `hubruntime` layered config persistence

## Requirement And Spec Impact

- Requirements impact: `add`
- Specs impact: `add`
- Related requirements:
  - `D:\project\MyFlowHub3\docs\requirements\management-node-display-name.md`
- Related specs:
  - `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`
  - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`

## Confirmed Decisions

- 显示名配置键：`node.display_name`
- 名称语义：节点级全局可编辑名称
- 无显示名时回退：`node_id`
- Win 编辑入口：`Devices` 现有 `Edit` 配置弹窗
- `node_info` 与 `list_nodes` 都要提供显示名
- `config_get` 展示当前生效值
- `hubruntime` 持久化默认层路径：`config/runtime_config.json`
- 配置优先级：`persistent default < env < flags/options`
- 未实现持久化能力的节点继续保持“仅运行期生效”

## Parallelization Assessment

- 可并行任务：
  - `PROTO1`、`SUB1`、`SERVER1`、`WIN1`
- 原因：
  - 写集基本独立，协议、SubProto、Server runtime、Win UI/backend 可分仓推进
- 集成责任：
  - 主 Agent 统一做跨仓联编、验证和冲突消解

## Checklist

### META0 - Workspace Docs And Plan Sync

- Task ID: `META0`
- Owner: `Main Agent`
- Worktree path: `D:\project\MyFlowHub3`
- Plan path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-management-node-display-name\plan.md`
- Write set:
  - `D:\project\MyFlowHub3\docs\requirements\*`
  - `D:\project\MyFlowHub3\docs\specs\*`
  - `D:\project\MyFlowHub3\plan.md`
  - current workflow `plan.md`
- Goal:
  - 记录跨仓稳定需求、技术约束与 active workflow 入口
- Acceptance:
  - requirement/spec impact 明确记录
  - 根控制面可导航到当前 workflow
- Test points:
  - 文档路径存在，索引可读
- Rollback:
  - 回退本 workflow 新增 / 修改的控制面文档

### PROTO1 - Add Display Name To Management NodeInfo

- Task ID: `PROTO1`
- Owner: `SubAgent or Main Agent`
- Worktree path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-feat-management-node-display-name`
- Plan path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-feat-management-node-display-name\plan.md`
- Write set:
  - `protocol/management/types.go`
  - protocol map generation outputs if required by repo conventions
- Goal:
  - 为 `list_nodes` / `list_subtree` 使用的 `NodeInfo` 增加可选 `display_name`
- Acceptance:
  - Proto 编译通过
  - 新字段保持向后兼容
- Test points:
  - `go test ./... -count=1`
- Rollback:
  - 回退 `NodeInfo` schema 变更

### SUB1 - Management Display Name And Optional Persistence Hook

- Task ID: `SUB1`
- Owner: `SubAgent or Main Agent`
- Worktree path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-feat-management-node-display-name`
- Plan path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-feat-management-node-display-name\plan.md`
- Write set:
  - `management/action_nodes.go`
  - `management/action_node_info.go`
  - `management/action_config.go`
  - management tests
- Goal:
  - `list_nodes` / `list_subtree` / `node_info` 回传 `node.display_name`
  - `config_set` 优先调用持久化能力，缺失时回退运行期 `Set`
- Acceptance:
  - 回传字段一致
  - 不支持持久化的节点行为不回退
- Test points:
  - `go test ./... -count=1`
- Rollback:
  - 回退 management handler 改动

### SERVER1 - Layered Persistent Config For Hubruntime

- Task ID: `SERVER1`
- Owner: `SubAgent or Main Agent`
- Worktree path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-feat-management-node-display-name`
- Plan path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-feat-management-node-display-name\plan.md`
- Write set:
  - `hubruntime/*`
  - `cmd/hub_server/main.go` if needed for effective layering compatibility
  - server tests under `tests/` or `hubruntime/`
- Goal:
  - 引入 `config/runtime_config.json` 默认层
  - 形成 effective config，并暴露持久化写入能力
- Acceptance:
  - `config_get` 返回 effective value
  - `config_set(node.display_name)` 在 `hubruntime` 重启后仍保留
  - env/flags 仍能覆盖持久化层
- Test points:
  - `go test ./hubruntime ./... -count=1`
- Rollback:
  - 回退 `hubruntime` layered config 实现

### WIN1 - Show Display Name In Devices And Reuse Existing Edit Entry

- Task ID: `WIN1`
- Owner: `SubAgent or Main Agent`
- Worktree path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-management-node-display-name`
- Plan path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-management-node-display-name\plan.md`
- Write set:
  - `frontend/src/pages/Devices.vue`
  - `frontend/src/stores/devices.ts`
  - related i18n/messages only if needed
  - `internal/services/management/service.go` only if Win binding needs field normalization
- Goal:
  - Win `Devices` 树、详情标题和相关展示优先显示 `display_name`
  - 继续使用现有 `Edit` 配置弹窗编辑 `node.display_name`
- Acceptance:
  - 有显示名时不再显示裸 `Node {nodeId}` 作为主标题
  - 无显示名时行为保持兼容
- Test points:
  - `go test ./... -count=1`
  - `cd frontend && npm run build`
- Rollback:
  - 回退 Win UI / store 改动

### INT1 - Cross Repo Integration And Verification

- Task ID: `INT1`
- Owner: `Main Agent`
- Worktree path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-management-node-display-name`
- Plan path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-management-node-display-name\plan.md`
- Write set:
  - no new feature write set beyond integration fixes
- Goal:
  - 串联四仓改动，完成编译、测试与手工验证步骤整理
- Acceptance:
  - 关键仓库测试通过
  - 计划内任务全部映射完成
- Test points:
  - `MyFlowHub-Proto`: `go test ./... -count=1`
  - `MyFlowHub-SubProto`: `go test ./... -count=1`
  - `MyFlowHub-Server`: `go test ./hubruntime ./... -count=1`
  - `MyFlowHub-Win`: `go test ./... -count=1` and `npm run build`
- Rollback:
  - 分仓按 Task ID 对应改动集回退

## Dependencies

- `SUB1` 依赖 `PROTO1` 的 schema 先落地或并行提供兼容字段
- `WIN1` 依赖 `PROTO1 + SUB1` 的显示名回传
- `SERVER1` 独立推进，但与 `SUB1` 的持久化能力命名需要对齐
- `INT1` 依赖 `PROTO1 + SUB1 + SERVER1 + WIN1`

## Risks And Notes

- `config_get` 返回 effective value 可能让用户看到“与持久化层不同”的值，这是预期语义
- `hubruntime` 需要避免持久化层与 env/flag 双向污染
- 若 protocol map 存在 generated/protected 规则，必须按仓库既有生成方式更新
