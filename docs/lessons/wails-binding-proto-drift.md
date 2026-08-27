# Generated Binding And Protocol Drift

## Summary

vNext 消除了“Win 先引用尚未发布的 Proto 类型”这一跨仓问题，但生成面仍可能漂移：Go schema、binding facade、TypeScript 声明和机器可读 contract 必须来自同一提交。

## Guardrails

- 公共 payload 先在 `protocol/schema_*.go` 中定义和校验；
- 平台 facade 只暴露 canonical SDK/host 能力，不重新定义协议；
- 使用 `./scripts/mfh.ps1 -Action generate -Target generated` 生成绑定；
- CI 检查生成后工作树差异，并在 contract 漂移时失败；
- 不允许用手写临时类型、复制旧 `wailsjs` 或 sibling module `replace` 掩盖缺失 contract。

若生成失败，先在 `GOWORK=off` 下编译相关 Go 包，确认输入 schema 和 facade 合法，再检查 Wails/前端工具链。生成成功只证明接口可导出，仍需前端类型检查、单测和生产构建。

## Related Docs

- [protocol_map.md](../specs/protocol_map.md)
- [build-and-ci.md](../specs/build-and-ci.md)
- [repository-and-module-boundaries.md](../specs/repository-and-module-boundaries.md)
