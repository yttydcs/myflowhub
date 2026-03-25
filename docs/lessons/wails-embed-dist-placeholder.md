# Wails Embed Dist Placeholder

## Summary
- `MyFlowHub-Win` 的 `frontend/dist` 既是前端构建输出目录，也是 Go 侧 `go:embed` 的编译前置条件。
- 只要 `main.go` 仍使用 `//go:embed all:frontend/dist`，就不能让 `frontend/dist` 在未构建前端时变成真正的空目录。

## Symptoms
- `pattern all:frontend/dist: cannot embed directory frontend/dist: contains no embeddable files`
- Wails CLI 卡在 `Generating bindings`
- 日志里会先出现 `Executing: go mod tidy`

## Trigger Conditions
- `frontend/dist` 为空目录
- 占位文件被删除，但 `main.go` 仍保留 `go:embed all:frontend/dist`
- 在未执行前端构建时先执行 `go mod tidy`、`go test` 或 `wails build`

## Root Cause
- `go:embed` 要求目标目录至少包含一个可嵌入文件。
- 如果只依赖易失的临时占位文件，而 `.gitignore`、版本库初始状态和前端 build 补回逻辑没有同步维护，`frontend/dist` 很容易在某个阶段被清空。

## Quick Checks
1. 检查 `frontend/dist` 里是否至少有一个受版本管理的普通文件。
2. 检查 `.gitignore` 是否对白名单指向了实际存在的占位文件。
3. 检查 `frontend/package.json` 的 `build` 脚本是否会在 `vite build` 后补回占位文件。
4. 复现时直接跑 `go mod tidy`，不要只看 `wails build` 的外层包装输出。

## Resolution
- 保留 `go:embed all:frontend/dist` 路径不变。
- 在版本库中提交一个普通占位文件，例如 `frontend/dist/placeholder.txt`。
- 在前端 build 脚本中于 `vite build` 之后重新写回该占位文件。

## Prevention
- 不要把 `frontend/dist` 视为纯运行时产物目录；它同时承担 Go 编译期路径占位职责。
- 占位文件策略变更时，必须同步修改：
  - `.gitignore`
  - `frontend/dist` 跟踪文件
  - `frontend/package.json` build 脚本

## Related Docs
- [2026-03-25_win-embed-dist-placeholder.md](../change/2026-03-25_win-embed-dist-placeholder.md)
- [2026-02-09_remove-fyne.md](../change/2026-02-09_remove-fyne.md)
