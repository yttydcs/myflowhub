# Frontend Worktree WailsJS Missing

## Summary

- `MyFlowHub-Win` 的 `frontend/wailsjs` 是前端解析 Wails 绑定的生成态目录，但它被 `.gitignore` 忽略，不会自动出现在新的 worktree。
- 如果直接在新 worktree 的 `frontend/` 下执行 `npm test` 或 `npm run build`，Vite 可能先报模块解析失败，而不是页面代码回归。

## Lookup Hints

- 症状
  - `Failed to resolve import "../../wailsjs/runtime/runtime"`
  - `vite:import-analysis`
  - `src/stores/*.ts` 无法解析 `../../wailsjs/...`
- 触发条件
  - 新建 worktree 后直接跑前端测试或构建
  - worktree 里没有生成过 Wails 前端 bindings
- 关键词
  - `frontend/wailsjs`
  - `runtime/runtime`
  - `EventsOn`
  - `vite:import-analysis`
  - `Failed to resolve import`
- 快速检查
  - `Test-Path frontend/wailsjs`
  - 对比主仓是否已有 `repo/MyFlowHub-Win/frontend/wailsjs`
  - 若不存在，先执行 `wails generate module` 或从已有工作副本复制该目录

## Symptoms

- `npm test` 在导入 store 时直接失败，而不是某个用例断言失败
- `npm run build` 在 Vite transform/import-analysis 阶段终止
- 错误文本集中在 `../../wailsjs/runtime/runtime` 或 `../../wailsjs/go/...`

## Impact

- 无法在 worktree 内完成前端测试与构建验证
- 容易把生成态依赖缺失误判为页面代码改坏

## Trigger Conditions

- `frontend/wailsjs` 被 `.gitignore` 排除
- 新 worktree 只带入受版本管理文件，没有执行过 Wails bindings 生成流程
- 前端代码通过相对路径直接依赖 `../../wailsjs/...`

## Root Cause

- `frontend/wailsjs` 不是版本库跟踪文件，而是由 Wails 生成的前端 bindings 产物。
- worktree 创建时不会自动补齐被忽略的生成目录，因此新的 worktree 在未生成 bindings 前天然缺少该依赖。

## Investigation Trail

1. `npm test` 首次失败，错误指向 `src/stores/stream.ts` 中的 `../../wailsjs/runtime/runtime`
2. 检查 worktree，确认 `frontend/wailsjs` 不存在
3. 对比主仓，确认 `repo/MyFlowHub-Win/frontend/wailsjs` 已存在可用 bindings
4. 补齐该目录后重新执行 `npm test` / `npm run build`，验证恢复正常

## Resolution

- 在进行 worktree 前端验证前，先补齐 `frontend/wailsjs`
- 优先方案：在当前 worktree 执行 `wails generate module`
- 快速验证方案：从已有、版本一致的工作副本复制 `frontend/wailsjs` 到当前 worktree

## Prevention / Guardrails

- 新 worktree 第一次跑前端测试前，先检查 `frontend/wailsjs` 是否存在
- 如果 workflow 只做前端变更，也不要跳过 Wails 生成态依赖检查
- 将该问题记录在 lessons 中，而不是只留在一次性的 change 归档里

## Related Docs

- [2026-03-29_win-stream-page-i18n.md](../change/2026-03-29_win-stream-page-i18n.md)
- [wails-embed-dist-placeholder.md](wails-embed-dist-placeholder.md)
