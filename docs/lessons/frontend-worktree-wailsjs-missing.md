# Frontend Worktree WailsJS Missing

## Summary

新的 worktree 不会天然拥有 Wails 生成绑定。若前端在 import-analysis 阶段报告 `wailsjs` 缺失，应先生成 canonical bindings，再判断页面代码是否回归。

## Checks And Resolution

1. 确认当前目录是 canonical monorepo worktree，并设置 `GOWORK=off`。
2. 运行 `./scripts/mfh.ps1 -Action generate -Target generated`。
3. 检查预期的 Go、TypeScript 和 runtime binding 是否生成。
4. 再执行前端单测和生产构建。

不要从另一个旧仓库或工作副本复制 `wailsjs`：即使编译通过，也可能让前端方法签名与当前 Go API 漂移。

## Related Docs

- [build-and-ci.md](../specs/build-and-ci.md)
- [desktop.md](../features/desktop.md)
- [wails-binding-proto-drift.md](wails-binding-proto-drift.md)
