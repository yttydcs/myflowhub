# Retired Cross-Repository Release Chain

## Summary

旧架构通过多个 Go module、`go.work`、sibling `replace` 和逐仓 tag 传递内部 API，导致“本地联调通过、独立构建失败”和缓存命中旧 tag。vNext 已把这些组件合并为一个仓库、一个 module 和一个版本，因此内部发布链不再是合法开发路径。

## Current Rule

- canonical 构建始终设置 `GOWORK=off`；
- 协议、runtime、SDK、host 与第一方产品在同一提交中修改和验证；
- 跨语言公开面由生成 contract 表达，不通过内部仓库 tag 拼接；
- CI 必须从干净 checkout 运行根级生成、测试和构建矩阵；
- 不得恢复 `repo/MyFlowHub-*`、本地 sibling `replace` 或旧 module path 作为 fallback。

如果只有 workspace 环境能通过，优先检查未跟踪生成物、环境依赖或边界违规，而不是建立新的多仓发布顺序。

## Related Docs

- [repository-and-module-boundaries.md](../specs/repository-and-module-boundaries.md)
- [build-and-ci.md](../specs/build-and-ci.md)
- [2026-08-27_canonical-monorepo-unified-node-runtime.md](../change/2026-08-27_canonical-monorepo-unified-node-runtime.md)
