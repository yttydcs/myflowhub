# 2026-03-26_metricsnode-wails-bindings-sync

## 变更背景 / 目标
- 用户在 `windows/frontend` 执行 `npm run build` 时看到 `TS2305`，提示 `../wailsjs/go/main/App` 缺少 `BootstrapGet`、`MetricsSettingsGet`、`StartReporting` 等导出。
- 本次目标不是改 UI 逻辑，而是确认 MetricsNode Windows 的 Wails 绑定是否被错误生成物污染，并给出可重复的恢复路径。

## 具体变更内容
- 在专用 worktree 中验证了 clean baseline：
  - `windows/frontend/wailsjs/go/main/App.d.ts` 正常导出 `BootstrapGet`、`Status`、`MetricsSettingsGet` 等 MetricsNode 接口。
  - `windows/frontend/src/App.vue` 与当前 Go `App` 绑定面一致。
  - `npm run build` 与 `scripts/build-windows.ps1` 都可通过。
- 为 `scripts/build-windows.ps1` 增加了绑定面校验：
  - 要求 `App.d.ts` 包含 MetricsNode 必需导出，如 `BootstrapGet`、`Status`、`MetricsSettingsGet`。
  - 若出现 `AboutState`、`FlowProjectsState`、`SaveHomeState` 这类外部应用导出，则直接失败并给出明确错误。
- 在 `windows/README.md` 增加了 stale / foreign Wails bindings 的排查说明与恢复命令。
- 新增 lesson 与 docs 索引，避免下次只能靠翻聊天记录或 change log 定位。

## Requirements impact
- none

## Specs impact
- none

## Lessons impact
- updated

## Related requirements
- none

## Related specs
- none

## Related lessons
- [../lessons/wails-bindings-cross-project.md](../lessons/wails-bindings-cross-project.md)

## 对应 plan.md 任务映射
- `MNWB-1`：验证 clean worktree 绑定面与前端 build 状态。
- `MNWB-2`：对构建脚本增加绑定面保护，并补 README 排查说明。
- `MNWB-3`：完成 review、change archive 与 lesson 归档。

## 经验 / 教训摘要
- 这次报错表面上是 Vue/TypeScript 缺导出，实际根因是本地 `wailsjs` 生成物被另一套 Wails 应用的绑定污染。
- 对生成物类问题，先比对 `src/App.vue` 导入、`wailsjs/go/main/App.d.ts` 导出、`windows/app.go` 的实际方法，再决定要不要改业务代码。

## 可复用排查线索
- 症状：
  - `TS2305` missing export `BootstrapGet`
  - `TS2305` missing export `MetricsSettingsGet`
  - `TS2305` missing export `StartReporting`
- 触发条件：
  - 本地 worktree 或 control-plane working copy 的 `windows/frontend/wailsjs/**` 被错误覆盖
  - 未按仓库脚本清理并重新生成 Wails bindings
- 关键词 / 错误文本：
  - `BootstrapGet`
  - `MetricsSettingsGet`
  - `AboutState`
  - `FlowProjectsState`
  - `SaveHomeState`
  - `wailsjs`
  - `App.d.ts`
- 快速检查：
  - 打开 `windows/frontend/wailsjs/go/main/App.d.ts`
  - 确认是否包含 `BootstrapGet` / `Status` / `MetricsSettingsGet`
  - 若看到 `AboutState` / `FlowProjectsState` / `SaveHomeState`，直接运行 `powershell -ExecutionPolicy Bypass -File .\scripts\build-windows.ps1`

## 关键设计决策与权衡
- 选择在构建脚本中 fail fast 校验绑定面，而不是手改生成文件。
- 保持业务代码不动，因为 clean worktree 已证明 MetricsNode 当前 frontend / Go 接口契约本身没有问题。
- 用 lesson 承载可复用排查知识，避免把重复性调试经验只留在 dated change 文档中。

## 测试与验证方式 / 结果
- `cd windows/frontend && npm ci`：通过
- `cd windows/frontend && npm run build`：通过
- `powershell -ExecutionPolicy Bypass -File scripts/build-windows.ps1`：通过
  - 包含清理 bindings、重新 `wails generate module`、绑定面校验、`wails build`
- 生成产物：
  - `windows/build/bin/windows.exe`

## 潜在影响
- `build-windows.ps1` 现在会在绑定面异常时更早失败，报错更明确。
- 对正常工作流无运行时影响。

## 回滚方案
- 回滚 `scripts/build-windows.ps1` 的绑定面校验逻辑。
- 回滚 `windows/README.md` 与本次新增 docs 索引 / lesson / change 文档。

## 子Agent执行轨迹
- 未使用子 Agent。
