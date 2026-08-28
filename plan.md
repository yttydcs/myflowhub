# Plan - MyFlowHub canonical local closeout

## Workflow Information

- Repository: `D:/project/MyFlowHub3/repo/MyFlowHub`
- Branch: `master`
- Docs Root: `D:/project/MyFlowHub3/repo/MyFlowHub/docs`
- Control Plane: canonical checkout；没有活动 feature worktree
- Current Stage: `4.0 archive and local closeout complete`
- Compatibility: clean break；不保留旧 module path、SubProto API 或 legacy wire bridge

## Current Status

- FM00–FM16 已完成，完整执行计划归档于 [vNext 全量迁移与最终切换](docs/plan/plan_archive_2026-08-27_vnext-full-migration.md)。
- 10 个本地旧仓已经退役并移除；来源、commit、tree、清单与恢复信息保留在 [migration/README.md](migration/README.md)。
- canonical source、文档、构建入口和本地开发入口统一位于当前仓库。
- Hub、Desktop、Metrics、Clipboard、Android、Embedded 和 SDK 的迁移状态与验证证据见 [vNext 全量迁移归档](docs/change/2026-08-27_vnext-full-migration.md)。
- 根级 `run-dev.ps1` 已修复 Desktop/Metrics Wails bindings 并发生成竞态；本次最终收口见 [Wails 开发启动并发修复与重构收口](docs/change/2026-08-28_wails-dev-concurrency-and-refactor-closeout.md)。

## Closeout Tasks

- [x] CL01 — 串行生成 Desktop 与 Metrics Wails bindings，dev 进程禁止并发改写 bindings 和根 `go.mod`。
- [x] CL02 — 增加架构回归守卫，并更新跨产品 Wails 排障 lesson。
- [x] CL03 — 运行完整 Go、vet、Desktop/Metrics 前端与真实三进程启动验证。
- [x] CL04 — 归档旧 Clipboard 根计划并恢复 canonical 根状态入口。
- [x] CL05 — 核对 control-plane Git 状态；无待合并分支或待清理 workflow worktree。

## Stable Docs Impact

- Intake impact: none
- Feature impact: none
- Requirements impact: none
- Specs impact: none
- Decision impact: none
- Lessons impact: updated
- Related specs: [Build and CI contract](docs/specs/build-and-ci.md)、[Repository and module boundaries](docs/specs/repository-and-module-boundaries.md)
- Related lesson: [Wails Bindings Across Product Facades](docs/lessons/wails-bindings-cross-project.md)

## Deferred Or Separately Authorized Work

- DX01 — 新增 serial/USB/WebSocket Transport；属于新功能。
- DX02 — legacy compatibility bridge；已明确拒绝。
- DX03 — remote push、release、sign、publish；需要单独授权。
- DX04b — 处理与本 workflow 无关的主 checkout 未提交内容；不自动纳入收口。
- DX05 — 真实硬件、签名平台和商店认证；依赖外部设备、主机或凭据。

## Validation Baseline

- `go test ./... -count=1`
- `go vet ./...`
- Desktop：Vitest、TypeScript、Vite production build
- Metrics Windows：Vitest、TypeScript、Vite production build
- `scripts/run-dev.ps1`：Hub、Desktop、Metrics 同时启动，两个 Wails 窗口响应且三个 dev/listen 端口就绪
- `git diff --check`

## Gate

- Blocked: no
- Active migration workflow: none
- Local closeout: complete
- Remote publication: not authorized and not performed
