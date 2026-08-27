# Lessons

存放 MyFlowHub3 可复用的调试路径、根因和防错规则。单次 workflow 结果放在 `change/`；这里仅保留跨实现、跨产品或可能再次发生的长期经验。

## Runtime And Architecture

- [authority-routing-and-subscription-state.md](authority-routing-and-subscription-state.md)：desired/attached 状态、relay 字段、pending 返程和 generation 边界。
- [authority-local-admin-actions.md](authority-local-admin-actions.md)：管理操作如何沿节点树路由并保留原始授权主体。
- [session-replacement-generation-cleanup.md](session-replacement-generation-cleanup.md)：同父重连、epoch 与 generation-scoped cleanup。
- [observable-side-effects-and-generated-contracts.md](observable-side-effects-and-generated-contracts.md)：替代 mutation 路径的订阅副作用与单一 contract 真相。

## Platform And Toolchain

- [android-runtime-and-mobile-bindings.md](android-runtime-and-mobile-bindings.md)：sticky restart、live session、FGS/RFCOMM、URI staging 与真实 AAR 证明。
- [embedded-toolchain-and-board-preflight.md](embedded-toolchain-and-board-preflight.md)：ESP-IDF/MicroPython 工具链、真板、网络和打包预检。
- [frontend-and-powershell-preflight.md](frontend-and-powershell-preflight.md)：npm、Vitest、Wails、PowerShell 自动变量与编码陷阱。
- [frontend-worktree-wailsjs-missing.md](frontend-worktree-wailsjs-missing.md)：新 worktree 生成 canonical Wails bindings。
- [wails-binding-proto-drift.md](wails-binding-proto-drift.md)：schema、facade、TypeScript 与机器 contract 一致性。
- [wails-bindings-cross-project.md](wails-bindings-cross-project.md)：多个第一方前端 facade 的生成输入与导出面错配。
- [wails-embed-dist-placeholder.md](wails-embed-dist-placeholder.md)：`go:embed` 与空前端产物目录。

## Retired Multi-Repository Failure Modes

- [cross-repo-semver-release.md](cross-repo-semver-release.md)：为何旧内部 tag 链已被单仓原子版本取代。
- [run-dev-stream-server-selection.md](run-dev-stream-server-selection.md)：为何启动脚本只能选择 canonical Hub，不能回退到旧 Server/worktree。

## Rules

- 使用稳定文件名，不使用日期前缀。
- 每条 lesson 回链到当前 spec/feature；历史 change 只能作为证据，不能成为当前技术真相。
- 旧仓文档中的有效经验应在这里综合提炼，不整目录复制，也不保留已废弃 API 表。
