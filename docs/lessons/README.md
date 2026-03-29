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
- [authority-local-admin-actions.md](authority-local-admin-actions.md)
  - 症状：remote authority 下审批 / permit 管理仍 timeout、仍提示 `requires authority-local session`，或 authority 拒绝 routed source。
  - 关键词：`authorityId != sourceId`、`routed source`、`requires authority-local session`、`list_register_permits`。
- [wails-binding-proto-drift.md](wails-binding-proto-drift.md)
  - 症状：`wails generate module` / `go test` 报 `undefined: flow.DetailReq`、`undefined: flow.ActionDetail`，或 `module ... myflowhub-proto ... replaced but not required`
  - 关键词：`GOWORK=off`、`myflowhub-proto`、`flow.detail`、`local typed payload`、`protocol/stream`。
- [wails-bindings-cross-project.md](wails-bindings-cross-project.md)
  - 症状：`npm run build` / `vue-tsc` 提示 `BootstrapGet`、`MetricsSettingsGet`、`StartReporting` 缺导出，但 `windows/app.go` 明明存在这些方法。
  - 关键词：`AboutState`、`FlowProjectsState`、`SaveHomeState`、`App.d.ts`、`wailsjs`、cross-project bindings。
- [frontend-worktree-wailsjs-missing.md](frontend-worktree-wailsjs-missing.md)
  - 症状：新 worktree 的 `npm test` / `npm run build` 报 `Failed to resolve import "../../wailsjs/runtime/runtime"`。
  - 关键词：`frontend/wailsjs`、`runtime/runtime`、`vite:import-analysis`、`EventsOn`。
- [cross-repo-semver-release.md](cross-repo-semver-release.md)
  - 症状：`go.work` 或本地 sibling worktree 通过，但 `GOWORK=off` / 远端 CI 仍失败，常见为 `NewHandlerWithDeps`、`SharedExecCapQueryBroker`、`ActionDelete` 等缺符号。
  - 关键词：未发布 tag、默认分支 checkout、`replace ../../...`、`go list -m`、`defaultset`、`runtimedeps`。
- [wails-embed-dist-placeholder.md](wails-embed-dist-placeholder.md)
  - 症状：Wails 在 `Generating bindings` 阶段报 `pattern all:frontend/dist: cannot embed directory frontend/dist: contains no embeddable files`。
  - 关键词：`go:embed all:frontend/dist`、`go mod tidy`、`frontend/dist`、`placeholder.txt`。

## Rules
- 使用稳定文件名，不使用日期前缀。
- 每条 lesson 应回链到对应的 `change` 或 `spec`。
