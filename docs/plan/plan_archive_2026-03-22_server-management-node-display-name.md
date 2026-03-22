# Plan - Server Hubruntime Config Layering

## Workflow Information

- Repo: `MyFlowHub-Server`
- Branch: `feat/management-node-display-name`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-feat-management-node-display-name`
- Current stage: `4`
- Linked main plan:
  - `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-management-node-display-name\plan.md`

## Requirement And Spec Impact

- Requirements impact: `none`
- Specs impact: `clarify`
- Related specs:
  - `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\README.md`

## Checklist

### SERVER1 - Add Persistent Default Layer To Hubruntime

- Task ID: `SERVER1`
- Owner: `SubAgent or Main Agent`
- Worktree path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-feat-management-node-display-name`
- Plan path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-feat-management-node-display-name\plan.md`
- Write set:
  - `hubruntime/runtime.go`
  - `hubruntime/options.go` only if path or behavior needs surfaced options
  - new helper/store files under `hubruntime/` if needed
  - `cmd/hub_server/main.go` only if required for compatibility
  - tests under `hubruntime/` or `tests/`
- Goal:
  - 引入 `config/runtime_config.json` 持久化默认层
  - 构造 `persistent < env < flags/options` 的 effective config
  - 向 management 暴露持久化写入能力
- Acceptance:
  - `config_get` 返回 effective value
  - `config_set(node.display_name)` 写入持久化层并在重启后保留
  - env/flags 仍覆盖持久化层
- Test points:
  - `go test ./hubruntime ./... -count=1`
- Rollback:
  - 回退 layered config 与持久化写入能力

## Risks

- 需要明确 flags/options 与持久化层的冲突处理只影响 effective value，不反向覆写持久化文件
