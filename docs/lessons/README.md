# Lessons

存放 MyFlowHub3 meta workspace 可复用的复盘、陷阱与防错经验。

## How To Use
- 只有当某个问题具备复用价值，才在这里新增叶子文档。
- 单次 workflow 结果优先放在 `change/`，不要把 `lessons/` 当作变更日志。

## What Belongs Here
- 重复出现的问题模式
- 调试路径与根因总结
- 未来 workflow 的防错规则

## Runtime And Architecture

- [desktop-binding-reconnect-and-admission-diagnostics.md](desktop-binding-reconnect-and-admission-diagnostics.md)：Desktop binding 重试必须重建单次生命周期 client，并在超时、permit 与准入失败之间保留可操作诊断。
- [authority-routing-and-subscription-state.md](authority-routing-and-subscription-state.md)：desired/attached 状态、relay 字段、pending 返程和 generation 边界。
- [authority-local-admin-actions.md](authority-local-admin-actions.md)：remote authority 管理如何沿节点树路由并保留原始授权主体。
- [session-replacement-generation-cleanup.md](session-replacement-generation-cleanup.md)：同父重连、epoch 与 generation-scoped cleanup。
- [observable-side-effects-and-generated-contracts.md](observable-side-effects-and-generated-contracts.md)：替代 mutation 路径的订阅副作用与单一 contract 真相。

## Platform And Toolchain

- [go-cross-compile-tests-on-windows.md](go-cross-compile-tests-on-windows.md)：Windows 上设置非本机 `GOOS` 后，普通 `go test` 会尝试执行目标二进制；跨平台门禁必须区分 compile-only 与真实 target runtime。
- [brand-symbol-optical-size-variants.md](brand-symbol-optical-size-variants.md)：品牌主标在 16/20/24px 必须使用独立光学校正版，避免 tray/favicon 负空间闭合。
- [windows-clean-checkout-eol-and-generated-drift.md](windows-clean-checkout-eol-and-generated-drift.md)：Windows `core.autocrlf`、gofmt 全仓误报、Wails/Vite byte drift、Gradle short TEMP 与 JDK 选择。
- [android-runtime-and-mobile-bindings.md](android-runtime-and-mobile-bindings.md)：sticky restart、live session、FGS/RFCOMM、URI staging 与真实 AAR 证明。
- [embedded-toolchain-and-board-preflight.md](embedded-toolchain-and-board-preflight.md)：ESP-IDF/MicroPython 工具链、真板、网络和打包预检。
- [embedded-esp32s3-ws2812-board-smoke.md](embedded-esp32s3-ws2812-board-smoke.md)：串口、固件、Wi-Fi、WS2812 与真板 smoke。
- [micropython-edge-local-auth-persistence-restore.md](micropython-edge-local-auth-persistence-restore.md)：边缘节点 local auth 恢复与 child route 索引。
- [flutter-windows-sdk-shared-bat-git.md](flutter-windows-sdk-shared-bat-git.md)：Flutter Windows `shared.bat`、`$git` 与启动卡死。
- [frontend-and-powershell-preflight.md](frontend-and-powershell-preflight.md)：npm、Vitest、Wails、PowerShell 自动变量与编码陷阱。
- [frontend-build-empty-node-modules.md](frontend-build-empty-node-modules.md)：空或残缺 `node_modules` 导致 Vite/Wails 构建失败。
- [frontend-worktree-wailsjs-missing.md](frontend-worktree-wailsjs-missing.md)：新 worktree 缺 canonical Wails bindings。
- [wails-binding-proto-drift.md](wails-binding-proto-drift.md)：schema、facade、TypeScript 与机器 contract 一致性。
- [wails-bindings-cross-project.md](wails-bindings-cross-project.md)：多个第一方前端 facade 的生成输入与导出面错配。
- [wails-embed-dist-placeholder.md](wails-embed-dist-placeholder.md)：`go:embed` 与空前端产物目录。
- [powershell-args-automatic-variable.md](powershell-args-automatic-variable.md)：`$Args` 自动变量吞掉透传参数。
- [powershell-nested-array-flattening.md](powershell-nested-array-flattening.md)：pipeline enumeration 导致安全预检计数漂移。
- [powershell-utf8-nobom-parse.md](powershell-utf8-nobom-parse.md)：PowerShell 5.1 解析 UTF-8 无 BOM 中文脚本失败。

## Retired Multi-Repository Failure Modes

- [cross-repo-semver-release.md](cross-repo-semver-release.md)：为何旧内部 tag 链已被单仓原子版本取代。
- [run-dev-stream-server-selection.md](run-dev-stream-server-selection.md)：为何启动脚本只能选择 canonical Hub，不能回退到旧 Server/worktree。

## Rules
- 使用稳定文件名，不使用日期前缀。
- 每条 lesson 应回链到当前 `feature/spec`；历史 `change` 只能作为证据，不能成为当前技术真相。
