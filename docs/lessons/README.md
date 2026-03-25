# Lessons

存放 MyFlowHub3 meta workspace 可复用的复盘、陷阱与防错经验。

## How To Use
- 只有当某个问题具备复用价值，才在这里新增叶子文档。
- 单次 workflow 结果优先放在 `change/`，不要把 `lessons/` 当作变更日志。

## What Belongs Here
- 重复出现的问题模式
- 调试路径与根因总结
- 未来 workflow 的防错规则

## Current Status
- [cross-repo-semver-release.md](cross-repo-semver-release.md)
  - 症状：`go.work` 或本地 sibling worktree 通过，但 `GOWORK=off` / 远端 CI 仍失败，常见为 `NewHandlerWithDeps`、`SharedExecCapQueryBroker`、`ActionDelete` 等缺符号。
  - 关键词：未发布 tag、默认分支 checkout、`replace ../../...`、`go list -m`、`defaultset`、`runtimedeps`。

## Rules
- 使用稳定文件名，不使用日期前缀。
- 每条 lesson 应回链到对应的 `change` 或 `spec`。
