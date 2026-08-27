# Wails Bindings Across Product Facades

## Summary

canonical monorepo 中 Desktop、Metrics 等第一方产品仍有不同 Wails facade。生成文件若来自另一个产品，TypeScript 会报告 Go 中明明存在的方法缺失；这是生成输入错配，不是前端业务回归。

## Checks

1. 确认当前 app、Go facade 和 frontend 属于同一产品目录。
2. 删除/重建该产品的生成输出时只使用 canonical 生成入口，不从其他 app 复制。
3. 对照 TypeScript 导出、绑定的 Go methods 和 `sdk/bindings/generated/contracts.json`。
4. 在干净 worktree 重新生成，再运行类型检查、单测和 production build。

生成流程必须对 foreign exports、缺失必需方法和生成后 diff 显式失败。不要为了让 TypeScript 通过而修改正确的 UI 调用去适配错误 binding。

## Related Docs

- [build-and-ci.md](../specs/build-and-ci.md)
- [wails-binding-proto-drift.md](wails-binding-proto-drift.md)
- [metrics-node.md](../features/metrics-node.md)
