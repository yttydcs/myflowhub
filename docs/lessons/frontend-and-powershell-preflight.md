# Frontend And PowerShell Preflight

## Summary

Windows 前端验证常被空/链接依赖目录、foreign Wails bindings、GUI 控制运行时、PowerShell profile/自动变量、编码和原生命令退出码干扰。先证明工具链、控制 harness 和生成输入正确，再把错误归因于 UI 代码。

## Lookup Hints

- 症状：`node_modules` 存在但 compiler 缺失、Vitest context mismatch、`$Args` 参数消失、PowerShell 5.1 parse error、`failed to write kernel assets`、结构化输入被判为非法。
- 关键词：`npm ci`、junction、preserve-symlinks、`wailsjs`、`$LASTEXITCODE`、UTF-8 no BOM、`-NoProfile`、`permit_json must be valid bounded JSON`、`window.open` packaged Wails。
- 快速检查：关键 package 是否真实存在；binding 是否属于当前 app；控制工具是否在第一次交互前失败；PS profile 是否输出额外文本；原生命令 exit code 和产物是否同时通过。

## Symptoms

- Vite/Vitest 在 import-analysis 或 runtime context 初始化阶段失败。
- TypeScript 缺少 Go 中已存在的方法，或出现另一个产品的 exports。
- PowerShell 函数看似收到参数，实际 `$Args` 自动变量覆盖了声明。
- 浏览器 smoke 成功，packaged Wails 辅助窗口无行为。
- GUI 控制工具在第一次点击前失败于 `failed to write kernel assets`，但产品进程本身正常启动。
- 从有效 JSON 文件读取的 Permit/配置进入界面后被判为非法，因为 shell profile 启动文本混入 stdout。

## Impact

把环境、控制 harness 或生成物问题误判为前端回归，产生不必要 UI 改写、不可复现构建、错误的失败结论或只在 Windows 打包后出现的故障。

## Trigger Conditions

- 仅检查目录存在，不检查锁文件和关键依赖。
- 用 junction 复用 `node_modules`，或复制其他项目 `wailsjs`。
- 入口脚本在 Windows PowerShell 5.1 下使用不兼容编码/自动变量名。
- 只在浏览器验证 Wails-specific window/runtime 行为。
- GUI 控制 runtime 缺少或无法写入自身 kernel assets，在产品交互开始前终止。
- 把 PowerShell 命令的整个 stdout 当作结构化字段值，而当前 profile 会打印 conda、banner 或其他启动文本。

## Root Cause

前端源码、Node 依赖图、生成 binding、GUI 控制 harness、PowerShell host 和 packaged runtime 是不同验证层。任何一层被隐式复用或把诊断输出混入数据流，都会制造“源码没变但行为漂移”。

## Investigation Trail

1. 从干净 app 目录执行锁文件驱动的 `npm ci`，确认 compiler/runtime package。
2. 对照 Go facade、TypeScript exports 和 generated contract，排除 foreign bindings。
3. 在 PowerShell 7 与 5.1 分别 parse 入口，检查 `$LASTEXITCODE` 与产物。
4. 对辅助窗口/session hydration 做 packaged click smoke，而不只看浏览器。
5. 若控制工具在首次交互前失败，分别确认产品进程、窗口和控制 runtime；相同初始化签名只能记为 harness/environment blocked。
6. 结构化输入先用无 profile shell 读取并独立 parse，不能把带 banner 的 stdout 直接填入 GUI。

## Resolution

- 每个 app 使用真实、锁定的依赖安装；确需 symlink 时显式验证 preserve-symlinks。
- 绑定只由 canonical generate 入口重建，不跨项目复制。
- 不用 `$Args` 等自动变量名；关键 PS 5.1 入口保持 ASCII 或兼容编码。
- detached window 主动 hydrate live session snapshot，Wails-specific 行为使用共享 adapter/fallback。
- 自动化数据通道使用无 profile shell，结构化 payload 在进入 UI 前完成 parse/边界检查。
- 控制 runtime 初始化失败与产品操作失败分开记录；不能用自制自动化旁路伪造受支持 GUI 证据。

## Prevention / Guardrails

- CI 同时运行 type-check、unit test、production build 和 packaged smoke。
- 原生命令必须同时满足退出码为零和预期产物存在。
- worktree 首次前端验证先完成 generation/dependency preflight。
- GUI 重验前确认受支持控制 runtime 可创建 kernel assets；payload 采集使用 `-NoProfile` 或等价无启动输出入口。

## Related Docs

- [Build and CI](../specs/build-and-ci.md)
- [Desktop feature](../features/desktop.md)
- [WailsJS missing](frontend-worktree-wailsjs-missing.md)
- [Generated binding drift](wails-binding-proto-drift.md)
- [vNext full migration](../change/2026-08-27_vnext-full-migration.md)
- [NodeHost、Enrollment 与 Profile 生命周期收敛](../change/2026-09-01_nodehost-enrollment-profile-convergence.md)
