# run-dev-stream-server-selection

## Summary
- 当 Win `Stream` 页面报 `stream announce: request timed out`，根因未必在 Win 或 `stream` handler 本身，也可能是 root `run-dev.ps1` 启动了不含 `stream` 的主线 Server。
- 即使已经找到了正确的 Server worktree，如果它位于 workspace 根 `go.work` 之下，未显式 `GOWORK=off` 仍可能把进程重新污染回旧模块图。

## Lookup Hints
- `stream announce: request timed out`
- `list_sources await failed`
- `list_consumers await failed`
- `run-dev.ps1`
- `server-stream-subproto-design`
- `newStreamHandler`
- `GOWORK=off`
- `go.work`

## Symptoms
- Win `Stream` 页创建本地 source / consumer 失败。
- Win 日志中对 `target=1` 的 `stream list_sources`、`stream list_consumers`、`stream announce` 全部超时。
- `run-dev.ps1` 看起来已经正常拉起 Server，但实际 Hub 不响应任何 `stream` 控制面请求。
- 在 `server-stream-subproto-design` worktree 下直接跑测试时，默认模式失败，切到 `GOWORK=off` 却能通过。

## Impact
- `Stream` 页核心控制面不可用。
- 用户会误以为 Win 页面改动、SDK 或 `stream` 本地 owner 仍未生效。
- root 启动脚本制造“本地明明修了，运行时却还是旧行为”的错觉。

## Trigger Conditions
- 使用 workspace 根 `.\scripts\run-dev.ps1` 作为默认启动入口。
- 主线 `repo/MyFlowHub-Server` 尚未合入 `stream` handler。
- 可用的 `stream` Hub 仅存在于 `worktrees/server-stream-subproto-design`。
- alternate Server worktree 继承了根 `go.work`。

## Root Cause
- 启动脚本默认把 `repo/MyFlowHub-Server` 当成正确的 Hub 项目目录，但该主线并不包含 `stream`。
- 同时，alternate Server worktree 处在 root `go.work` 的向上搜索范围内；不显式关闭 workspace 模式时，Go 会优先采用根模块图，而不是该 worktree 的已发布 `Proto / SubProto stream` 依赖。

## Investigation Trail
- 先看 Win 日志，确认超时都发生在 `target=1` 的 Hub 控制面。
- 对比：
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\modules\defaultset\hub.go`
  - `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design\modules\defaultset\hub.go`
- 检查是否存在 `newStreamHandler`。
- 在 `server-stream-subproto-design` 下执行 `go env GOWORK`，确认它会命中根 `D:\project\MyFlowHub3\go.work`。
- 对比：
  - `go test ./tests -run TestStreamRootHubConnectDisconnect -count=1`
  - `$env:GOWORK='off'; go test ./tests -run TestStreamRootHubConnectDisconnect -count=1`
- 若后者通过、前者失败，说明不仅要切对 Server 路径，还要处理 workspace 污染。

## Resolution
- 在 root `run-dev.ps1` 中增加 Server 项目目录选择层：
  - 显式 `-ServerProjectDir` 优先
  - 主线 Server 若已支持 `stream`，优先主线
  - 否则自动切到 `worktrees/server-stream-subproto-design`
- 新增 `stream` 支持探测与启动摘要，直接显示当前 Server 项目、来源和 `GOWORK` 模式。
- 对非主线 Server worktree 默认附加 `GOWORK=off`。
- 当用户显式切回不支持 `stream` 的主线 Server 时，脚本在启动前直接 warning，而不是把问题留到 UI 层才暴露。

## Prevention / Guardrails
- 任何 root 级启动脚本都应输出“实际启动的是哪个项目目录”，不要把它隐藏成默认假设。
- 当新能力仅存在于 alternate worktree / 未合入主线时，脚本必须显式探测能力支持，而不是继续默认主线路径。
- 对位于 workspace 子目录的 alternate Go module，只要目标是稳定冒烟而不是整仓联调，就默认优先 `GOWORK=off`。
- 遇到 UI 层统一 timeout 时，优先确认“目标 Hub 是否真的启动了对应 handler”，再往 SDK/前端继续深挖。

## Related Docs
- [2026-03-29_root-run-dev-stream-server.md](../change/2026-03-29_root-run-dev-stream-server.md)
- [2026-03-29_root-run-dev-wails-gowork.md](../change/2026-03-29_root-run-dev-wails-gowork.md)
- `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design\docs\change\2026-03-28_stream-server-release.md`
- `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design\docs\lessons\stream-control-plane-validation.md`
