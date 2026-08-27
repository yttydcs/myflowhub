# Canonical run-dev Target Selection

## Summary

旧多仓环境中，`run-dev.ps1` 可能启动了不含目标 SubProto 的 Server 或被父级 `go.work` 污染。vNext 只有 canonical `cmd/mfh-hub` 和同仓第一方应用；启动脚本不得在旧仓库或历史 worktree 之间自动选择实现。

## Current Debugging Order

当 Variable、Stream 或 Command 请求统一超时时：

1. 查看启动摘要，确认进程来自当前 canonical worktree；
2. 确认 `GOWORK=off` 且实际入口是 `cmd/mfh-hub`；
3. 检查目标资源是否出现在 `system/catalog`；
4. 区分链路未连接、资源未注册、权限拒绝、deadline 到期和真实 handler 错误；
5. 只在资源存在且授权成功后继续排查 UI/SDK。

启动脚本必须显式失败，不能静默回退到 `repo/MyFlowHub-*`、旧 Server 或替代协议实现。

## Related Docs

- [build-and-ci.md](../specs/build-and-ci.md)
- [resource-catalog.md](../specs/resource-catalog.md)
- [operational-lifecycle.md](../specs/operational-lifecycle.md)
