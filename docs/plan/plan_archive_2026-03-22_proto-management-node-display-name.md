# Plan - Proto Management Display Name

## Workflow Information

- Repo: `MyFlowHub-Proto`
- Branch: `feat/management-node-display-name`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-feat-management-node-display-name`
- Current stage: `4`
- Linked main plan:
  - `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-management-node-display-name\plan.md`

## Requirement And Spec Impact

- Requirements impact: `none`
- Specs impact: `clarify`
- Related specs:
  - `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`
  - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`

## Checklist

### PROTO1 - Extend Management NodeInfo Schema

- Task ID: `PROTO1`
- Owner: `SubAgent or Main Agent`
- Worktree path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-feat-management-node-display-name`
- Plan path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-feat-management-node-display-name\plan.md`
- Write set:
  - `protocol/management/types.go`
  - protocol map outputs if required
- Goal:
  - 在 `NodeInfo` 中增加可选 `display_name`
- Acceptance:
  - `ListNodesResp` / `ListSubtreeResp` 的 schema 能表达显示名
  - 对旧消费者保持 JSON 向后兼容
- Test points:
  - `go test ./... -count=1`
- Rollback:
  - 回退 schema 变更

## Notes

- `node_info` 使用 `items map[string]string`，本仓通常不需要额外 schema 变更来表达 `display_name`
