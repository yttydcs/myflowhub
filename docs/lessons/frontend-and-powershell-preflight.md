# Frontend And PowerShell Preflight

## Summary

Windows 前端验证常被空/链接依赖目录、foreign Wails bindings、PowerShell 自动变量、编码和原生命令退出码干扰。先证明工具链和生成输入正确，再把错误归因于 UI 代码。

## Lookup Hints

- 症状：`node_modules` 存在但 compiler 缺失、Vitest context mismatch、`$Args` 参数消失、PowerShell 5.1 parse error、脚本打印完成却没有产物。
- 关键词：`npm ci`、junction、preserve-symlinks、`wailsjs`、`$LASTEXITCODE`、UTF-8 no BOM、`window.open` packaged Wails。
- 快速检查：关键 package 是否真实存在；binding 是否属于当前 app；PS 5.1 能否 parse；原生命令 exit code 和产物是否同时通过。

## Symptoms

- Vite/Vitest 在 import-analysis 或 runtime context 初始化阶段失败。
- TypeScript 缺少 Go 中已存在的方法，或出现另一个产品的 exports。
- PowerShell 函数看似收到参数，实际 `$Args` 自动变量覆盖了声明。
- 浏览器 smoke 成功，packaged Wails 辅助窗口无行为。

## Impact

把环境/生成物问题误判为前端回归，产生不必要 UI 改写、不可复现构建或只在 Windows 打包后出现的故障。

## Trigger Conditions

- 仅检查目录存在，不检查锁文件和关键依赖。
- 用 junction 复用 `node_modules`，或复制其他项目 `wailsjs`。
- 入口脚本在 Windows PowerShell 5.1 下使用不兼容编码/自动变量名。
- 只在浏览器验证 Wails-specific window/runtime 行为。

## Root Cause

前端源码、Node 依赖图、生成 binding、PowerShell host 和 packaged runtime 是不同验证层。任何一层被隐式复用都会制造“源码没变但行为漂移”。

## Investigation Trail

1. 从干净 app 目录执行锁文件驱动的 `npm ci`，确认 compiler/runtime package。
2. 对照 Go facade、TypeScript exports 和 generated contract，排除 foreign bindings。
3. 在 PowerShell 7 与 5.1 分别 parse 入口，检查 `$LASTEXITCODE` 与产物。
4. 对辅助窗口/session hydration 做 packaged click smoke，而不只看浏览器。

## Resolution

- 每个 app 使用真实、锁定的依赖安装；确需 symlink 时显式验证 preserve-symlinks。
- 绑定只由 canonical generate 入口重建，不跨项目复制。
- 不用 `$Args` 等自动变量名；关键 PS 5.1 入口保持 ASCII 或兼容编码。
- detached window 主动 hydrate live session snapshot，Wails-specific 行为使用共享 adapter/fallback。

## Prevention / Guardrails

- CI 同时运行 type-check、unit test、production build 和 packaged smoke。
- 原生命令必须同时满足退出码为零和预期产物存在。
- worktree 首次前端验证先完成 generation/dependency preflight。

## Related Docs

- [Build and CI](../specs/build-and-ci.md)
- [Desktop feature](../features/desktop.md)
- [WailsJS missing](frontend-worktree-wailsjs-missing.md)
- [Generated binding drift](wails-binding-proto-drift.md)
- [vNext full migration](../change/2026-08-27_vnext-full-migration.md)
