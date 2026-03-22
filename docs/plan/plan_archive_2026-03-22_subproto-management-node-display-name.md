# Plan - SubProto Management Display Name And Persistence Hook

## Workflow Information

- Repo: `MyFlowHub-SubProto`
- Branch: `feat/management-node-display-name`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-feat-management-node-display-name`
- Current stage: `4`
- Linked main plan:
  - `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-management-node-display-name\plan.md`

## Requirement And Spec Impact

- Requirements impact: `none`
- Specs impact: `clarify`
- Related specs:
  - `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`

## Checklist

### SUB1 - Return Display Name In Management Queries

- Task ID: `SUB1`
- Owner: `SubAgent or Main Agent`
- Worktree path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-feat-management-node-display-name`
- Plan path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-feat-management-node-display-name\plan.md`
- Write set:
  - `management/action_nodes.go`
  - `management/action_node_info.go`
  - related tests
- Goal:
  - 从 effective config 读取 `node.display_name`
  - 在 `list_nodes` / `list_subtree` / `node_info` 中返回显示名
- Acceptance:
  - 名称为空时不返回脏值
  - `node_info.items["display_name"]` 与 `list_nodes.nodes[].display_name` 语义一致
- Test points:
  - `go test ./... -count=1`
- Rollback:
  - 回退 management 显示名回传逻辑

### SUB2 - Optional Persistent Config Write Capability

- Task ID: `SUB2`
- Owner: `SubAgent or Main Agent`
- Worktree path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-feat-management-node-display-name`
- Plan path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-feat-management-node-display-name\plan.md`
- Write set:
  - `management/action_config.go`
  - related tests
- Goal:
  - `config_set` 优先调用目标配置的持久化能力
  - 缺失持久化能力时回退到现有 `Set`
- Acceptance:
  - 旧节点行为不变
  - 持久化能力返回错误时，`config_set_resp` 反映失败
- Test points:
  - `go test ./... -count=1`
- Rollback:
  - 回退 optional persistence hook
