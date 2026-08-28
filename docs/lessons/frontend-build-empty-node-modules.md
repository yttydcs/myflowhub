# Frontend Build Empty NodeModules

## Summary
- 在 `MyFlowHub-Win` 当前构建链里，`frontend/node_modules` 只要保留目录本身但内容为空，`wails build` 仍可能继续进入 `npm run build`，随后因为缺少 `vite/bin/vite.js` 崩溃。
- 这个症状和 `@babel/parser` 缺失、`frontend/wailsjs/**` 缺失、`frontend/dist` embed 占位问题属于不同层级，需要分别诊断。

## Lookup Hints
- 症状：
  - `Installing frontend dependencies: Done.`
  - `Cannot find module '...frontend\\node_modules\\vite\\bin\\vite.js'`
- 关键词：
  - `vite/bin/vite.js`
  - `empty node_modules`
  - `Installing frontend dependencies: Done.`
  - `package.json.md5`
  - `wails build`
- 快速检查：
  - `frontend/node_modules` 是否只是空目录
  - `frontend/node_modules/vite/bin/vite.js` 是否存在
  - `frontend/node_modules/@vitejs/plugin-vue/package.json` 是否存在
  - `frontend/node_modules/@babel/parser/package.json` 是否存在

## Symptoms
- Wails CLI 在前端编译阶段失败。
- 日志先显示依赖安装完成，再在 `npm run build` 内报 `vite.js` 缺失。
- 直接执行 `npm run build` 也会在找不到 `node_modules/vite/bin/vite.js` 时退出。

## Impact
- Win 客户端无法完成 `wails build` / `wails dev` 的前端阶段。
- 这类失败会伪装成“Vite 入口写法错误”，但实际是依赖目录状态损坏。

## Trigger Conditions
- `frontend/node_modules` 目录存在，但内容为空或缺少关键构建依赖。
- 某次安装或清理操作中断后，只留下空目录。
- 构建入口直接假设本地 Vite CLI 一定存在。

## Root Cause
- 当前仓内前端脚本如果直接调用 `node_modules/vite/bin/vite.js`，会把“目录存在”误当成“依赖完整”。
- 一旦 `node_modules` 进入空目录或残缺状态，构建入口没有二次自检，就会在真正调用 Vite 时才暴露崩溃。

## Investigation Trail
1. 根据用户日志定位到失败点不是业务模块，而是 `npm run build` 的 Vite CLI 入口缺失。
2. 检查主仓 `frontend/node_modules`，确认目录存在但计数为 `0`。
3. 在独立 worktree 中复现实验：
   - 删除整个 `node_modules` 后执行 `wails build`，构建通过。
   - 只保留一个空 `node_modules` 目录后执行 `wails build`，稳定复现 `vite/bin/vite.js` 缺失。
4. 评估修复方案：
   - 只靠人工清理目录
   - 把每次构建改成 `npm ci`
   - 在仓内脚本入口增加依赖自检和缺失时 install
5. 选择最小安全修复：仓内 `run-vite.mjs` 守卫脚本。

## Resolution
- 在 `frontend/package.json` 中，把 `dev` / `build` / `preview` 改为走 `node ./scripts/run-vite.mjs <mode>`。
- 在 `frontend/scripts/run-vite.mjs` 中：
  - 检查关键依赖文件是否存在
  - 若缺失则执行 `npm install`
  - 自愈后仍缺失则显式失败
  - `build` 成功后补回 `dist/placeholder.txt`

## Prevention / Guardrails
- 不要把 `node_modules` “目录存在”当作构建依赖已经完整的证据。
- 排查 Win 前端构建链时，先按错误文本分层：
  - `vite/bin/vite.js` 缺失：先看 `node_modules` 目录完整性
  - `@babel/parser` 缺失：先看 Vue 编译链 direct dependency
  - `../../wailsjs/...` 无法解析：先跑 `wails generate module`
  - `contains no embeddable files`：先看 `frontend/dist/placeholder.txt`
- 若未来升级 Vite 或 Vue 编译链，记得同步维护仓内守卫脚本的关键依赖清单。

## Related Docs
- [2026-04-04_win-frontend-empty-node-modules-guard.md](../change/2026-04-04_win-frontend-empty-node-modules-guard.md)
- [frontend-and-powershell-preflight.md](frontend-and-powershell-preflight.md)
- [构建与 CI](../specs/build-and-ci.md)
- [wails-embed-dist-placeholder.md](wails-embed-dist-placeholder.md)
